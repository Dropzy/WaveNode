package org.wavenode.player

import android.app.Application
import android.util.Log
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import org.wavenode.player.data.Album
import org.wavenode.player.data.Artist
import org.wavenode.player.data.DiscoveredServer
import org.wavenode.player.data.Playlist
import org.wavenode.player.data.PluginHomeRow
import org.wavenode.player.data.SavedSession
import org.wavenode.player.data.ServerDiscovery
import org.wavenode.player.data.SessionStore
import org.wavenode.player.data.Track
import org.wavenode.player.data.UserSession
import org.wavenode.player.data.WaveNodeApi
import org.wavenode.player.playback.PlayerState
import org.wavenode.player.playback.WaveNodePlayer
import java.time.Duration
import java.time.Instant

data class AppState(
    val session: SavedSession? = null,
    val tracks: List<Track> = emptyList(),
    val albums: List<Album> = emptyList(),
    val artists: List<Artist> = emptyList(),
    val playlists: List<Playlist> = emptyList(),
    val pluginRows: List<PluginHomeRow> = emptyList(),
    val discoveredServers: List<DiscoveredServer> = emptyList(),
    val connectSessions: List<UserSession> = emptyList(),
    val currentSessionId: String = "",
    val connectedPlaybackSessionId: String = "",
    val connectedPlaybackDeviceName: String = "",
    val isLoadingConnectSessions: Boolean = false,
    val connectMessage: String? = null,
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
}

class WaveNodeViewModel(application: Application) : AndroidViewModel(application) {
    private val api = WaveNodeApi()
    private val serverDiscovery = ServerDiscovery()
    private val sessionStore = SessionStore(application)
    private val player = WaveNodePlayer.get(application, api)

    private val _state = MutableStateFlow(AppState(session = sessionStore.load()))
    val state: StateFlow<AppState> = _state.asStateFlow()
    val playerState: StateFlow<PlayerState> = player.state
    private var lastRecordedTrackId: String? = null
    private var appVisible = false

    init {
        if (_state.value.session != null) {
            refreshTracks()
        } else {
            discoverServers()
        }
        viewModelScope.launch {
            playerState.collect { state ->
                if (state.isPlaying) {
                    state.currentTrack?.let(::recordPlay)
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
        val playableQueue = queue.ifEmpty { listOf(track) }
        val startIndex = playableQueue.indexOfFirst { it.id == track.id }.takeIf { it >= 0 } ?: 0
        if (_state.value.connectedPlaybackSessionId.isNotBlank()) {
            sendRemoteQueue(playableQueue, startIndex)
            return
        }
        player.playQueue(session, playableQueue, startIndex)
        recordPlay(track)
        refreshRadioMetadata(track)
    }

    fun playQueueTrack(track: Track) {
        val session = _state.value.session ?: return
        val queue = playerState.value.queue.ifEmpty { listOf(track) }
        val startIndex = queue.indexOfFirst { it.id == track.id }.takeIf { it >= 0 } ?: 0
        if (_state.value.connectedPlaybackSessionId.isNotBlank()) {
            sendRemoteQueue(queue, startIndex)
            return
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
        if (!track.isExternal || track.streamUrl.isBlank()) {
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

    fun setAppVisible(isVisible: Boolean) {
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
