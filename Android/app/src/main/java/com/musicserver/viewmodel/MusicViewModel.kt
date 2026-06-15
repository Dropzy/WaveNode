package com.musicserver.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.musicserver.BuildConfig
import com.musicserver.data.models.*
import com.musicserver.data.repository.MusicRepository
import androidx.media3.common.MediaItem
import com.musicserver.player.MusicService
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.delay
import javax.inject.Inject
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.content.ServiceConnection
import android.os.IBinder

@HiltViewModel
class MusicViewModel @Inject constructor(
    private val musicRepository: MusicRepository
) : ViewModel() {
    
    // Service Connection
    private var musicService: MusicService? = null
    private val _isServiceConnected = MutableStateFlow(false)
    val isServiceConnected: StateFlow<Boolean> = _isServiceConnected.asStateFlow()
    
    private val serviceConnection = object : ServiceConnection {
        override fun onServiceConnected(name: ComponentName?, service: IBinder?) {
            val binder = service as MusicService.MusicBinder
            musicService = binder.getService()
            _isServiceConnected.value = true
            android.util.Log.d("MusicViewModel", "MusicService connected")
            
            // Start observing service state
            observeServiceState()
        }
        
        override fun onServiceDisconnected(name: ComponentName?) {
            musicService = null
            _isServiceConnected.value = false
            android.util.Log.d("MusicViewModel", "MusicService disconnected")
        }
    }
    
    // Music Library
    private val _musicLibrary = MutableStateFlow<List<Music>>(emptyList())
    val musicLibrary: StateFlow<List<Music>> = _musicLibrary.asStateFlow()
    
    private val _searchResults = MutableStateFlow<List<Music>>(emptyList())
    val searchResults: StateFlow<List<Music>> = _searchResults.asStateFlow()
    
    // Albums
    private val _albums = MutableStateFlow<List<Album>>(emptyList())
    val albums: StateFlow<List<Album>> = _albums.asStateFlow()
    
    private val _currentAlbum = MutableStateFlow<AlbumTracksResponse?>(null)
    val currentAlbum: StateFlow<AlbumTracksResponse?> = _currentAlbum.asStateFlow()
    
    // Artists
    private val _artists = MutableStateFlow<List<String>>(emptyList())
    val artists: StateFlow<List<String>> = _artists.asStateFlow()
    
    private val _currentArtist = MutableStateFlow<ArtistTracksResponse?>(null)
    val currentArtist: StateFlow<ArtistTracksResponse?> = _currentArtist.asStateFlow()
    
    // Playlists
    private val _playlists = MutableStateFlow<List<Playlist>>(emptyList())
    val playlists: StateFlow<List<Playlist>> = _playlists.asStateFlow()
    
    private val _currentPlaylist = MutableStateFlow<Playlist?>(null)
    val currentPlaylist: StateFlow<Playlist?> = _currentPlaylist.asStateFlow()
    
    // Liked Tracks
    private val _likedTracks = MutableStateFlow<List<Music>>(emptyList())
    val likedTracks: StateFlow<List<Music>> = _likedTracks.asStateFlow()
    
    private val _likedTrackIds = MutableStateFlow<Set<String>>(emptySet())
    val likedTrackIds: StateFlow<Set<String>> = _likedTrackIds.asStateFlow()
    
    // Current Playing Track
    private val _currentTrack = MutableStateFlow<Music?>(null)
    val currentTrack: StateFlow<Music?> = _currentTrack.asStateFlow()
    
    private val _isPlaying = MutableStateFlow(false)
    val isPlaying: StateFlow<Boolean> = _isPlaying.asStateFlow()
    
    private val _playbackProgress = MutableStateFlow(0f)
    val playbackProgress: StateFlow<Float> = _playbackProgress.asStateFlow()
    
    private val _currentPosition = MutableStateFlow(0L)
    val currentPosition: StateFlow<Long> = _currentPosition.asStateFlow()
    
    private val _duration = MutableStateFlow(0L)
    val duration: StateFlow<Long> = _duration.asStateFlow()
    
    // Loading and Error States
    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()
    
    private val _errorMessage = MutableStateFlow<String?>(null)
    val errorMessage: StateFlow<String?> = _errorMessage.asStateFlow()
    
    init {
        loadMusicLibrary()
        loadAlbums()
        loadArtists()
        loadPlaylists()
        loadLikedTracks()
        
        // Add sample data for testing if no data is loaded
        viewModelScope.launch {
            delay(2000) // Wait for API calls to complete
            if (_musicLibrary.value.isEmpty()) {
                android.util.Log.d("MusicViewModel", "No music data loaded, adding sample data")
                addSampleData()
            }
        }
    }
    
    // Service Connection Methods
    fun connectToService(context: Context) {
        val intent = Intent(context, MusicService::class.java)
        context.bindService(intent, serviceConnection, Context.BIND_AUTO_CREATE)
        android.util.Log.d("MusicViewModel", "Attempting to connect to MusicService")
    }
    
    fun disconnectFromService(context: Context) {
        context.unbindService(serviceConnection)
        _isServiceConnected.value = false
        musicService = null
    }
    
    private fun observeServiceState() {
        musicService?.let { service ->
            // Observe current track from service
            viewModelScope.launch {
                service.currentTrack.collect { mediaItem ->
                    updateCurrentTrack(mediaItem)
                }
            }
            
            // Observe playing state from service
            viewModelScope.launch {
                service.isPlaying.collect { playing ->
                    updatePlaybackState(playing)
                }
            }
            
            // Observe current position from service
            viewModelScope.launch {
                service.currentPosition.collect { position ->
                    updateCurrentPosition(position)
                }
            }
            
            // Observe duration from service
            viewModelScope.launch {
                service.duration.collect { duration ->
                    updateDuration(duration)
                }
            }
        }
    }
    
    // Playback Control Methods
    fun playTrack(track: Music) {
        viewModelScope.launch {
            try {
                android.util.Log.d("MusicViewModel", "Attempting to play track: ${track.title}")
                
                // Ensure service is connected
                if (!isServiceConnected.value) {
                    android.util.Log.d("MusicViewModel", "Service not connected, waiting...")
                    delay(2000) // Wait longer for service connection
                    if (!isServiceConnected.value) {
                        android.util.Log.e("MusicViewModel", "Service still not connected after delay")
                        _errorMessage.value = "Music service not connected"
                        return@launch
                    }
                }
                
                // Create MediaItem from track
                val mediaItem = createMediaItem(track)
                android.util.Log.d("MusicViewModel", "Created MediaItem: ${mediaItem.mediaMetadata.title}")
                
                musicService?.playTrack(mediaItem)
                android.util.Log.d("MusicViewModel", "playTrack call completed")
            } catch (e: Exception) {
                android.util.Log.e("MusicViewModel", "Error playing track", e)
                _errorMessage.value = "Failed to play track: ${e.message}"
            }
        }
    }
    
    fun playTracks(tracks: List<Music>) {
        viewModelScope.launch {
            try {
                android.util.Log.d("MusicViewModel", "Attempting to play ${tracks.size} tracks")
                
                // Ensure service is connected
                if (!isServiceConnected.value) {
                    android.util.Log.d("MusicViewModel", "Service not connected, waiting...")
                    delay(2000) // Wait longer for service connection
                    if (!isServiceConnected.value) {
                        android.util.Log.e("MusicViewModel", "Service still not connected after delay")
                        _errorMessage.value = "Music service not connected"
                        return@launch
                    }
                }
                
                if (tracks.isEmpty()) {
                    android.util.Log.w("MusicViewModel", "No tracks to play")
                    return@launch
                }
                
                // Create MediaItems from tracks
                val mediaItems = tracks.map { track ->
                    createMediaItem(track)
                }
                android.util.Log.d("MusicViewModel", "Created ${mediaItems.size} MediaItems")
                
                musicService?.setPlaylist(mediaItems, 0) // Start from first track
                android.util.Log.d("MusicViewModel", "setPlaylist call completed")
            } catch (e: Exception) {
                android.util.Log.e("MusicViewModel", "Error playing tracks", e)
                _errorMessage.value = "Failed to play tracks: ${e.message}"
            }
        }
    }
    
    fun play() {
        musicService?.play()
    }
    
    fun pause() {
        musicService?.pause()
    }
    
    fun skipToNext() {
        musicService?.skipToNext()
    }
    
    fun skipToPrevious() {
        musicService?.skipToPrevious()
    }
    
    fun seekTo(position: Long) {
        musicService?.seekTo(position)
    }
    
    // Music Library
    fun loadMusicLibrary() {
        viewModelScope.launch {
            _isLoading.value = true
            _errorMessage.value = null
            
            musicRepository.getAllMusic()
                .onSuccess { music ->
                    _musicLibrary.value = music
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to load music library"
                }
            
            _isLoading.value = false
        }
    }
    
    fun refreshMusicLibrary() {
        android.util.Log.d("MusicViewModel", "refreshMusicLibrary called")
        loadMusicLibrary()
        loadAlbums()
        loadArtists()
        loadPlaylists()
        loadLikedTracks()
    }
    
    fun searchMusic(query: String) {
        if (query.isBlank()) {
            _searchResults.value = emptyList()
            return
        }
        
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.searchMusic(query)
                .onSuccess { results ->
                    _searchResults.value = results
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Search failed"
                }
            
            _isLoading.value = false
        }
    }
    
    // Albums
    fun loadAlbums() {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.getAlbums()
                .onSuccess { albums ->
                    _albums.value = albums
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to load albums"
                }
            
            _isLoading.value = false
        }
    }
    
    fun loadAlbumTracks(albumName: String) {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.getAlbumTracks(albumName)
                .onSuccess { albumTracks ->
                    _currentAlbum.value = albumTracks
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to load album tracks"
                }
            
            _isLoading.value = false
        }
    }
    
    // Artists
    fun loadArtists() {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.getArtists()
                .onSuccess { artists ->
                    _artists.value = artists
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to load artists"
                }
            
            _isLoading.value = false
        }
    }
    
    fun loadArtistTracks(artistName: String) {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.getArtistTracks(artistName)
                .onSuccess { artistTracks ->
                    _currentArtist.value = artistTracks
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to load artist tracks"
                }
            
            _isLoading.value = false
        }
    }
    
    // Playlists
    fun loadPlaylists() {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.getAllPlaylists()
                .onSuccess { playlists ->
                    _playlists.value = playlists
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to load playlists"
                }
            
            _isLoading.value = false
        }
    }
    
    fun createPlaylist(name: String, description: String) {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.createPlaylist(name, description)
                .onSuccess { playlist ->
                    _playlists.value = _playlists.value + playlist
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to create playlist"
                }
            
            _isLoading.value = false
        }
    }
    
    fun updatePlaylist(id: String, name: String?, description: String?) {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.updatePlaylist(id, name, description)
                .onSuccess { updatedPlaylist ->
                    _playlists.value = _playlists.value.map { playlist ->
                        if (playlist.id == id) updatedPlaylist else playlist
                    }
                    if (_currentPlaylist.value?.id == id) {
                        _currentPlaylist.value = updatedPlaylist
                    }
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to update playlist"
                }
            
            _isLoading.value = false
        }
    }
    
    fun deletePlaylist(id: String) {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.deletePlaylist(id)
                .onSuccess {
                    _playlists.value = _playlists.value.filter { it.id != id }
                    if (_currentPlaylist.value?.id == id) {
                        _currentPlaylist.value = null
                    }
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to delete playlist"
                }
            
            _isLoading.value = false
        }
    }
    
    fun addTrackToPlaylist(playlistId: String, trackId: String) {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.addTrackToPlaylist(playlistId, trackId)
                .onSuccess { updatedPlaylist ->
                    _playlists.value = _playlists.value.map { playlist ->
                        if (playlist.id == playlistId) updatedPlaylist else playlist
                    }
                    if (_currentPlaylist.value?.id == playlistId) {
                        _currentPlaylist.value = updatedPlaylist
                    }
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to add track to playlist"
                }
            
            _isLoading.value = false
        }
    }
    
    fun removeTrackFromPlaylist(playlistId: String, trackId: String) {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.removeTrackFromPlaylist(playlistId, trackId)
                .onSuccess { updatedPlaylist ->
                    _playlists.value = _playlists.value.map { playlist ->
                        if (playlist.id == playlistId) updatedPlaylist else playlist
                    }
                    if (_currentPlaylist.value?.id == playlistId) {
                        _currentPlaylist.value = updatedPlaylist
                    }
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to remove track from playlist"
                }
            
            _isLoading.value = false
        }
    }
    
    // Liked Tracks
    fun loadLikedTracks() {
        viewModelScope.launch {
            _isLoading.value = true
            
            musicRepository.getLikedTracks()
                .onSuccess { tracks ->
                    _likedTracks.value = tracks
                    _likedTrackIds.value = tracks.map { it.id }.toSet()
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to load liked tracks"
                }
            
            _isLoading.value = false
        }
    }
    
    fun likeTrack(trackId: String) {
        viewModelScope.launch {
            musicRepository.likeTrack(trackId)
                .onSuccess {
                    _likedTrackIds.value = _likedTrackIds.value + trackId
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to like track"
                }
        }
    }
    
    fun unlikeTrack(trackId: String) {
        viewModelScope.launch {
            musicRepository.unlikeTrack(trackId)
                .onSuccess {
                    _likedTrackIds.value = _likedTrackIds.value - trackId
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Failed to unlike track"
                }
        }
    }
    
    // Playback Control Methods
    fun updateCurrentTrack(mediaItem: MediaItem?) {
        viewModelScope.launch {
            if (mediaItem != null) {
                android.util.Log.d("MusicViewModel", "updateCurrentTrack called with: ${mediaItem.mediaMetadata.title}")
                android.util.Log.d("MusicViewModel", "MediaItem mediaId: ${mediaItem.mediaId}")
                android.util.Log.d("MusicViewModel", "MediaItem URI: ${mediaItem.localConfiguration?.uri}")
                val music = mediaItemToMusic(mediaItem)
                android.util.Log.d("MusicViewModel", "Converted to Music: ${music.title} - ID: ${music.id}")
                _currentTrack.value = music
            } else {
                android.util.Log.d("MusicViewModel", "updateCurrentTrack called with null mediaItem")
                _currentTrack.value = null
            }
        }
    }
    
    fun updatePlaybackState(isPlaying: Boolean) {
        _isPlaying.value = isPlaying
    }
    
    fun updatePlaybackProgress(progress: Float) {
        _playbackProgress.value = progress
    }
    
    fun updateCurrentPosition(position: Long) {
        _currentPosition.value = position
        // Update progress as a float between 0 and 1
        val duration = _duration.value
        if (duration > 0) {
            _playbackProgress.value = position.toFloat() / duration.toFloat()
        }
    }
    
    fun updateDuration(duration: Long) {
        _duration.value = duration
        // Update progress if we have a current position
        val currentPosition = _currentPosition.value
        if (duration > 0 && currentPosition > 0) {
            _playbackProgress.value = currentPosition.toFloat() / duration.toFloat()
        }
    }
    
    // Utility Methods
    fun createMediaItem(music: Music): MediaItem {
        val streamUrl = "${BuildConfig.BASE_URL}music/${music.id}/stream"
        return MediaItem.Builder()
            .setUri(streamUrl)
            .setMediaId(music.id)
            .setMediaMetadata(
                androidx.media3.common.MediaMetadata.Builder()
                    .setTitle(music.title)
                    .setArtist(music.artist)
                    .setAlbumTitle(music.album)
                    .setArtworkUri(null) // TODO: Add album art if available
                    .build()
            )
            .build()
    }
    
    private fun mediaItemToMusic(mediaItem: MediaItem): Music {
        return Music(
            id = mediaItem.mediaId ?: "",
            title = mediaItem.mediaMetadata.title?.toString() ?: "Unknown Title",
            artist = mediaItem.mediaMetadata.artist?.toString() ?: "Unknown Artist",
            album = mediaItem.mediaMetadata.albumTitle?.toString() ?: "Unknown Album",
            duration = 0, // We'll need to get this from the player
            filePath = mediaItem.localConfiguration?.uri?.toString() ?: "",
            genre = "",
            releaseDate = "",
            createdAt = "",
            updatedAt = ""
        )
    }
    
    fun clearError() {
        _errorMessage.value = null
    }
    
    fun clearSearchResults() {
        _searchResults.value = emptyList()
    }
    
    private fun addSampleData() {
        val sampleTracks = listOf(
            Music(
                id = "1",
                title = "Sample Song 1",
                artist = "Artist One",
                album = "Album One",
                duration = 180,
                filePath = "",
                genre = "Pop",
                releaseDate = "2023",
                createdAt = "2023-01-01",
                updatedAt = "2023-01-01"
            ),
            Music(
                id = "2",
                title = "Sample Song 2",
                artist = "Artist Two",
                album = "Album Two",
                duration = 210,
                filePath = "",
                genre = "Rock",
                releaseDate = "2023",
                createdAt = "2023-01-01",
                updatedAt = "2023-01-01"
            ),
            Music(
                id = "3",
                title = "Sample Song 3",
                artist = "Artist One",
                album = "Album One",
                duration = 195,
                filePath = "",
                genre = "Pop",
                releaseDate = "2023",
                createdAt = "2023-01-01",
                updatedAt = "2023-01-01"
            ),
            Music(
                id = "4",
                title = "Sample Song 4",
                artist = "Artist Three",
                album = "Album Three",
                duration = 240,
                filePath = "",
                genre = "Electronic",
                releaseDate = "2023",
                createdAt = "2023-01-01",
                updatedAt = "2023-01-01"
            ),
            Music(
                id = "5",
                title = "Sample Song 5",
                artist = "Artist Two",
                album = "Album Two",
                duration = 165,
                filePath = "",
                genre = "Rock",
                releaseDate = "2023",
                createdAt = "2023-01-01",
                updatedAt = "2023-01-01"
            )
        )
        
        _musicLibrary.value = sampleTracks
        
        // Create sample albums from the tracks
        val sampleAlbums = createAlbumsFromLibrary(sampleTracks)
        _albums.value = sampleAlbums
        
        // Create sample artists from the tracks
        val sampleArtists = createArtistsFromLibrary(sampleTracks)
        _artists.value = sampleArtists
        
        // Create sample playlists
        val samplePlaylists = listOf(
            Playlist(
                id = "1",
                name = "My Favorites",
                description = "My favorite songs",
                trackIds = listOf("1", "3", "5"),
                createdAt = "2023-01-01",
                updatedAt = "2023-01-01"
            ),
            Playlist(
                id = "2",
                name = "Rock Classics",
                description = "Best rock songs",
                trackIds = listOf("2", "5"),
                createdAt = "2023-01-01",
                updatedAt = "2023-01-01"
            )
        )
        _playlists.value = samplePlaylists
        
        android.util.Log.d("MusicViewModel", "Added ${sampleTracks.size} sample tracks")
        android.util.Log.d("MusicViewModel", "Created ${sampleAlbums.size} sample albums")
        android.util.Log.d("MusicViewModel", "Created ${sampleArtists.size} sample artists")
        android.util.Log.d("MusicViewModel", "Created ${samplePlaylists.size} sample playlists")
    }
    
    // Helper functions to create fallback data from music library
    private fun createAlbumsFromLibrary(musicLibrary: List<Music>): List<Album> {
        return musicLibrary
            .groupBy { it.album }
            .map { (albumName, tracks) ->
                val firstTrack = tracks.firstOrNull()
                Album(
                    name = albumName,
                    artist = firstTrack?.artist ?: "Unknown Artist",
                    year = try {
                        firstTrack?.releaseDate?.take(4)?.toInt() ?: 2023
                    } catch (e: Exception) {
                        2023
                    },
                    tracks = tracks
                )
            }
            .sortedBy { it.name }
    }
    
    private fun createArtistsFromLibrary(musicLibrary: List<Music>): List<String> {
        return musicLibrary
            .map { it.artist }
            .distinct()
            .sorted()
    }
    
    override fun onCleared() {
        super.onCleared()
        // Clean up service connection if needed
        musicService = null
        _isServiceConnected.value = false
    }
}
