package org.wavenode.player.playback

import android.content.Context
import com.google.android.gms.cast.MediaInfo
import com.google.android.gms.cast.MediaLoadRequestData
import com.google.android.gms.cast.MediaMetadata
import com.google.android.gms.cast.framework.CastContext
import com.google.android.gms.cast.framework.CastSession
import com.google.android.gms.cast.framework.SessionManagerListener
import org.wavenode.player.data.Track

class WaveNodeCastController private constructor(context: Context) {
    private val castContext = CastContext.getSharedInstance(context.applicationContext)
    private var pending: PendingMedia? = null

    init {
        castContext.sessionManager.addSessionManagerListener(object : SessionManagerListener<CastSession> {
            override fun onSessionStarted(session: CastSession, sessionId: String) = loadPending(session)
            override fun onSessionResumed(session: CastSession, wasSuspended: Boolean) = loadPending(session)
            override fun onSessionStarting(session: CastSession) = Unit
            override fun onSessionStartFailed(session: CastSession, error: Int) = Unit
            override fun onSessionEnding(session: CastSession) = Unit
            override fun onSessionEnded(session: CastSession, error: Int) = Unit
            override fun onSessionResuming(session: CastSession, sessionId: String) = Unit
            override fun onSessionResumeFailed(session: CastSession, error: Int) = Unit
            override fun onSessionSuspended(session: CastSession, reason: Int) = Unit
        }, CastSession::class.java)
    }

    fun prepare(track: Track, mediaUrl: String, positionMs: Long, onLoaded: () -> Unit) {
        pending = PendingMedia(track, mediaUrl, positionMs, onLoaded)
        castContext.sessionManager.currentCastSession?.let(::loadPending)
    }

    private fun loadPending(session: CastSession) {
        val item = pending ?: return
        val metadata = MediaMetadata(MediaMetadata.MEDIA_TYPE_MUSIC_TRACK).apply {
            putString(MediaMetadata.KEY_TITLE, item.track.title)
            putString(MediaMetadata.KEY_ARTIST, item.track.artist)
            putString(MediaMetadata.KEY_ALBUM_TITLE, item.track.album)
        }
        val mediaInfo = MediaInfo.Builder(item.url)
            .setContentType("audio/mpeg")
            .setStreamType(MediaInfo.STREAM_TYPE_BUFFERED)
            .setMetadata(metadata)
            .build()
		val remoteMediaClient = session.remoteMediaClient ?: return
		remoteMediaClient.load(
            MediaLoadRequestData.Builder()
                .setMediaInfo(mediaInfo)
                .setAutoplay(true)
                .setCurrentTime(item.positionMs.coerceAtLeast(0L))
                .build(),
		).setResultCallback { result ->
			if (result.status.isSuccess && pending == item) {
				item.onLoaded()
				pending = null
			}
		}
    }

	private data class PendingMedia(val track: Track, val url: String, val positionMs: Long, val onLoaded: () -> Unit)

    companion object {
        @Volatile private var instance: WaveNodeCastController? = null
        fun get(context: Context): WaveNodeCastController = instance ?: synchronized(this) {
            instance ?: WaveNodeCastController(context).also { instance = it }
        }
    }
}
