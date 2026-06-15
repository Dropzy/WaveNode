package com.musicserver.player

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.os.Binder
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat
import androidx.media3.common.MediaItem
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@UnstableApi
@AndroidEntryPoint
class MusicService : MediaSessionService() {
    
    @Inject
    lateinit var exoPlayer: ExoPlayer
    
    @Inject
    lateinit var authenticatedMediaDataSourceFactory: AuthenticatedMediaDataSourceFactory
    
    private lateinit var mediaSession: MediaSession
    private val binder = MusicBinder()
    
    private val _currentTrack = MutableStateFlow<MediaItem?>(null)
    val currentTrack: StateFlow<MediaItem?> = _currentTrack
    
    private val _isPlaying = MutableStateFlow(false)
    val isPlaying: StateFlow<Boolean> = _isPlaying
    
    private val _playbackState = MutableStateFlow(Player.STATE_IDLE)
    val playbackState: StateFlow<Int> = _playbackState
    
    private val _currentPosition = MutableStateFlow(0L)
    val currentPosition: StateFlow<Long> = _currentPosition
    
    private val _duration = MutableStateFlow(0L)
    val duration: StateFlow<Long> = _duration
    
    private val serviceScope = CoroutineScope(Dispatchers.Main)
    
    inner class MusicBinder : Binder() {
        fun getService(): MusicService = this@MusicService
    }
    
    override fun onCreate() {
        android.util.Log.d("MusicService", "MusicService onCreate called")
        super.onCreate()
        initializePlayer()
        createNotificationChannel()
        android.util.Log.d("MusicService", "MusicService initialized successfully")
    }
    
    private fun initializePlayer() {
        mediaSession = MediaSession.Builder(this, exoPlayer).build()
        
        exoPlayer.addListener(object : Player.Listener {
            override fun onIsPlayingChanged(isPlaying: Boolean) {
                android.util.Log.d("MusicService", "Is playing changed: $isPlaying")
                _isPlaying.value = isPlaying
                updateNotification()
            }
            
            override fun onPlaybackStateChanged(playbackState: Int) {
                android.util.Log.d("MusicService", "Playback state changed: $playbackState")
                _playbackState.value = playbackState
                
                when (playbackState) {
                    Player.STATE_IDLE -> android.util.Log.d("MusicService", "Player state: IDLE")
                    Player.STATE_BUFFERING -> android.util.Log.d("MusicService", "Player state: BUFFERING")
                    Player.STATE_READY -> android.util.Log.d("MusicService", "Player state: READY")
                    Player.STATE_ENDED -> android.util.Log.d("MusicService", "Player state: ENDED")
                }
            }
            
            override fun onMediaItemTransition(mediaItem: MediaItem?, reason: Int) {
                android.util.Log.d("MusicService", "Media item transition: ${mediaItem?.mediaMetadata?.title}")
                _currentTrack.value = mediaItem
                updateNotification()
            }
            
            override fun onPlayerError(error: androidx.media3.common.PlaybackException) {
                android.util.Log.e("MusicService", "Player error: ${error.message}", error)
            }
        })
        
        // Start progress updates
        startProgressUpdates()
    }
    
    private fun startProgressUpdates() {
        serviceScope.launch {
            while (true) {
                try {
                    if (exoPlayer.playbackState == Player.STATE_READY || exoPlayer.playbackState == Player.STATE_BUFFERING) {
                        _currentPosition.value = exoPlayer.currentPosition
                        _duration.value = exoPlayer.duration
                    }
                    kotlinx.coroutines.delay(100) // Update every 100ms for smooth progress
                } catch (e: Exception) {
                    android.util.Log.e("MusicService", "Error updating progress", e)
                    kotlinx.coroutines.delay(1000) // Wait longer on error
                }
            }
        }
    }
    
    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                MUSIC_CHANNEL_ID,
                "Music Playback",
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply {
                description = "Music playback controls and information"
                setShowBadge(true)
                enableLights(true)
                enableVibration(false)
                setSound(null, null)
            }
            
            val notificationManager = getSystemService(NotificationManager::class.java)
            notificationManager.createNotificationChannel(channel)
        }
    }
    
    private fun updateNotification() {
        val notification = createNotification()
        startForeground(NOTIFICATION_ID, notification)
    }
    
    private fun createNotification(): Notification {
        val currentTrack = _currentTrack.value
        
        // Create content intent for opening the app when notification is tapped
        val contentIntent = PendingIntent.getActivity(
            this, 0, Intent(this, com.musicserver.MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE
        )
        
        val builder = NotificationCompat.Builder(this, MUSIC_CHANNEL_ID)
            .setContentTitle(currentTrack?.mediaMetadata?.title ?: "No track playing")
            .setContentText(currentTrack?.mediaMetadata?.artist ?: "")
            .setSubText(currentTrack?.mediaMetadata?.albumTitle ?: "")
            .setSmallIcon(android.R.drawable.ic_media_play)
            .setContentIntent(contentIntent)
            .setOngoing(true)
            .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)
            .setShowWhen(false)
            .setOnlyAlertOnce(true)
        
        // Add MediaStyle for better media controls
        val mediaStyle = androidx.media.app.NotificationCompat.MediaStyle()
            .setMediaSession(mediaSession.sessionCompatToken)
            .setShowActionsInCompactView(0, 1, 2) // Show play/pause, next, previous in compact view
        
        // Add playback controls
        val actions = mutableListOf<NotificationCompat.Action>()
        
        // Previous button
        actions.add(
            NotificationCompat.Action.Builder(
                android.R.drawable.ic_media_previous,
                "Previous",
                createPlaybackIntent(ACTION_PREVIOUS)
            ).build()
        )
        
        // Play/Pause button
        val playPauseIcon = if (_isPlaying.value) android.R.drawable.ic_media_pause else android.R.drawable.ic_media_play
        val playPauseTitle = if (_isPlaying.value) "Pause" else "Play"
        actions.add(
            NotificationCompat.Action.Builder(
                playPauseIcon,
                playPauseTitle,
                createPlaybackIntent(if (_isPlaying.value) ACTION_PAUSE else ACTION_PLAY)
            ).build()
        )
        
        // Next button
        actions.add(
            NotificationCompat.Action.Builder(
                android.R.drawable.ic_media_next,
                "Next",
                createPlaybackIntent(ACTION_NEXT)
            ).build()
        )
        
        // Stop button (shown in expanded view)
        actions.add(
            NotificationCompat.Action.Builder(
                android.R.drawable.ic_menu_close_clear_cancel,
                "Stop",
                createPlaybackIntent(ACTION_STOP)
            ).build()
        )
        
        // Add actions to builder
        actions.forEach { builder.addAction(it) }
        
        // Set MediaStyle
        builder.setStyle(mediaStyle)
        
        // TODO: Implement album artwork loading when needed
        // For now, skip album artwork to avoid complexity
        
        return builder.build()
    }
    
    private fun createPlaybackIntent(action: String): PendingIntent {
        val intent = Intent(this, MusicService::class.java).apply {
            this.action = action
        }
        return PendingIntent.getService(this, 0, intent, PendingIntent.FLAG_IMMUTABLE)
    }
    
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_PLAY -> play()
            ACTION_PAUSE -> pause()
            ACTION_NEXT -> skipToNext()
            ACTION_PREVIOUS -> skipToPrevious()
            ACTION_STOP -> stopSelf()
        }
        return super.onStartCommand(intent, flags, startId)
    }
    
    override fun onBind(intent: Intent?): IBinder {
        return super.onBind(intent) ?: binder
    }
    
    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession {
        return mediaSession
    }
    
    // Player control methods
    fun playTrack(mediaItem: MediaItem) {
        try {
            android.util.Log.d("MusicService", "Playing track: ${mediaItem.mediaMetadata.title} - ${mediaItem.localConfiguration?.uri}")
            android.util.Log.d("MusicService", "MediaItem metadata - Title: ${mediaItem.mediaMetadata.title}, Artist: ${mediaItem.mediaMetadata.artist}")
            
            // Set current track immediately
            _currentTrack.value = mediaItem
            
            // Start foreground service with notification immediately
            startForegroundService()
            
            // Create a media source with authenticated data source
            val uri = mediaItem.localConfiguration?.uri ?: run {
                android.util.Log.e("MusicService", "URI is null for media item: ${mediaItem.mediaMetadata.title}")
                return
            }
            android.util.Log.d("MusicService", "Creating media source for URI: $uri")
            
            // Clear any existing media items first
            exoPlayer.clearMediaItems()
            
            // Create a new MediaItem with the same metadata but the correct URI for the data source
            val authenticatedMediaItem = MediaItem.Builder()
                .setUri(uri)
                .setMediaId(mediaItem.mediaId)
                .setMediaMetadata(mediaItem.mediaMetadata)
                .build()
            
            val mediaSource = try {
                // Try with authenticated data source first
                val dataSource = authenticatedMediaDataSourceFactory.createDataSource()
                androidx.media3.exoplayer.source.ProgressiveMediaSource.Factory { dataSource }
                    .createMediaSource(authenticatedMediaItem)
            } catch (e: Exception) {
                android.util.Log.w("MusicService", "Authenticated data source failed, trying fallback", e)
                // Fallback to default data source
                val fallbackDataSource = androidx.media3.datasource.DefaultHttpDataSource.Factory().apply {
                    setConnectTimeoutMs(30000)
                    setReadTimeoutMs(30000)
                }.createDataSource()
                androidx.media3.exoplayer.source.ProgressiveMediaSource.Factory { fallbackDataSource }
                    .createMediaSource(authenticatedMediaItem)
            }
            
            android.util.Log.d("MusicService", "Setting media source and preparing player")
            exoPlayer.setMediaSource(mediaSource)
            
            // Add listener to wait for player to be ready
            val playbackListener = object : Player.Listener {
                override fun onPlaybackStateChanged(playbackState: Int) {
                    when (playbackState) {
                        Player.STATE_READY -> {
                            android.util.Log.d("MusicService", "Player is ready, starting playback")
                            exoPlayer.play()
                            exoPlayer.removeListener(this)
                        }
                        Player.STATE_BUFFERING -> {
                            android.util.Log.d("MusicService", "Player is buffering...")
                        }
                        Player.STATE_IDLE -> {
                            android.util.Log.e("MusicService", "Player is in IDLE state after preparation")
                        }
                        Player.STATE_ENDED -> {
                            android.util.Log.d("MusicService", "Playback ended immediately")
                        }
                    }
                }
                
                override fun onPlayerError(error: androidx.media3.common.PlaybackException) {
                    android.util.Log.e("MusicService", "Playback error during preparation: ${error.message}", error)
                    exoPlayer.removeListener(this)
                }
            }
            
            exoPlayer.addListener(playbackListener)
            exoPlayer.prepare()
            
            android.util.Log.d("MusicService", "Player preparation initiated")
        } catch (e: Exception) {
            android.util.Log.e("MusicService", "Error playing track", e)
        }
    }
    
    private fun startForegroundService() {
        try {
            val notification = createNotification()
            startForeground(NOTIFICATION_ID, notification)
            android.util.Log.d("MusicService", "Foreground service started with notification")
        } catch (e: Exception) {
            android.util.Log.e("MusicService", "Error starting foreground service", e)
        }
    }
    
    private fun createAuthenticatedMediaItem(mediaItem: MediaItem): MediaItem {
        val uri = mediaItem.localConfiguration?.uri ?: return mediaItem
        
        android.util.Log.d("MusicService", "Creating authenticated media item for URI: $uri")
        
        return MediaItem.Builder()
            .setUri(uri)
            .setMediaMetadata(mediaItem.mediaMetadata)
            .setMediaId(mediaItem.mediaId)
            .build()
    }
    
    fun play() {
        if (exoPlayer.playbackState == Player.STATE_READY) {
            exoPlayer.play()
        } else {
            android.util.Log.w("MusicService", "Cannot play - player is not ready. State: ${exoPlayer.playbackState}")
        }
    }
    
    fun pause() {
        exoPlayer.pause()
    }
    
    fun stop() {
        exoPlayer.stop()
        stopForeground(STOP_FOREGROUND_REMOVE)
    }
    
    fun skipToNext() {
        exoPlayer.seekToNext()
    }
    
    fun skipToPrevious() {
        exoPlayer.seekToPrevious()
    }
    
    fun seekTo(position: Long) {
        exoPlayer.seekTo(position)
    }
    
    fun getCurrentPosition(): Long {
        return exoPlayer.currentPosition
    }
    
    fun getDuration(): Long {
        return exoPlayer.duration
    }
    
    fun setPlaylist(mediaItems: List<MediaItem>, startIndex: Int = 0) {
        try {
            // Convert media items to authenticated media sources
            val mediaSources = mediaItems.map { mediaItem ->
                val uri = mediaItem.localConfiguration?.uri ?: return@map null
                val dataSource = authenticatedMediaDataSourceFactory.createDataSource()
                androidx.media3.exoplayer.source.ProgressiveMediaSource.Factory { dataSource }
                    .createMediaSource(mediaItem)
            }.filterNotNull()
            
            exoPlayer.setMediaSources(mediaSources, startIndex, 0L)
            exoPlayer.prepare()
            exoPlayer.play()
        } catch (e: Exception) {
            android.util.Log.e("MusicService", "Error setting playlist", e)
        }
    }
    
    override fun onDestroy() {
        mediaSession.release()
        exoPlayer.release()
        super.onDestroy()
    }
    
    companion object {
        private const val MUSIC_CHANNEL_ID = "music_channel"
        private const val NOTIFICATION_ID = 1
        
        const val ACTION_PLAY = "com.musicserver.ACTION_PLAY"
        const val ACTION_PAUSE = "com.musicserver.ACTION_PAUSE"
        const val ACTION_NEXT = "com.musicserver.ACTION_NEXT"
        const val ACTION_PREVIOUS = "com.musicserver.ACTION_PREVIOUS"
        const val ACTION_STOP = "com.musicserver.ACTION_STOP"
    }
}
