package org.wavenode.player

import android.app.Application
import android.util.Log
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import org.wavenode.player.data.Album
import org.wavenode.player.data.Artist
import org.wavenode.player.data.Audiobook
import org.wavenode.player.data.AudiobookChapter
import org.wavenode.player.data.AudiobookDetail
import org.wavenode.player.data.AudiobookHome
import org.wavenode.player.data.AudiobookProgress
import org.wavenode.player.data.DiscoveredServer
import org.wavenode.player.data.Playlist
import org.wavenode.player.data.OutputDevice
import org.wavenode.player.data.Lyrics
import org.wavenode.player.data.Podcast
import org.wavenode.player.data.PodcastEpisode
import org.wavenode.player.data.PodcastDownloadStore
import org.wavenode.player.data.PodcastHomeResponse
import org.wavenode.player.data.PodcastPreferences
import org.wavenode.player.data.PodcastProgress
import org.wavenode.player.data.PodcastSubscription
import org.wavenode.player.data.PluginHomeRow
import org.wavenode.player.data.SavedSession
import org.wavenode.player.data.ServerDiscovery
import org.wavenode.player.data.SessionStore
import org.wavenode.player.data.Track
import org.wavenode.player.data.UserSession
import org.wavenode.player.data.WaveNodeApi
import org.wavenode.player.playback.PlayerState
import org.wavenode.player.playback.WaveNodePlayer
import org.wavenode.player.playback.WaveNodeCastController
import java.time.Duration
import java.time.Instant

data class AppState(
    val session: SavedSession? = null,
    val tracks: List<Track> = emptyList(),
    val albums: List<Album> = emptyList(),
    val artists: List<Artist> = emptyList(),
    val playlists: List<Playlist> = emptyList(),
    val pluginRows: List<PluginHomeRow> = emptyList(),
    val podcastQuery: String = "",
    val podcasts: List<Podcast> = emptyList(),
    val isLoadingPodcasts: Boolean = false,
    val podcastError: String? = null,
    val podcastHome: PodcastHomeResponse = PodcastHomeResponse(),
    val isLoadingPodcastHome: Boolean = false,
    val podcastHomeError: String? = null,
	val audiobookQuery: String = "",
	val audiobooks: List<Audiobook> = emptyList(),
	val audiobookHome: AudiobookHome = AudiobookHome(),
	val isLoadingAudiobooks: Boolean = false,
	val audiobookError: String? = null,
	val podcastPreferences: PodcastPreferences = PodcastPreferences(),
	val downloadedPodcastEpisodeIds: Set<String> = emptySet(),
	val downloadingPodcastEpisodeIds: Set<String> = emptySet(),
	val sleepTimerRemainingSeconds: Int = 0,
    val discoveredServers: List<DiscoveredServer> = emptyList(),
    val connectSessions: List<UserSession> = emptyList(),
    val currentSessionId: String = "",
    val connectedPlaybackSessionId: String = "",
    val connectedPlaybackDeviceName: String = "",
    val isLoadingConnectSessions: Boolean = false,
    val connectMessage: String? = null,
	val outputDevices: List<OutputDevice> = emptyList(),
	val isLoadingOutputDevices: Boolean = false,
    val lyrics: Lyrics? = null,
    val lyricsTrackId: String = "",
    val isLoadingLyrics: Boolean = false,
    val lyricsError: String? = null,
    val activeDetail: LibraryDetail? = null,
    val isDetailLoading: Boolean = false,
    val detailError: String? = null,
    val isDiscoveringServers: Boolean = false,
    val isLoading: Boolean = false,
    val error: String? = null,
)

sealed interface LibraryDetail {
    data class AlbumPage(val album: Album, val tracks: List<Track>) : LibraryDetail
    data class ArtistPage(val artist: Artist, val tracks: List<Track>, val albums: List<Album>) : LibraryDetail
    data class PlaylistPage(val playlist: Playlist, val tracks: List<Track>) : LibraryDetail
    data class PodcastPage(val podcast: Podcast, val episodes: List<PodcastEpisode>) : LibraryDetail
	data class AudiobookPage(val detail: AudiobookDetail) : LibraryDetail
}

class WaveNodeViewModel(application: Application) : AndroidViewModel(application) {
    private val api = WaveNodeApi()
    private val serverDiscovery = ServerDiscovery()
    private val sessionStore = SessionStore(application)
    private val player = WaveNodePlayer.get(application, api)
    private val podcastDownloads = PodcastDownloadStore(application)
	private val castController by lazy { runCatching { WaveNodeCastController.get(application) }.getOrNull() }

    private val _state = MutableStateFlow(AppState(session = sessionStore.load()))
    val state: StateFlow<AppState> = _state.asStateFlow()
    val playerState: StateFlow<PlayerState> = player.state
    private var lastRecordedTrackId: String? = null
    private var appVisible = false
    private var podcastSearchJob: Job? = null
	private var audiobookSearchJob: Job? = null
    private var lastPodcastSnapshot: PlayerState? = null
    private var lastPodcastReportKey = ""
    private var lastPodcastReportPosition = -1
	private var sleepTimerJob: Job? = null

    init {
        if (_state.value.session != null) {
            refreshTracks()
            refreshPodcastHome()
			refreshAudiobookHome()
			refreshPodcastPreferences()
        } else {
            discoverServers()
        }
		_state.value = _state.value.copy(downloadedPodcastEpisodeIds = podcastDownloads.downloadedEpisodeIds())
        viewModelScope.launch {
            playerState.collect { state ->
                if (state.isPlaying) {
                    state.currentTrack?.let(::recordPlay)
                }
                val previous = lastPodcastSnapshot
                if (previous != null && previous.currentTrack?.externalKind in setOf("podcast", "audiobook") && previous.currentTrack?.id != state.currentTrack?.id) {
                    reportPodcastProgress(previous, force = true)
                }
                if (state.currentTrack?.externalKind in setOf("podcast", "audiobook")) {
                    lastPodcastSnapshot = state
                    reportPodcastProgress(state, force = !state.isPlaying)
                } else {
                    lastPodcastSnapshot = null
                }
            }
        }
        viewModelScope.launch {
            while (isActive) {
                val session = _state.value.session
                if (session != null && appVisible) {
                    pollPlaybackHandoff(session)
                }
                delay(3_000)
            }
        }
    }

    fun discoverServers() {
        if (_state.value.isDiscoveringServers) {
            return
        }
        viewModelScope.launch {
            _state.value = _state.value.copy(isDiscoveringServers = true)
            runCatching { serverDiscovery.discover() }
                .onSuccess { servers ->
                    _state.value = _state.value.copy(
                        discoveredServers = servers,
                        isDiscoveringServers = false,
                    )
                }
                .onFailure {
                    _state.value = _state.value.copy(isDiscoveringServers = false)
                }
        }
    }

    fun login(serverUrl: String, username: String, password: String) {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, error = null)
            runCatching { api.login(serverUrl, username, password) }
                .onSuccess { session ->
                    sessionStore.save(session)
                    _state.value = AppState(session = session, isLoading = true)
                    refreshTracks()
                    refreshPodcastHome()
					refreshAudiobookHome()
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(isLoading = false, error = error.message ?: "Login failed")
                }
        }
    }

    fun refreshTracks() {
        val session = _state.value.session ?: return
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, error = null)
            runCatching {
                LibraryPayload(
                    api.getTracks(session),
                    api.getAlbums(session),
                    api.getArtists(session),
                    api.getPlaylists(session),
                    api.getPluginHomeRows(session),
                )
            }
                .onSuccess { payload ->
                    _state.value = _state.value.copy(
                        tracks = payload.tracks,
                        albums = payload.albums,
                        artists = payload.artists,
                        playlists = payload.playlists,
                        pluginRows = payload.pluginRows,
                        isLoading = false,
                    )
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(isLoading = false, error = error.message ?: "Could not load library")
                }
        }
    }

    fun play(track: Track) {
        playFromHere(track, listOf(track))
    }

    fun openAlbum(album: Album) {
        val session = _state.value.session ?: return
        if (album.id.isBlank()) {
            val tracks = _state.value.tracks.filter { it.album == album.name && (album.artist.isBlank() || it.artist == album.artist) }
            _state.value = _state.value.copy(activeDetail = LibraryDetail.AlbumPage(album, tracks), detailError = null)
            return
        }
        viewModelScope.launch {
            _state.value = _state.value.copy(
                activeDetail = LibraryDetail.AlbumPage(album, emptyList()),
                isDetailLoading = true,
                detailError = null,
            )
            runCatching { api.getAlbumTracks(session, album.id) }
                .onSuccess { detail ->
                    _state.value = _state.value.copy(
                        activeDetail = LibraryDetail.AlbumPage(detail.album, detail.tracks),
                        isDetailLoading = false,
                    )
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(
                        isDetailLoading = false,
                        detailError = error.message ?: "Could not load album",
                    )
                }
        }
    }

    fun openArtist(artist: Artist) {
        val session = _state.value.session ?: return
        val artistKey = artist.id.ifBlank { artist.name }
        if (artistKey.isBlank()) {
            return
        }
        viewModelScope.launch {
            _state.value = _state.value.copy(
                activeDetail = LibraryDetail.ArtistPage(artist, emptyList(), emptyList()),
                isDetailLoading = true,
                detailError = null,
            )
            runCatching { api.getArtistTracks(session, artistKey) }
                .onSuccess { detail ->
                    _state.value = _state.value.copy(
                        activeDetail = LibraryDetail.ArtistPage(detail.artist, detail.tracks, detail.albums),
                        isDetailLoading = false,
                    )
                }
                .onFailure { error ->
                    val fallbackTracks = _state.value.tracks.filter { it.artist == artist.name }
                    _state.value = _state.value.copy(
                        activeDetail = LibraryDetail.ArtistPage(artist, fallbackTracks, emptyList()),
                        isDetailLoading = false,
                        detailError = error.message ?: "Could not load artist",
                    )
                }
        }
    }

    fun openPlaylist(playlist: Playlist) {
        val session = _state.value.session ?: return
        if (playlist.id.isBlank()) {
            return
        }
        viewModelScope.launch {
            _state.value = _state.value.copy(
                activeDetail = LibraryDetail.PlaylistPage(playlist, emptyList()),
                isDetailLoading = true,
                detailError = null,
            )
            runCatching { api.getPlaylistTracks(session, playlist.id) }
                .onSuccess { tracks ->
                    _state.value = _state.value.copy(
                        activeDetail = LibraryDetail.PlaylistPage(playlist, tracks),
                        isDetailLoading = false,
                    )
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(
                        isDetailLoading = false,
                        detailError = error.message ?: "Could not load playlist",
                    )
                }
        }
    }

    fun updatePodcastSearch(query: String) {
        val session = _state.value.session ?: return
        podcastSearchJob?.cancel()
        _state.value = _state.value.copy(
            podcastQuery = query,
            podcasts = if (query.isBlank()) emptyList() else _state.value.podcasts,
            isLoadingPodcasts = false,
            podcastError = null,
        )
        if (query.isBlank()) {
            return
        }
        podcastSearchJob = viewModelScope.launch {
            delay(350)
            _state.value = _state.value.copy(isLoadingPodcasts = true)
            runCatching { api.searchPodcasts(session, query.trim()) }
                .onSuccess { response ->
                    if (_state.value.podcastQuery == query) {
                        _state.value = _state.value.copy(
                            podcasts = response.results,
                            isLoadingPodcasts = false,
                            podcastError = null,
                        )
                    }
                }
                .onFailure { error ->
                    if (_state.value.podcastQuery == query) {
                        _state.value = _state.value.copy(
                            podcasts = emptyList(),
                            isLoadingPodcasts = false,
                            podcastError = error.message ?: "Could not search podcasts",
                        )
                    }
                }
        }
    }

    fun refreshPodcastHome() {
        val session = _state.value.session ?: return
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoadingPodcastHome = true, podcastHomeError = null)
            runCatching { api.getPodcastHome(session) }
                .onSuccess { home ->
                    _state.value = _state.value.copy(
                        podcastHome = home,
                        isLoadingPodcastHome = false,
                        podcastHomeError = null,
                    )
					home.subscriptions.filter { it.autoDownload }.take(10).forEach { subscription ->
						autoDownloadLatestEpisode(session, subscription)
					}
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(
                        isLoadingPodcastHome = false,
                        podcastHomeError = error.message ?: "Could not load podcasts",
                    )
                }
        }
    }

	fun updateAudiobookSearch(query: String) {
		val session = _state.value.session ?: return
		audiobookSearchJob?.cancel()
		_state.value = _state.value.copy(
			audiobookQuery = query,
			audiobooks = if (query.isBlank()) _state.value.audiobookHome.featured else _state.value.audiobooks,
			isLoadingAudiobooks = false,
			audiobookError = null,
		)
		if (query.isBlank()) return
		audiobookSearchJob = viewModelScope.launch {
			delay(350)
			_state.value = _state.value.copy(isLoadingAudiobooks = true)
			runCatching { api.searchAudiobooks(session, query.trim()) }
				.onSuccess { books ->
					if (_state.value.audiobookQuery == query) {
						_state.value = _state.value.copy(audiobooks = books, isLoadingAudiobooks = false)
					}
				}
				.onFailure { error ->
					if (_state.value.audiobookQuery == query) {
						_state.value = _state.value.copy(audiobooks = emptyList(), isLoadingAudiobooks = false, audiobookError = error.message ?: "Could not search audiobooks")
					}
				}
		}
	}

	fun refreshAudiobookHome() {
		val session = _state.value.session ?: return
		viewModelScope.launch {
			_state.value = _state.value.copy(isLoadingAudiobooks = true, audiobookError = null)
			runCatching { api.getAudiobookHome(session) }
				.onSuccess { home -> _state.value = _state.value.copy(
					audiobookHome = home,
					audiobooks = if (_state.value.audiobookQuery.isBlank()) home.featured else _state.value.audiobooks,
					isLoadingAudiobooks = false,
				) }
				.onFailure { error -> _state.value = _state.value.copy(isLoadingAudiobooks = false, audiobookError = error.message ?: "Could not load audiobooks") }
		}
	}

	fun openAudiobook(book: Audiobook) {
		val session = _state.value.session ?: return
		viewModelScope.launch {
			_state.value = _state.value.copy(isDetailLoading = true, detailError = null)
			runCatching { api.getAudiobook(session, book.id) }
				.onSuccess { detail -> _state.value = _state.value.copy(activeDetail = LibraryDetail.AudiobookPage(detail), isDetailLoading = false) }
				.onFailure { error -> _state.value = _state.value.copy(isDetailLoading = false, detailError = error.message ?: "Could not load audiobook") }
		}
	}

	fun resumeAudiobook(progress: AudiobookProgress) {
		val session = _state.value.session ?: return
		viewModelScope.launch {
			runCatching { api.getAudiobook(session, progress.bookId) }
				.onSuccess { detail ->
					val queue = detail.chapters.map { it.toTrack(detail.book) }
					val track = queue.firstOrNull { it.audiobookChapterId == progress.chapterId } ?: progress.toTrack()
					playFromHere(track, queue.ifEmpty { listOf(track) })
				}
				.onFailure { playFromHere(progress.toTrack(), listOf(progress.toTrack())) }
		}
	}

	private fun autoDownloadLatestEpisode(session: SavedSession, subscription: PodcastSubscription) {
		viewModelScope.launch {
			runCatching { api.getPodcastEpisodes(session, subscription.podcastId) }
				.onSuccess { detail -> detail.episodes.firstOrNull()?.let { downloadPodcastEpisode(detail.podcast, it) } }
				.onFailure { error -> Log.w("WaveNode", "Could not auto-download ${subscription.title}", error) }
		}
	}

	fun refreshPodcastPreferences() {
		val session = _state.value.session ?: return
		viewModelScope.launch {
			runCatching { api.getPodcastPreferences(session) }
				.onSuccess { preferences ->
					_state.value = _state.value.copy(podcastPreferences = preferences)
					if (playerState.value.currentTrack?.externalKind in setOf("podcast", "audiobook")) {
						player.setPlaybackSpeed(preferences.defaultPlaybackSpeed)
					}
				}
				.onFailure { error -> Log.w("WaveNode", "Could not load podcast preferences", error) }
		}
	}

	fun togglePodcastSubscription(podcast: Podcast, autoDownload: Boolean) {
		val session = _state.value.session ?: return
		val existing = _state.value.podcastHome.subscriptions.firstOrNull { it.podcastId == podcast.id }
		viewModelScope.launch {
			if (existing != null) {
				runCatching { api.deletePodcastSubscription(session, podcast.id) }
					.onSuccess {
						_state.value = _state.value.copy(
							podcastHome = _state.value.podcastHome.copy(
								subscriptions = _state.value.podcastHome.subscriptions.filterNot { it.podcastId == podcast.id },
							),
						)
					}
					.onFailure { error -> _state.value = _state.value.copy(detailError = error.message ?: "Could not unfollow podcast") }
				return@launch
			}
			val subscription = PodcastSubscription(
				podcastId = podcast.id,
				title = podcast.title,
				publisher = podcast.publisher,
				description = podcast.description,
				imageUrl = podcast.imageUrl,
				thumbnailUrl = podcast.thumbnailUrl,
				websiteUrl = podcast.websiteUrl,
				feedUrl = podcast.feedUrl,
				autoDownload = autoDownload,
				playbackSpeed = _state.value.podcastPreferences.defaultPlaybackSpeed,
			)
			runCatching { api.savePodcastSubscription(session, subscription) }
				.onSuccess { saved ->
					_state.value = _state.value.copy(
						podcastHome = _state.value.podcastHome.copy(
							subscriptions = listOf(saved) + _state.value.podcastHome.subscriptions.filterNot { it.podcastId == saved.podcastId },
						),
					)
					if (saved.autoDownload) {
						val active = _state.value.activeDetail as? LibraryDetail.PodcastPage
						active?.episodes?.firstOrNull()?.let { downloadPodcastEpisode(active.podcast, it) }
					}
				}
				.onFailure { error -> _state.value = _state.value.copy(detailError = error.message ?: "Could not follow podcast") }
		}
	}

	fun updatePodcastAutoDownload(podcast: Podcast, enabled: Boolean) {
		val session = _state.value.session ?: return
		val existing = _state.value.podcastHome.subscriptions.firstOrNull { it.podcastId == podcast.id } ?: return
		viewModelScope.launch {
			runCatching { api.savePodcastSubscription(session, existing.copy(autoDownload = enabled)) }
				.onSuccess { saved ->
					_state.value = _state.value.copy(podcastHome = _state.value.podcastHome.copy(
						subscriptions = _state.value.podcastHome.subscriptions.map { if (it.podcastId == saved.podcastId) saved else it },
					))
					if (saved.autoDownload) {
						val active = _state.value.activeDetail as? LibraryDetail.PodcastPage
						active?.episodes?.firstOrNull()?.let { downloadPodcastEpisode(active.podcast, it) }
					}
				}
				.onFailure { error -> _state.value = _state.value.copy(detailError = error.message ?: "Could not update auto-download") }
		}
	}

	fun downloadPodcastEpisode(podcast: Podcast, episode: PodcastEpisode) {
		val track = episode.toTrack(podcast)
		if (track.id in _state.value.downloadedPodcastEpisodeIds || track.id in _state.value.downloadingPodcastEpisodeIds) return
		_state.value = _state.value.copy(downloadingPodcastEpisodeIds = _state.value.downloadingPodcastEpisodeIds + track.id)
		viewModelScope.launch {
			runCatching { podcastDownloads.download(track) }
				.onSuccess {
					_state.value = _state.value.copy(
						downloadedPodcastEpisodeIds = podcastDownloads.downloadedEpisodeIds(),
						downloadingPodcastEpisodeIds = _state.value.downloadingPodcastEpisodeIds - track.id,
					)
				}
				.onFailure { error ->
					_state.value = _state.value.copy(
						downloadingPodcastEpisodeIds = _state.value.downloadingPodcastEpisodeIds - track.id,
						detailError = error.message ?: "Episode download failed",
					)
				}
		}
	}

	fun deletePodcastDownload(trackId: String) {
		podcastDownloads.delete(trackId)
		_state.value = _state.value.copy(downloadedPodcastEpisodeIds = podcastDownloads.downloadedEpisodeIds())
	}

    fun openPodcast(podcast: Podcast) {
        val session = _state.value.session ?: return
        viewModelScope.launch {
            _state.value = _state.value.copy(
                activeDetail = LibraryDetail.PodcastPage(podcast, emptyList()),
                isDetailLoading = true,
                detailError = null,
            )
            runCatching { api.getPodcastEpisodes(session, podcast.id) }
                .onSuccess { detail ->
                    _state.value = _state.value.copy(
                        activeDetail = LibraryDetail.PodcastPage(detail.podcast, detail.episodes),
                        isDetailLoading = false,
                    )
					if (_state.value.podcastHome.subscriptions.firstOrNull { it.podcastId == detail.podcast.id }?.autoDownload == true) {
						detail.episodes.firstOrNull()?.let { downloadPodcastEpisode(detail.podcast, it) }
					}
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(
                        isDetailLoading = false,
                        detailError = error.message ?: "Could not load podcast episodes",
                    )
                }
        }
    }

    fun resumePodcast(progress: PodcastProgress) {
        val session = _state.value.session ?: return
        viewModelScope.launch {
            runCatching { api.getPodcastEpisodes(session, progress.podcastId) }
                .onSuccess { detail ->
                    val queue = detail.episodes.map { it.toTrack(detail.podcast) }
                    val track = queue.firstOrNull { it.podcastEpisodeId == progress.episodeId }
                        ?: progress.toTrack()
                    playFromHere(track, queue.ifEmpty { listOf(track) })
                }
                .onFailure {
                    val track = progress.toTrack()
                    playFromHere(track, listOf(track))
                }
        }
    }

    fun closeDetail() {
        _state.value = _state.value.copy(activeDetail = null, isDetailLoading = false, detailError = null)
    }

    fun createPlaylist(name: String, description: String) {
        val session = _state.value.session ?: return
        val trimmedName = name.trim()
        if (trimmedName.isBlank()) {
            _state.value = _state.value.copy(error = "Playlist name is required")
            return
        }
        viewModelScope.launch {
            runCatching { api.createPlaylist(session, trimmedName, description.trim()) }
                .onSuccess { playlist ->
                    _state.value = _state.value.copy(
                        playlists = _state.value.playlists + playlist,
                        error = null,
                    )
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(error = error.message ?: "Could not create playlist")
                }
        }
    }

    fun createPlaylistWithTracks(name: String, description: String, tracks: List<Track>) {
        val session = _state.value.session ?: return
        val trimmedName = name.trim()
        val validTracks = tracks
            .filter { it.id.isNotBlank() && !it.isExternal }
            .distinctBy { it.id }
        if (trimmedName.isBlank()) {
            _state.value = _state.value.copy(error = "Playlist name is required")
            return
        }
        viewModelScope.launch {
            runCatching {
                val playlist = api.createPlaylist(session, trimmedName, description.trim())
                if (validTracks.isEmpty()) {
                    playlist
                } else {
                    api.addTracksToPlaylist(session, playlist.id, validTracks.map { it.id })
                }
            }
                .onSuccess { playlist ->
                    _state.value = _state.value.copy(
                        playlists = _state.value.playlists + playlist,
                        error = null,
                    )
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(error = error.message ?: "Could not create playlist")
                }
        }
    }

    fun updatePlaylist(playlist: Playlist) {
        val session = _state.value.session ?: return
        if (playlist.id.isBlank() || playlist.name.trim().isBlank()) {
            _state.value = _state.value.copy(error = "Playlist name is required")
            return
        }
        if (playlist.type == "smart") {
            _state.value = _state.value.copy(error = "Smart playlists are edited from the smart playlist editor")
            return
        }
        viewModelScope.launch {
            runCatching { api.updatePlaylist(session, playlist.copy(name = playlist.name.trim(), description = playlist.description.trim())) }
                .onSuccess { updatedPlaylist ->
                    val current = _state.value
                    val updatedDetail = when (val detail = current.activeDetail) {
                        is LibraryDetail.PlaylistPage -> {
                            if (detail.playlist.id == updatedPlaylist.id) {
                                val tracksById = detail.tracks.associateBy { it.id }
                                detail.copy(
                                    playlist = updatedPlaylist,
                                    tracks = updatedPlaylist.trackIds.mapNotNull { tracksById[it] },
                                )
                            } else {
                                detail
                            }
                        }
                        else -> detail
                    }
                    _state.value = current.copy(
                        playlists = current.playlists.map { if (it.id == updatedPlaylist.id) updatedPlaylist else it },
                        activeDetail = updatedDetail,
                        detailError = null,
                        error = null,
                    )
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(error = error.message ?: "Could not update playlist")
                }
        }
    }

    fun deletePlaylist(playlist: Playlist) {
        val session = _state.value.session ?: return
        if (playlist.id.isBlank()) {
            return
        }
        viewModelScope.launch {
            runCatching { api.deletePlaylist(session, playlist.id) }
                .onSuccess {
                    val current = _state.value
                    val updatedDetail = when (val detail = current.activeDetail) {
                        is LibraryDetail.PlaylistPage -> if (detail.playlist.id == playlist.id) null else detail
                        else -> current.activeDetail
                    }
                    _state.value = current.copy(
                        playlists = current.playlists.filterNot { it.id == playlist.id },
                        activeDetail = updatedDetail,
                        detailError = null,
                        error = null,
                    )
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(error = error.message ?: "Could not delete playlist")
                }
        }
    }

    fun addTrackToPlaylist(track: Track, playlist: Playlist) {
        addTracksToPlaylist(listOf(track), playlist)
    }

    fun addTracksToPlaylist(tracks: List<Track>, playlist: Playlist) {
        val session = _state.value.session ?: return
        val validTracks = tracks
            .filter { it.id.isNotBlank() && !it.isExternal }
            .distinctBy { it.id }
        if (validTracks.isEmpty() || playlist.id.isBlank() || playlist.type == "smart") {
            return
        }
        viewModelScope.launch {
            runCatching {
                if (validTracks.size == 1) {
                    api.addTrackToPlaylist(session, playlist.id, validTracks.first().id)
                } else {
                    api.addTracksToPlaylist(session, playlist.id, validTracks.map { it.id })
                }
            }
                .onSuccess { updatedPlaylist ->
                    val current = _state.value
                    val updatedPlaylists = current.playlists.map {
                        if (it.id == updatedPlaylist.id) updatedPlaylist else it
                    }
                    val updatedDetail = when (val detail = current.activeDetail) {
                        is LibraryDetail.PlaylistPage -> {
                            if (detail.playlist.id == updatedPlaylist.id) {
                                val existingIds = detail.tracks.map { it.id }.toSet()
                                detail.copy(
                                    playlist = updatedPlaylist,
                                    tracks = detail.tracks + validTracks.filter { it.id !in existingIds },
                                )
                            } else {
                                detail
                            }
                        }
                        else -> detail
                    }
                    _state.value = current.copy(
                        playlists = updatedPlaylists,
                        activeDetail = updatedDetail,
                        detailError = null,
                        error = null,
                    )
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(error = error.message ?: "Could not add tracks to playlist")
                }
        }
    }

    fun playFromHere(track: Track, queue: List<Track>) {
        val session = _state.value.session ?: return
		val playableQueue = queue.ifEmpty { listOf(track) }.map { candidate ->
			if (candidate.externalKind == "podcast") podcastDownloads.withLocalAudio(candidate) else candidate
		}
        val startIndex = playableQueue.indexOfFirst { it.id == track.id }.takeIf { it >= 0 } ?: 0
        val isLongformQueue = playableQueue.any { it.externalKind in setOf("podcast", "audiobook") }
        if (_state.value.connectedPlaybackSessionId.isNotBlank() && !isLongformQueue) {
            sendRemoteQueue(playableQueue, startIndex)
            return
        }
        if (isLongformQueue) {
            _state.value = _state.value.copy(
                connectedPlaybackSessionId = "",
                connectedPlaybackDeviceName = "",
                connectMessage = null,
            )
        }
        val startPositionMs = when (track.externalKind) {
			"podcast" -> track.takeUnless { it.podcastCompleted }?.podcastProgressSeconds?.toLong()?.times(1000L)
			"audiobook" -> track.takeUnless { it.audiobookCompleted }?.audiobookProgressSeconds?.toLong()?.times(1000L)
			else -> null
		} ?: 0L
        player.playQueue(session, playableQueue, startIndex, startPositionMs)
		if (isLongformQueue) {
			val podcastId = playableQueue[startIndex].podcastId
			val speed = _state.value.podcastHome.subscriptions.firstOrNull { it.podcastId == podcastId }?.playbackSpeed
				?: _state.value.podcastPreferences.defaultPlaybackSpeed
			player.setPlaybackSpeed(speed)
		}
        recordPlay(track)
        refreshRadioMetadata(track)
    }

    fun playQueueTrack(track: Track) {
        val session = _state.value.session ?: return
        val queue = playerState.value.queue.ifEmpty { listOf(track) }
        val startIndex = queue.indexOfFirst { it.id == track.id }.takeIf { it >= 0 } ?: 0
        val isLongformQueue = queue.any { it.externalKind in setOf("podcast", "audiobook") }
        if (_state.value.connectedPlaybackSessionId.isNotBlank() && !isLongformQueue) {
            sendRemoteQueue(queue, startIndex)
            return
        }
        if (isLongformQueue) {
            _state.value = _state.value.copy(
                connectedPlaybackSessionId = "",
                connectedPlaybackDeviceName = "",
                connectMessage = null,
            )
        }
        player.playQueue(session, queue, startIndex)
        recordPlay(track)
        refreshRadioMetadata(track)
    }

    fun skipNext() {
        if (_state.value.connectedPlaybackSessionId.isNotBlank()) {
            val state = playerState.value
            val queue = state.queue
            if (queue.isEmpty()) {
                return
            }
            val nextIndex = if (state.currentIndex < queue.lastIndex) state.currentIndex + 1 else 0
            sendRemoteQueue(queue, nextIndex)
            return
        }
        player.skipNext()
        playerState.value.currentTrack?.let(::recordPlay)
    }

    fun skipPrevious() {
        if (_state.value.connectedPlaybackSessionId.isNotBlank()) {
            val state = playerState.value
            val queue = state.queue
            if (queue.isEmpty()) {
                return
            }
            val previousIndex = if (state.currentIndex > 0) state.currentIndex - 1 else queue.lastIndex
            sendRemoteQueue(queue, previousIndex)
            return
        }
        player.skipPrevious()
        playerState.value.currentTrack?.let(::recordPlay)
    }

    private fun recordPlay(track: Track) {
        val session = _state.value.session ?: return
        if (track.isExternal) {
            return
        }
        if (lastRecordedTrackId == track.id) {
            return
        }
        lastRecordedTrackId = track.id
        viewModelScope.launch {
            runCatching { api.addRecentlyPlayed(session, track.id) }
                .onFailure { error ->
                    lastRecordedTrackId = null
                    Log.w("WaveNode", "Could not record mobile play", error)
                }
        }
    }

    private fun refreshRadioMetadata(track: Track) {
        val session = _state.value.session ?: return
        if (!track.isExternal || track.externalKind == "podcast" || track.streamUrl.isBlank()) {
            return
        }
        viewModelScope.launch {
            runCatching { api.getRadioMetadata(session, track.streamUrl) }
                .onSuccess { metadata ->
                    val streamTitle = metadata.streamTitle.trim()
                    if (streamTitle.isBlank()) {
                        return@onSuccess
                    }
                    val updatedTrack = track.copy(
                        title = metadata.stationTitle.ifBlank { track.title },
                        artist = streamTitle,
                        duration = 0,
                    )
                    player.updateCurrentTrack(updatedTrack)
                }
                .onFailure { error ->
                    Log.w("WaveNode", "Could not load radio metadata", error)
                }
        }
    }

    fun togglePlayPause() {
        if (_state.value.connectedPlaybackSessionId.isNotBlank()) {
            sendRemoteControl("toggle_play_pause")
            player.setRemotePlaying(!playerState.value.isPlaying)
            return
        }
        player.togglePlayPause()
    }

    fun toggleShuffle() {
        player.toggleShuffle()
    }

    fun cycleRepeatMode() {
        player.cycleRepeatMode()
    }

    fun seekTo(positionMs: Long) {
        if (_state.value.connectedPlaybackSessionId.isNotBlank()) {
            player.setRemotePosition(positionMs)
            sendRemoteControl("seek", positionMs = positionMs)
            return
        }
        player.seekTo(positionMs)
    }

	fun skipPodcastBack() {
		player.seekBy(-_state.value.podcastPreferences.skipBackSeconds * 1_000L)
	}

	fun skipPodcastForward() {
		player.seekBy(_state.value.podcastPreferences.skipForwardSeconds * 1_000L)
	}

	fun setPodcastPlaybackSpeed(speed: Float) {
		player.setPlaybackSpeed(speed)
		val session = _state.value.session ?: return
		val preferences = _state.value.podcastPreferences.copy(defaultPlaybackSpeed = speed.coerceIn(0.5f, 3f))
		_state.value = _state.value.copy(podcastPreferences = preferences)
		viewModelScope.launch {
			runCatching { api.updatePodcastPreferences(session, preferences) }
				.onSuccess { saved -> _state.value = _state.value.copy(podcastPreferences = saved) }
				.onFailure { error -> Log.w("WaveNode", "Could not save podcast speed", error) }
		}
	}

	fun setPodcastSleepTimer(minutes: Int?) {
		sleepTimerJob?.cancel()
		if (minutes == null || minutes <= 0) {
			_state.value = _state.value.copy(sleepTimerRemainingSeconds = 0)
			return
		}
		startSleepTimer(minutes * 60)
	}

	fun setPodcastSleepTimerToEpisodeEnd() {
		val playback = playerState.value
		val duration = playback.durationMs.takeIf { it > 0 } ?: playback.currentTrack?.duration?.times(1_000L) ?: 0L
		val remaining = ((duration - playback.positionMs).coerceAtLeast(0L) / 1_000L).toInt()
		if (remaining > 0) startSleepTimer(remaining)
	}

	private fun startSleepTimer(seconds: Int) {
		sleepTimerJob?.cancel()
		sleepTimerJob = viewModelScope.launch {
			for (remaining in seconds downTo 1) {
				_state.value = _state.value.copy(sleepTimerRemainingSeconds = remaining)
				delay(1_000)
			}
			player.pause()
			_state.value = _state.value.copy(sleepTimerRemainingSeconds = 0)
		}
	}

    fun setAppVisible(isVisible: Boolean) {
        if (!isVisible) {
            reportPodcastProgress(playerState.value, force = true)
        }
        appVisible = isVisible
        player.setAppVisible(isVisible)
        if (isVisible) {
            _state.value.session?.let { session ->
                viewModelScope.launch {
                    pollPlaybackHandoff(session)
                }
            }
        }
    }

    private fun reportPodcastProgress(playback: PlayerState, force: Boolean) {
        val session = _state.value.session ?: return
        val track = playback.currentTrack ?: return
        val isAudiobook = track.externalKind == "audiobook"
        if (track.externalKind !in setOf("podcast", "audiobook") || track.streamUrl.isBlank()) return
        if (isAudiobook && (track.audiobookId.isBlank() || track.audiobookChapterId.isBlank())) return
        if (!isAudiobook && (track.podcastId.isBlank() || track.podcastEpisodeId.isBlank())) return
        val position = (playback.positionMs / 1000L).toInt().coerceAtLeast(0)
        val duration = ((playback.durationMs.takeIf { it > 0L } ?: track.duration.toLong() * 1000L) / 1000L).toInt().coerceAtLeast(0)
        val key = if (isAudiobook) "${track.audiobookId}:${track.audiobookChapterId}" else "${track.podcastId}:${track.podcastEpisodeId}"
        if (!force && lastPodcastReportKey == key && kotlin.math.abs(position - lastPodcastReportPosition) < 10) return
        if (lastPodcastReportKey == key && lastPodcastReportPosition == position) return
        lastPodcastReportKey = key
        lastPodcastReportPosition = position
        viewModelScope.launch {
            if (isAudiobook) {
				runCatching {
					api.updateAudiobookProgress(session, AudiobookProgress(
						bookId = track.audiobookId,
						chapterId = track.audiobookChapterId,
						bookTitle = track.audiobookTitle.ifBlank { track.album.ifBlank { "Audiobook" } },
						author = track.audiobookAuthor.ifBlank { track.artist },
						chapterTitle = track.title,
						chapterNumber = track.audiobookChapterNumber,
						description = track.audiobookDescription,
						imageUrl = track.imageUrl,
						audioUrl = track.streamUrl,
						websiteUrl = track.audiobookWebsiteUrl,
						durationSeconds = duration,
						positionSeconds = position,
					))
				}.onSuccess(::applyAudiobookProgress)
					.onFailure { error -> lastPodcastReportKey = ""; Log.w("WaveNode", "Could not save audiobook progress", error) }
			} else runCatching {
				api.updatePodcastProgress(
                    session,
                    PodcastProgress(
                        podcastId = track.podcastId,
                        episodeId = track.podcastEpisodeId,
                        podcastTitle = track.podcastTitle.ifBlank { track.album.ifBlank { "Podcast" } },
                        publisher = track.podcastPublisher,
                        episodeTitle = track.title,
                        description = track.podcastDescription,
                        imageUrl = track.imageUrl,
                        audioUrl = track.podcastAudioUrl.ifBlank { track.streamUrl },
                        websiteUrl = track.podcastWebsiteUrl,
                        publishedAt = track.releaseDate,
                        durationSeconds = duration,
                        positionSeconds = position,
                    ),
                )
            }.onSuccess(::applyPodcastProgress)
                .onFailure { error ->
                    lastPodcastReportKey = ""
                    Log.w("WaveNode", "Could not save podcast progress", error)
                }
        }
    }

	private fun applyAudiobookProgress(saved: AudiobookProgress) {
		val current = _state.value
		val remaining = current.audiobookHome.continueListening.filterNot { it.bookId == saved.bookId }
		val updated = if (saved.positionSeconds > 0 && !saved.completed) (listOf(saved) + remaining).take(12) else remaining
		val detail = when (val active = current.activeDetail) {
			is LibraryDetail.AudiobookPage -> active.copy(detail = active.detail.copy(chapters = active.detail.chapters.map { chapter ->
				if (chapter.id == saved.chapterId) chapter.copy(progressSeconds = saved.positionSeconds, completed = saved.completed) else chapter
			}))
			else -> active
		}
		_state.value = current.copy(audiobookHome = current.audiobookHome.copy(continueListening = updated), activeDetail = detail)
	}

    private fun applyPodcastProgress(saved: PodcastProgress) {
		if (saved.completed && _state.value.podcastPreferences.autoDeletePlayed) {
			podcastDownloads.delete("podcast:${saved.podcastId}:${saved.episodeId}")
		}
        val current = _state.value
        val remaining = current.podcastHome.continueListening.filterNot {
            it.podcastId == saved.podcastId && it.episodeId == saved.episodeId
        }
        val updatedContinue = if (saved.positionSeconds > 0 && !saved.completed) {
            (listOf(saved) + remaining).take(12)
        } else {
            remaining
        }
        val detail = when (val active = current.activeDetail) {
            is LibraryDetail.PodcastPage -> active.copy(episodes = active.episodes.map { episode ->
                if (episode.id == saved.episodeId) episode.copy(
                    progressSeconds = saved.positionSeconds,
                    completed = saved.completed,
                ) else episode
            })
            else -> active
        }
        _state.value = current.copy(
            podcastHome = current.podcastHome.copy(continueListening = updatedContinue),
            activeDetail = detail,
			downloadedPodcastEpisodeIds = podcastDownloads.downloadedEpisodeIds(),
        )
    }

    private suspend fun pollPlaybackHandoff(session: SavedSession) {
        runCatching { api.consumePendingPlaybackHandoff(session) }
            .onSuccess { command ->
                if (command != null) {
                    handlePlaybackHandoff(session, command)
                }
            }
            .onFailure { error ->
                Log.w("WaveNode", "Could not receive playback handoff", error)
            }
    }

    private fun handlePlaybackHandoff(session: SavedSession, command: org.wavenode.player.data.PlaybackHandoffCommand) {
        _state.value = _state.value.copy(
            connectedPlaybackSessionId = "",
            connectedPlaybackDeviceName = "",
            connectMessage = null,
        )
        when (command.action) {
            "toggle_play_pause" -> player.togglePlayPause()
            "seek" -> player.seekTo(command.positionMs)
            else -> {
                val queue = command.tracks.ifEmpty {
                    val tracksById = _state.value.tracks.associateBy { it.id }
                    command.trackIds.mapNotNull { tracksById[it] }
                }
                if (queue.isNotEmpty()) {
                    player.playQueue(session, queue, command.startIndex, command.positionMs)
                }
            }
        }
    }

    fun refreshConnectSessions() {
        val session = _state.value.session ?: return
		refreshOutputDevices()
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoadingConnectSessions = true, connectMessage = null)
            runCatching { api.getSessions(session) }
                .onSuccess { response ->
                    val sessions = response.sessions.visibleConnectSessions(response.currentSessionId)
                    _state.value = _state.value.copy(
                        connectSessions = sessions,
                        currentSessionId = response.currentSessionId,
                        isLoadingConnectSessions = false,
                    )
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(
                        isLoadingConnectSessions = false,
                        connectMessage = error.message ?: "Could not load connected devices",
                    )
                }
        }
    }

	private fun refreshOutputDevices() {
		val session = _state.value.session ?: return
		viewModelScope.launch {
			_state.value = _state.value.copy(isLoadingOutputDevices = true)
			runCatching { api.discoverOutputDevices(session) }
				.onSuccess { devices -> _state.value = _state.value.copy(outputDevices = devices, isLoadingOutputDevices = false) }
				.onFailure { _state.value = _state.value.copy(outputDevices = emptyList(), isLoadingOutputDevices = false) }
		}
	}

	fun prepareGoogleCast() {
		val session = _state.value.session ?: return
		val playback = playerState.value
		val track = playback.currentTrack ?: run {
			_state.value = _state.value.copy(connectMessage = "Choose something to play first")
			return
		}
		viewModelScope.launch {
			runCatching {
				val mediaURL = if (track.isExternal) track.podcastAudioUrl.ifBlank { track.streamUrl } else api.createCastURL(session, track.id).url
				if (!mediaURL.startsWith("http")) throw IllegalStateException("This download is only available on this phone")
				val controller = castController ?: throw IllegalStateException("Google Cast is unavailable on this device")
				controller.prepare(track, mediaURL, playback.positionMs) { player.pause() }
			}
				.onFailure { error -> _state.value = _state.value.copy(connectMessage = error.message ?: "Could not prepare Google Cast") }
		}
	}

    fun loadLyrics(track: Track) {
        val session = _state.value.session ?: return
        if (track.isExternal) {
            _state.value = _state.value.copy(lyrics = null, lyricsTrackId = track.id, lyricsError = "Lyrics are unavailable for this source")
            return
        }
        if (_state.value.lyricsTrackId == track.id && (_state.value.lyrics != null || _state.value.isLoadingLyrics)) return
        _state.value = _state.value.copy(lyrics = null, lyricsTrackId = track.id, isLoadingLyrics = true, lyricsError = null)
        viewModelScope.launch {
            runCatching { api.getLyrics(session, track.id) }
                .onSuccess { lyrics ->
                    if (_state.value.lyricsTrackId == track.id) {
                        _state.value = _state.value.copy(lyrics = lyrics, isLoadingLyrics = false)
                    }
                }
                .onFailure { error ->
                    if (_state.value.lyricsTrackId == track.id) {
                        _state.value = _state.value.copy(isLoadingLyrics = false, lyricsError = error.message ?: "Lyrics could not be loaded")
                    }
                }
        }
    }

	fun playOnOutputDevice(deviceId: String) {
		val session = _state.value.session ?: return
		val playback = playerState.value
		val track = playback.currentTrack ?: run {
			_state.value = _state.value.copy(connectMessage = "Choose something to play first")
			return
		}
		viewModelScope.launch {
			_state.value = _state.value.copy(connectMessage = null)
			runCatching {
				val mediaURL = if (track.isExternal) track.podcastAudioUrl.ifBlank { track.streamUrl } else api.createCastURL(session, track.id).url
				api.playOnDLNADevice(session, deviceId, mediaURL, track.title)
			}
				.onSuccess { device ->
					player.pause()
					_state.value = _state.value.copy(connectMessage = "Playing on ${device.name.ifBlank { "DLNA renderer" }}")
				}
				.onFailure { error -> _state.value = _state.value.copy(connectMessage = error.message ?: "Could not play on renderer") }
		}
	}

    fun connectPlaybackTo(sessionId: String) {
        val session = _state.value.session ?: return
        if (sessionId == _state.value.currentSessionId) {
            if (!player.resumeLocalPlayback(session)) {
                _state.value = _state.value.copy(connectMessage = "Choose something to play first")
                return
            }
            _state.value = _state.value.copy(
                connectedPlaybackSessionId = "",
                connectedPlaybackDeviceName = "",
                connectMessage = "Playback switched to this phone",
            )
            return
        }
        val playerState = playerState.value
        val queue = playerState.queue.ifEmpty { playerState.currentTrack?.let(::listOf).orEmpty() }
        if (queue.isEmpty()) {
            _state.value = _state.value.copy(connectMessage = "Choose something to play first")
            return
        }
        viewModelScope.launch {
            _state.value = _state.value.copy(connectMessage = null)
            runCatching {
                api.createPlaybackHandoff(
                    session = session,
                    targetSessionId = sessionId,
                    trackIds = queue.map { it.id },
                    startIndex = playerState.currentIndex.coerceAtLeast(0),
                    positionMs = playerState.positionMs,
                )
            }
                .onSuccess {
                    val deviceName = _state.value.connectSessions.firstOrNull { it.id == sessionId }?.deviceName ?: "device"
                    player.setRemoteQueue(queue, playerState.currentIndex.coerceAtLeast(0), isPlaying = true, positionMs = playerState.positionMs)
                    _state.value = _state.value.copy(
                        connectedPlaybackSessionId = sessionId,
                        connectedPlaybackDeviceName = deviceName,
                        connectMessage = "Controlling $deviceName",
                    )
                }
                .onFailure { error ->
                    _state.value = _state.value.copy(connectMessage = error.message ?: "Could not connect to device")
                }
        }
    }

    private fun sendRemoteQueue(queue: List<Track>, startIndex: Int) {
        val session = _state.value.session ?: return
        val targetSessionId = _state.value.connectedPlaybackSessionId
        if (targetSessionId.isBlank()) {
            return
        }
        val playableQueue = queue.filter { it.id.isNotBlank() }
        if (playableQueue.isEmpty()) {
            return
        }
        val safeIndex = startIndex.coerceIn(0, playableQueue.lastIndex)
        val positionMs = if (safeIndex == playerState.value.currentIndex) playerState.value.positionMs else 0L
        player.setRemoteQueue(playableQueue, safeIndex, isPlaying = true, positionMs = positionMs)
        viewModelScope.launch {
            runCatching {
                api.createPlaybackHandoff(
                    session = session,
                    targetSessionId = targetSessionId,
                    trackIds = playableQueue.map { it.id },
                    startIndex = safeIndex,
                    positionMs = positionMs,
                )
            }.onFailure { error ->
                _state.value = _state.value.copy(connectMessage = error.message ?: "Could not control remote device")
            }
        }
    }

    private fun sendRemoteControl(action: String, positionMs: Long? = null) {
        val session = _state.value.session ?: return
        val targetSessionId = _state.value.connectedPlaybackSessionId
        if (targetSessionId.isBlank()) {
            return
        }
        viewModelScope.launch {
            runCatching {
                api.createPlaybackHandoff(
                    session = session,
                    targetSessionId = targetSessionId,
                    trackIds = emptyList(),
                    startIndex = 0,
                    action = action,
                    positionMs = positionMs,
                )
            }.onFailure { error ->
                _state.value = _state.value.copy(connectMessage = error.message ?: "Could not control remote device")
            }
        }
    }

    fun logout() {
        podcastSearchJob?.cancel()
		audiobookSearchJob?.cancel()
        sessionStore.clear()
        _state.value = AppState()
    }

    fun artworkUrl(track: Track): String? {
        if (track.isExternal && track.imageUrl.isNotBlank()) {
            return track.imageUrl
        }
        val session = _state.value.session ?: return null
        return api.artworkUrl(session, track)
    }

    fun artworkUrl(album: Album): String? {
        val session = _state.value.session ?: return null
        return api.artworkUrl(session, album)
    }

    fun artworkUrl(artist: Artist): String? {
        val session = _state.value.session ?: return null
        return api.artworkUrl(session, artist)
    }

    override fun onCleared() {
        super.onCleared()
    }

    private data class LibraryPayload(
        val tracks: List<Track>,
        val albums: List<Album>,
        val artists: List<Artist>,
        val playlists: List<Playlist>,
        val pluginRows: List<PluginHomeRow>,
    )
}

private fun PodcastEpisode.toTrack(podcast: Podcast): Track {
    return Track(
        id = "podcast:${podcast.id}:$id",
        title = title.ifBlank { "Untitled episode" },
        artist = podcast.title.ifBlank { podcast.publisher.ifBlank { "Podcast" } },
        album = podcast.title.ifBlank { "Podcast" },
        genre = "Podcast",
        duration = duration,
        releaseDate = publishedAt,
        imageUrl = imageUrl.ifBlank { podcast.imageUrl.ifBlank { podcast.thumbnailUrl } },
        streamUrl = audioUrl,
		podcastAudioUrl = audioUrl,
        isExternal = true,
        externalKind = "podcast",
        podcastId = podcast.id,
        podcastTitle = podcast.title,
        podcastPublisher = podcast.publisher,
        podcastEpisodeId = id,
        podcastDescription = description,
        podcastWebsiteUrl = websiteUrl.ifBlank { podcast.websiteUrl },
		podcastChaptersUrl = chaptersUrl,
        podcastProgressSeconds = progressSeconds,
        podcastCompleted = completed,
        createdAt = publishedAt,
    )
}

private fun PodcastProgress.toTrack(): Track {
    return Track(
        id = "podcast:$podcastId:$episodeId",
        title = episodeTitle,
        artist = podcastTitle.ifBlank { publisher.ifBlank { "Podcast" } },
        album = podcastTitle.ifBlank { "Podcast" },
        genre = "Podcast",
        duration = durationSeconds,
        releaseDate = publishedAt,
        imageUrl = imageUrl,
        streamUrl = audioUrl,
		podcastAudioUrl = audioUrl,
        isExternal = true,
        externalKind = "podcast",
        podcastId = podcastId,
        podcastTitle = podcastTitle,
        podcastPublisher = publisher,
        podcastEpisodeId = episodeId,
        podcastDescription = description,
        podcastWebsiteUrl = websiteUrl,
        podcastProgressSeconds = positionSeconds,
        podcastCompleted = completed,
        createdAt = publishedAt.orEmpty(),
    )
}

private fun AudiobookChapter.toTrack(book: Audiobook): Track = Track(
	id = "audiobook:${book.id}:$id",
	title = title.ifBlank { "Chapter $number" },
	artist = book.author,
	album = book.title,
	genre = "Audiobook",
	duration = durationSeconds,
	imageUrl = book.imageUrl,
	streamUrl = audioUrl,
	isExternal = true,
	externalKind = "audiobook",
	audiobookId = book.id,
	audiobookTitle = book.title,
	audiobookAuthor = book.author,
	audiobookChapterId = id,
	audiobookChapterNumber = number,
	audiobookDescription = book.description,
	audiobookWebsiteUrl = book.websiteUrl,
	audiobookProgressSeconds = progressSeconds,
	audiobookCompleted = completed,
)

private fun AudiobookProgress.toTrack(): Track = Track(
	id = "audiobook:$bookId:$chapterId",
	title = chapterTitle,
	artist = author,
	album = bookTitle,
	genre = "Audiobook",
	duration = durationSeconds,
	imageUrl = imageUrl,
	streamUrl = audioUrl,
	isExternal = true,
	externalKind = "audiobook",
	audiobookId = bookId,
	audiobookTitle = bookTitle,
	audiobookAuthor = author,
	audiobookChapterId = chapterId,
	audiobookChapterNumber = chapterNumber,
	audiobookDescription = description,
	audiobookWebsiteUrl = websiteUrl,
	audiobookProgressSeconds = positionSeconds,
	audiobookCompleted = completed,
)

private fun List<UserSession>.visibleConnectSessions(currentSessionId: String): List<UserSession> {
    val activeSince = Instant.now().minus(Duration.ofMinutes(15))
    return filter { session ->
        session.id == currentSessionId || session.lastSeenInstant()?.isAfter(activeSince) == true
    }
        .sortedWith(
            compareByDescending<UserSession> { it.id == currentSessionId }
                .thenByDescending { it.lastSeenInstant() ?: Instant.EPOCH },
        )
        .distinctBy { session ->
            if (session.id == currentSessionId) {
                "current:${session.id}"
            } else {
                "${session.deviceName.trim().lowercase()}|${session.ipAddress.trim()}"
            }
        }
}

private fun UserSession.lastSeenInstant(): Instant? {
    return runCatching { Instant.parse(lastSeenAt) }.getOrNull()
}
