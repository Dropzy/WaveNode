package org.wavenode.player.playback

import android.content.Context
import android.content.Intent
import android.os.SystemClock
import androidx.core.content.ContextCompat
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.common.Player
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.session.MediaSession
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import org.wavenode.player.data.SavedSession
import org.wavenode.player.data.Track
import org.wavenode.player.data.WaveNodeApi

data class PlayerState(
    val currentTrack: Track? = null,
    val isPlaying: Boolean = false,
    val queue: List<Track> = emptyList(),
    val currentIndex: Int = -1,
    val positionMs: Long = 0L,
    val durationMs: Long = 0L,
    val isShuffleEnabled: Boolean = false,
    val repeatMode: WaveRepeatMode = WaveRepeatMode.Off,
	val playbackSpeed: Float = 1f,
)

enum class WaveRepeatMode {
    Off,
    All,
    One,
}

class WaveNodePlayer private constructor(
    private val context: Context,
    private val api: WaveNodeApi,
) {
    private val httpDataSourceFactory = DefaultHttpDataSource.Factory()
    private val exoPlayer = ExoPlayer.Builder(context)
        .setMediaSourceFactory(DefaultMediaSourceFactory(httpDataSourceFactory))
        .setHandleAudioBecomingNoisy(true)
        .setWakeMode(C.WAKE_MODE_NETWORK)
        .build()
    private val mediaSession = MediaSession.Builder(context, exoPlayer)
        .setId("wavenode_player")
        .build()

    private val _state = MutableStateFlow(PlayerState())
    val state: StateFlow<PlayerState> = _state
    private var currentSession: SavedSession? = null
    private var appVisible = false
    private var remoteControlled = false
    private var remoteProgressUpdatedAtMs = 0L
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)

    init {
        exoPlayer.setAudioAttributes(
            AudioAttributes.Builder()
                .setUsage(C.USAGE_MEDIA)
                .setContentType(C.AUDIO_CONTENT_TYPE_MUSIC)
                .build(),
            true,
        )
        exoPlayer.addListener(object : Player.Listener {
            override fun onIsPlayingChanged(isPlaying: Boolean) {
                _state.value = _state.value.copy(isPlaying = isPlaying)
            }

            override fun onPlaybackStateChanged(playbackState: Int) {
                _state.value = _state.value.copy(isPlaying = exoPlayer.isPlaying)
            }

            override fun onMediaItemTransition(mediaItem: MediaItem?, reason: Int) {
                updateCurrentTrackFromPlayer()
            }

            override fun onShuffleModeEnabledChanged(shuffleModeEnabled: Boolean) {
                _state.value = _state.value.copy(isShuffleEnabled = shuffleModeEnabled)
            }

            override fun onRepeatModeChanged(repeatMode: Int) {
                _state.value = _state.value.copy(repeatMode = repeatMode.toWaveRepeatMode())
            }
        })
        scope.launch {
            while (isActive) {
                val remotePlaying = remoteControlled && _state.value.isPlaying
                if (appVisible || exoPlayer.isPlaying || remotePlaying) {
                    updatePlaybackProgress()
                }
                delay(if (exoPlayer.isPlaying || remotePlaying) 1_000 else 2_500)
            }
        }
    }

    fun play(session: SavedSession, track: Track) {
        playQueue(session, listOf(track), 0)
    }

    fun resumeLocalPlayback(session: SavedSession): Boolean {
        updateRemotePlaybackProgress()
        val state = _state.value
        val playableQueue = state.queue.ifEmpty { state.currentTrack?.let(::listOf).orEmpty() }
            .filter { it.id.isNotBlank() }
        if (playableQueue.isEmpty()) {
            return false
        }
        val currentTrackId = state.currentTrack?.id
        val fallbackIndex = playableQueue.indexOfFirst { it.id == currentTrackId }.takeIf { it >= 0 } ?: 0
        val safeIndex = state.currentIndex.takeIf { it in playableQueue.indices } ?: fallbackIndex
        val safeDurationMs = state.durationMs.takeIf { it > 0L }
            ?: playableQueue[safeIndex].duration.toLong().coerceAtLeast(0L) * 1000L
        val safePositionMs = if (safeDurationMs > 0L) {
            state.positionMs.coerceIn(0L, (safeDurationMs - 500L).coerceAtLeast(0L))
        } else {
            state.positionMs.coerceAtLeast(0L)
        }
        playQueue(session, playableQueue, safeIndex, safePositionMs)
        return true
    }

    fun playQueue(session: SavedSession, tracks: List<Track>, startIndex: Int, startPositionMs: Long = 0L) {
        val playableTracks = tracks.filter { it.id.isNotBlank() }
        if (playableTracks.isEmpty()) {
            return
        }
        val safeIndex = startIndex.coerceIn(0, playableTracks.lastIndex)
        val track = playableTracks[safeIndex]
        val previousState = _state.value
        val isPodcastQueue = playableTracks.any { it.externalKind == "podcast" }
        val shuffleEnabled = if (isPodcastQueue) false else previousState.isShuffleEnabled
        val repeatMode = if (isPodcastQueue) WaveRepeatMode.Off else previousState.repeatMode
        remoteControlled = false
        remoteProgressUpdatedAtMs = 0L
        currentSession = session
        ContextCompat.startForegroundService(context, Intent(context, WaveNodePlaybackService::class.java))
        httpDataSourceFactory.setDefaultRequestProperties(api.playbackHeaders(session, track))
        exoPlayer.shuffleModeEnabled = shuffleEnabled
        exoPlayer.repeatMode = repeatMode.toPlayerRepeatMode()
        exoPlayer.stop()
        exoPlayer.clearMediaItems()
        _state.value = PlayerState(
            currentTrack = track,
            isPlaying = true,
            queue = playableTracks,
            currentIndex = safeIndex,
            positionMs = startPositionMs.coerceAtLeast(0L),
            isShuffleEnabled = shuffleEnabled,
            repeatMode = repeatMode,
			playbackSpeed = if (isPodcastQueue) previousState.playbackSpeed else 1f,
        )
        exoPlayer.setMediaItems(playableTracks.map { mediaItem(session, it) }, safeIndex, startPositionMs.coerceAtLeast(0L))
        exoPlayer.prepare()
		exoPlayer.setPlaybackSpeed(_state.value.playbackSpeed)
        exoPlayer.play()
    }

    fun skipNext() {
        if (exoPlayer.hasNextMediaItem()) {
            exoPlayer.seekToNextMediaItem()
            updateCurrentTrackFromPlayer()
        }
    }

    fun skipPrevious() {
        if (exoPlayer.hasPreviousMediaItem()) {
            exoPlayer.seekToPreviousMediaItem()
            updateCurrentTrackFromPlayer()
        } else {
            exoPlayer.seekTo(0)
        }
    }

    fun togglePlayPause() {
        if (exoPlayer.isPlaying) {
            exoPlayer.pause()
        } else {
            exoPlayer.play()
        }
    }

    fun pause() {
        exoPlayer.pause()
        _state.value = _state.value.copy(isPlaying = false)
    }

    fun setRemoteQueue(tracks: List<Track>, startIndex: Int, isPlaying: Boolean, positionMs: Long = 0L) {
        val playableTracks = tracks.filter { it.id.isNotBlank() }
        if (playableTracks.isEmpty()) {
            return
        }
        exoPlayer.pause()
        exoPlayer.clearMediaItems()
        remoteControlled = true
        remoteProgressUpdatedAtMs = SystemClock.elapsedRealtime()
        val safeIndex = startIndex.coerceIn(0, playableTracks.lastIndex)
        _state.value = _state.value.copy(
            currentTrack = playableTracks[safeIndex],
            isPlaying = isPlaying,
            queue = playableTracks,
            currentIndex = safeIndex,
            positionMs = positionMs.coerceAtLeast(0L),
            durationMs = playableTracks[safeIndex].duration.toLong().coerceAtLeast(0L) * 1000L,
        )
    }

    fun setRemotePlaying(isPlaying: Boolean) {
        if (remoteControlled) {
            updateRemotePlaybackProgress()
            remoteProgressUpdatedAtMs = SystemClock.elapsedRealtime()
        }
        _state.value = _state.value.copy(isPlaying = isPlaying)
    }

    fun setRemotePosition(positionMs: Long) {
        if (!remoteControlled) {
            return
        }
        remoteProgressUpdatedAtMs = SystemClock.elapsedRealtime()
        _state.value = _state.value.copy(positionMs = positionMs.coerceAtLeast(0L))
    }

    fun toggleShuffle() {
        exoPlayer.shuffleModeEnabled = !exoPlayer.shuffleModeEnabled
        _state.value = _state.value.copy(isShuffleEnabled = exoPlayer.shuffleModeEnabled)
    }

    fun cycleRepeatMode() {
        val nextMode = when (_state.value.repeatMode) {
            WaveRepeatMode.Off -> WaveRepeatMode.All
            WaveRepeatMode.All -> WaveRepeatMode.One
            WaveRepeatMode.One -> WaveRepeatMode.Off
        }
        exoPlayer.repeatMode = nextMode.toPlayerRepeatMode()
        _state.value = _state.value.copy(repeatMode = nextMode)
    }

    fun seekTo(positionMs: Long) {
        exoPlayer.seekTo(positionMs.coerceAtLeast(0L))
        updatePlaybackProgress()
    }

	fun seekBy(offsetMs: Long) {
		val duration = normalizedDuration()
		val maximum = if (duration > 0L) duration else Long.MAX_VALUE
		seekTo((exoPlayer.currentPosition + offsetMs).coerceIn(0L, maximum))
	}

	fun setPlaybackSpeed(speed: Float) {
		val safeSpeed = speed.coerceIn(0.5f, 3f)
		exoPlayer.setPlaybackSpeed(safeSpeed)
		_state.value = _state.value.copy(playbackSpeed = safeSpeed)
	}

    fun setAppVisible(isVisible: Boolean) {
        appVisible = isVisible
        if (isVisible) {
            updatePlaybackProgress()
        }
    }

    fun updateCurrentTrack(track: Track) {
        val session = currentSession ?: return
        val state = _state.value
        val index = state.currentIndex
        if (index !in state.queue.indices || state.queue[index].id != track.id) {
            return
        }
        val updatedQueue = state.queue.toMutableList().also { it[index] = track }
        _state.value = state.copy(currentTrack = track, queue = updatedQueue)
        exoPlayer.replaceMediaItem(index, mediaItem(session, track))
    }

    fun release() {
        mediaSession.release()
        exoPlayer.release()
    }

    fun mediaSession(): MediaSession = mediaSession

    fun player(): Player = exoPlayer

    private fun mediaItem(session: SavedSession, track: Track): MediaItem {
        return MediaItem.Builder()
            .setMediaId(track.id)
            .setUri(api.streamUrl(session, track))
            .setMediaMetadata(
                MediaMetadata.Builder()
                    .setTitle(track.title)
                    .setArtist(track.artist)
                    .setAlbumTitle(track.album)
                    .build(),
            )
            .build()
    }

    private fun updateCurrentTrackFromPlayer() {
        val queue = _state.value.queue
        val index = exoPlayer.currentMediaItemIndex
        val current = queue.getOrNull(index)
        _state.value = _state.value.copy(
            currentTrack = current,
            currentIndex = if (current == null) -1 else index,
            isPlaying = exoPlayer.isPlaying,
            positionMs = exoPlayer.currentPosition.coerceAtLeast(0L),
            durationMs = normalizedDuration(),
        )
    }

    private fun updatePlaybackProgress() {
        if (remoteControlled) {
            updateRemotePlaybackProgress()
            return
        }
        if (_state.value.currentTrack == null) {
            return
        }
        _state.value = _state.value.copy(
            positionMs = exoPlayer.currentPosition.coerceAtLeast(0L),
            durationMs = normalizedDuration(),
            isPlaying = exoPlayer.isPlaying,
        )
    }

    private fun updateRemotePlaybackProgress() {
        val state = _state.value
        if (state.currentTrack == null) {
            return
        }
        val now = SystemClock.elapsedRealtime()
        val elapsed = if (state.isPlaying && remoteProgressUpdatedAtMs > 0L) {
            now - remoteProgressUpdatedAtMs
        } else {
            0L
        }
        val nextPosition = state.positionMs + elapsed
        val clampedPosition = if (state.durationMs > 0L) {
            nextPosition.coerceAtMost(state.durationMs)
        } else {
            nextPosition
        }
        remoteProgressUpdatedAtMs = now
        _state.value = state.copy(positionMs = clampedPosition)
    }

    private fun normalizedDuration(): Long {
        val duration = exoPlayer.duration
        return if (duration > 0 && duration != C.TIME_UNSET) duration else 0L
    }

    private fun WaveRepeatMode.toPlayerRepeatMode(): Int {
        return when (this) {
            WaveRepeatMode.Off -> Player.REPEAT_MODE_OFF
            WaveRepeatMode.All -> Player.REPEAT_MODE_ALL
            WaveRepeatMode.One -> Player.REPEAT_MODE_ONE
        }
    }

    private fun Int.toWaveRepeatMode(): WaveRepeatMode {
        return when (this) {
            Player.REPEAT_MODE_ALL -> WaveRepeatMode.All
            Player.REPEAT_MODE_ONE -> WaveRepeatMode.One
            else -> WaveRepeatMode.Off
        }
    }

    companion object {
        @Volatile
        private var instance: WaveNodePlayer? = null

        fun get(context: Context, api: WaveNodeApi): WaveNodePlayer {
            return instance ?: synchronized(this) {
                instance ?: WaveNodePlayer(context.applicationContext, api).also { instance = it }
            }
        }

        fun currentMediaSession(): MediaSession? = instance?.mediaSession()

        fun releaseCurrent() {
            synchronized(this) {
                instance?.release()
                instance = null
            }
        }
    }
}
