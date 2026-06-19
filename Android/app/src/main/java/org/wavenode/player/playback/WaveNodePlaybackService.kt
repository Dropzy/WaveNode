package org.wavenode.player.playback

import android.app.Notification
import android.app.PendingIntent
import android.content.Intent
import android.graphics.Bitmap
import androidx.core.app.NotificationManagerCompat
import androidx.media3.common.Player
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService
import androidx.media3.ui.PlayerNotificationManager
import org.wavenode.player.MainActivity
import org.wavenode.player.R
import org.wavenode.player.data.WaveNodeApi

class WaveNodePlaybackService : MediaSessionService() {
    private var notificationManager: PlayerNotificationManager? = null
    private var isForeground = false

    override fun onCreate() {
        super.onCreate()
        val wavePlayer = WaveNodePlayer.get(this, WaveNodeApi())

        notificationManager = PlayerNotificationManager.Builder(
            this,
            PLAYBACK_NOTIFICATION_ID,
            PLAYBACK_CHANNEL_ID,
        )
            .setChannelNameResourceId(R.string.playback_channel_name)
            .setChannelDescriptionResourceId(R.string.playback_channel_description)
            .setSmallIconResourceId(R.drawable.ic_stat_wavenode)
            .setMediaDescriptionAdapter(WaveNodeDescriptionAdapter())
            .setNotificationListener(WaveNodeNotificationListener())
            .build()
            .apply {
                setMediaSessionToken(wavePlayer.mediaSession().platformToken)
                setUsePreviousAction(false)
                setUseNextAction(false)
                setUseFastForwardAction(false)
                setUseRewindAction(false)
                setUseStopAction(true)
                setPlayer(wavePlayer.player())
            }
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? {
        return WaveNodePlayer.currentMediaSession()
    }

    override fun onDestroy() {
        notificationManager?.setPlayer(null)
        notificationManager = null
        WaveNodePlayer.releaseCurrent()
        super.onDestroy()
    }

    private inner class WaveNodeDescriptionAdapter : PlayerNotificationManager.MediaDescriptionAdapter {
        override fun getCurrentContentTitle(player: Player): CharSequence {
            return player.mediaMetadata.title ?: "WaveNode"
        }

        override fun createCurrentContentIntent(player: Player): PendingIntent {
            val intent = Intent(this@WaveNodePlaybackService, MainActivity::class.java)
            return PendingIntent.getActivity(
                this@WaveNodePlaybackService,
                0,
                intent,
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
            )
        }

        override fun getCurrentContentText(player: Player): CharSequence {
            return player.mediaMetadata.artist ?: player.mediaMetadata.albumTitle ?: "Playing from WaveNode"
        }

        override fun getCurrentLargeIcon(
            player: Player,
            callback: PlayerNotificationManager.BitmapCallback,
        ): Bitmap? = null
    }

    private inner class WaveNodeNotificationListener : PlayerNotificationManager.NotificationListener {
        override fun onNotificationPosted(notificationId: Int, notification: Notification, ongoing: Boolean) {
            if (ongoing) {
                startForeground(notificationId, notification)
                isForeground = true
            } else if (isForeground) {
                stopForeground(STOP_FOREGROUND_DETACH)
                NotificationManagerCompat.from(this@WaveNodePlaybackService).notify(notificationId, notification)
                isForeground = false
            }
        }

        override fun onNotificationCancelled(notificationId: Int, dismissedByUser: Boolean) {
            stopForeground(STOP_FOREGROUND_REMOVE)
            isForeground = false
            stopSelf()
        }
    }

    private companion object {
        const val PLAYBACK_NOTIFICATION_ID = 1001
        const val PLAYBACK_CHANNEL_ID = "wavenode_playback"
    }
}
