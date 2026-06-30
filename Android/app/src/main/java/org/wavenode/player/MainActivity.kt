package org.wavenode.player

import android.Manifest
import android.os.Bundle
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import org.wavenode.player.ui.WaveNodeApp
import org.wavenode.player.ui.WaveNodeTheme

class MainActivity : AppCompatActivity() {
    private val viewModel: WaveNodeViewModel by viewModels()

    override fun onStart() {
        super.onStart()
        viewModel.setAppVisible(true)
    }

    override fun onStop() {
        viewModel.setAppVisible(false)
        super.onStop()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            val state by viewModel.state.collectAsState()
            val playerState by viewModel.playerState.collectAsState()
            val permissionLauncher = rememberLauncherForActivityResult(
                ActivityResultContracts.RequestMultiplePermissions(),
            ) {}

            LaunchedEffect(Unit) {
                val permissions = buildList {
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                        add(Manifest.permission.BLUETOOTH_CONNECT)
                    }
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                        add(Manifest.permission.POST_NOTIFICATIONS)
                    }
                }
                if (permissions.isNotEmpty()) {
                    permissionLauncher.launch(permissions.toTypedArray())
                }
            }

            WaveNodeTheme {
                WaveNodeApp(
                    state = state,
                    playerState = playerState,
                    onLogin = viewModel::login,
                    onDiscoverServers = viewModel::discoverServers,
                    onRefresh = viewModel::refreshTracks,
                    onLogout = viewModel::logout,
                    onPlayFromHere = viewModel::playFromHere,
                    onPlayQueueTrack = viewModel::playQueueTrack,
                    onOpenAlbum = viewModel::openAlbum,
                    onOpenArtist = viewModel::openArtist,
                    onOpenPlaylist = viewModel::openPlaylist,
                    onPodcastQueryChange = viewModel::updatePodcastSearch,
                    onOpenPodcast = viewModel::openPodcast,
                    onResumePodcast = viewModel::resumePodcast,
					onAudiobookQueryChange = viewModel::updateAudiobookSearch,
					onOpenAudiobook = viewModel::openAudiobook,
					onResumeAudiobook = viewModel::resumeAudiobook,
					onTogglePodcastSubscription = viewModel::togglePodcastSubscription,
					onUpdatePodcastAutoDownload = viewModel::updatePodcastAutoDownload,
					onDownloadPodcastEpisode = viewModel::downloadPodcastEpisode,
					onDeletePodcastDownload = viewModel::deletePodcastDownload,
                    onCloseDetail = viewModel::closeDetail,
                    onTogglePlayPause = viewModel::togglePlayPause,
                    onToggleShuffle = viewModel::toggleShuffle,
                    onCycleRepeatMode = viewModel::cycleRepeatMode,
                    onSkipNext = viewModel::skipNext,
                    onSkipPrevious = viewModel::skipPrevious,
                    onSeekTo = viewModel::seekTo,
					onLoadLyrics = viewModel::loadLyrics,
					onSkipPodcastBack = viewModel::skipPodcastBack,
					onSkipPodcastForward = viewModel::skipPodcastForward,
					onSetPodcastPlaybackSpeed = viewModel::setPodcastPlaybackSpeed,
					onSetPodcastSleepTimer = viewModel::setPodcastSleepTimer,
					onSetPodcastSleepTimerToEpisodeEnd = viewModel::setPodcastSleepTimerToEpisodeEnd,
                    onRefreshConnectSessions = viewModel::refreshConnectSessions,
                    onConnectPlaybackTo = viewModel::connectPlaybackTo,
					onPrepareGoogleCast = viewModel::prepareGoogleCast,
					onPlayOnOutputDevice = viewModel::playOnOutputDevice,
                    onAddTracksToPlaylist = viewModel::addTracksToPlaylist,
                    onCreatePlaylist = viewModel::createPlaylist,
                    onCreatePlaylistWithTracks = viewModel::createPlaylistWithTracks,
                    onUpdatePlaylist = viewModel::updatePlaylist,
                    onDeletePlaylist = viewModel::deletePlaylist,
                    trackArtworkUrl = viewModel::artworkUrl,
                    albumArtworkUrl = viewModel::artworkUrl,
                    artistArtworkUrl = viewModel::artworkUrl,
                )
            }
        }
    }
}
