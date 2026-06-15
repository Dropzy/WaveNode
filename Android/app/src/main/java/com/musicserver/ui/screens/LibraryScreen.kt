package com.musicserver.ui.screens

import androidx.compose.animation.*
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.musicserver.player.MusicService
import com.musicserver.ui.theme.*
import com.musicserver.viewmodel.MusicViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LibraryScreen(
    musicService: MusicService?,
    musicViewModel: MusicViewModel = hiltViewModel(),
    onNavigateToPlaylist: (String) -> Unit,
    onNavigateToAlbum: (String) -> Unit,
    onNavigateToArtist: (String) -> Unit,
    onNavigateBack: () -> Unit
) {
    val musicLibrary by musicViewModel.musicLibrary.collectAsState()
    val albums by musicViewModel.albums.collectAsState()
    val artists by musicViewModel.artists.collectAsState()
    val playlists by musicViewModel.playlists.collectAsState()
    val isLoading by musicViewModel.isLoading.collectAsState()
    val likedTrackIds by musicViewModel.likedTrackIds.collectAsState()
    
    var selectedTab by remember { mutableStateOf("playlists") }
    var searchQuery by remember { mutableStateOf("") }
    var showCreatePlaylistDialog by remember { mutableStateOf(false) }
    
    val tabs = listOf(
        TabItem("playlists", Icons.Default.PlaylistPlay, "Playlists"),
        TabItem("albums", Icons.Default.Album, "Albums"),
        TabItem("artists", Icons.Default.Person, "Artists"),
        TabItem("tracks", Icons.Default.MusicNote, "Tracks"),
        TabItem("downloads", Icons.Default.Download, "Downloads")
    )
    
    Scaffold(
        topBar = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(Background)
                    .padding(16.dp)
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            Icons.Default.ArrowBack,
                            contentDescription = "Back",
                            tint = OnBackground
                        )
                    }
                    
                    Text(
                        text = "Your Library",
                        color = OnBackground,
                        fontSize = 24.sp,
                        fontWeight = FontWeight.Bold
                    )
                    
                    Spacer(modifier = Modifier.width(48.dp)) // Balance the back button
                }
                
                // Search Bar
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 16.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        Icons.Default.Search,
                        contentDescription = "Search",
                        tint = OnSurfaceVariant,
                        modifier = Modifier.padding(end = 12.dp)
                    )
                    TextField(
                        value = searchQuery,
                        onValueChange = { searchQuery = it },
                        placeholder = {
                            Text(
                                "Search in your library...",
                                color = OnSurfaceVariant
                            )
                        },
                        colors = TextFieldDefaults.colors(
                            unfocusedContainerColor = SurfaceVariant,
                            focusedContainerColor = SurfaceVariant,
                            unfocusedTextColor = OnBackground,
                            focusedTextColor = OnBackground,
                            unfocusedPlaceholderColor = OnSurfaceVariant,
                            focusedPlaceholderColor = OnSurfaceVariant,
                            unfocusedIndicatorColor = Color.Transparent,
                            focusedIndicatorColor = Color.Transparent
                        ),
                        shape = RoundedCornerShape(24.dp),
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(48.dp)
                    )
                }
                
                // Custom Tabs
                Column(
                    modifier = Modifier.padding(top = 16.dp)
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 4.dp),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        tabs.forEach { tab ->
                            Column(
                                modifier = Modifier
                                    .clickable { selectedTab = tab.id }
                                    .padding(vertical = 8.dp, horizontal = 6.dp)
                                    .weight(1f),
                                horizontalAlignment = Alignment.CenterHorizontally
                            ) {
                                Icon(
                                    tab.icon,
                                    contentDescription = tab.label,
                                    tint = if (selectedTab == tab.id) Primary else OnSurfaceVariant,
                                    modifier = Modifier.size(24.dp)
                                )
                                Spacer(modifier = Modifier.height(4.dp))
                                Text(
                                    text = tab.label,
                                    color = if (selectedTab == tab.id) Primary else OnSurfaceVariant,
                                    fontSize = 10.sp,
                                    fontWeight = if (selectedTab == tab.id) FontWeight.Bold else FontWeight.Normal,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis
                                )
                            }
                        }
                    }
                    
                    // Indicator line
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(2.dp)
                            .background(Primary)
                    )
                    
                    Divider(
                        color = SurfaceVariant,
                        thickness = 1.dp
                    )
                }
            }
        }
    ) { paddingValues ->
        if (isLoading) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Background)
                    .padding(paddingValues),
                contentAlignment = Alignment.Center
            ) {
                CircularProgressIndicator(color = Primary)
            }
        } else {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Background)
                    .padding(paddingValues)
                    .padding(16.dp)
            ) {
                when (selectedTab) {
                    "playlists" -> PlaylistsTab(
                        playlists = playlists,
                        onPlaylistClick = onNavigateToPlaylist,
                        onCreatePlaylist = { showCreatePlaylistDialog = true },
                        onPlayPlaylist = { playlist ->
                            // Get tracks for playlist and play them
                            val playlistTracks = musicLibrary.filter { it.id in playlist.trackIds }
                            if (playlistTracks.isNotEmpty()) {
                                musicViewModel.playTracks(playlistTracks)
                            }
                        }
                    )
                    "albums" -> {
                        val albumsToShow = if (albums.isEmpty()) {
                            android.util.Log.d("LibraryScreen", "Using fallback albums from music library, musicLibrary size: ${musicLibrary.size}")
                            createAlbumsFromLibrary(musicLibrary)
                        } else {
                            android.util.Log.d("LibraryScreen", "Using API albums, size: ${albums.size}")
                            albums
                        }
                        AlbumsTab(
                            albums = albumsToShow,
                            onAlbumClick = onNavigateToAlbum,
                            onPlayAlbum = { album ->
                                musicViewModel.playTracks(album.tracks)
                            }
                        )
                    }
                    "artists" -> {
                        val artistsToShow = if (artists.isEmpty()) {
                            android.util.Log.d("LibraryScreen", "Using fallback artists from music library, musicLibrary size: ${musicLibrary.size}")
                            createArtistsFromLibrary(musicLibrary)
                        } else {
                            android.util.Log.d("LibraryScreen", "Using API artists, size: ${artists.size}")
                            artists
                        }
                        ArtistsTab(
                            artists = artistsToShow,
                            musicLibrary = musicLibrary,
                            onArtistClick = onNavigateToArtist,
                            onPlayArtist = { artistName ->
                                val artistTracks = musicLibrary.filter { it.artist == artistName }
                                if (artistTracks.isNotEmpty()) {
                                    musicViewModel.playTracks(artistTracks)
                                }
                            }
                        )
                    }
                    "tracks" -> TracksTab(
                        tracks = musicLibrary,
                        searchQuery = searchQuery,
                        onPlayTrack = { track ->
                            musicViewModel.playTracks(listOf(track))
                        },
                        onPlayAll = { tracks ->
                            if (tracks.isNotEmpty()) {
                                musicViewModel.playTracks(tracks)
                            }
                        }
                    )
                    "downloads" -> DownloadsTab(
                        tracks = musicLibrary.take(3), // Simulate downloaded tracks
                        onPlayTrack = { track ->
                            musicViewModel.playTracks(listOf(track))
                        }
                    )
                }
            }
        }
    }
    
    // Create Playlist Dialog
    if (showCreatePlaylistDialog) {
        CreatePlaylistDialog(
            onDismiss = { showCreatePlaylistDialog = false },
            onCreate = { name, description ->
                // Handle playlist creation
                showCreatePlaylistDialog = false
            }
        )
    }
}

data class TabItem(
    val id: String,
    val icon: ImageVector,
    val label: String
)

@Composable
fun PlaylistsTab(
    playlists: List<com.musicserver.data.models.Playlist>,
    onPlaylistClick: (String) -> Unit,
    onCreatePlaylist: () -> Unit,
    onPlayPlaylist: (com.musicserver.data.models.Playlist) -> Unit = {}
) {
    Column {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 20.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "Playlists",
                color = OnBackground,
                fontSize = 20.sp,
                fontWeight = FontWeight.Bold
            )
            
            Button(
                onClick = onCreatePlaylist,
                colors = ButtonDefaults.buttonColors(
                    containerColor = Primary,
                    contentColor = OnPrimary
                ),
                shape = RoundedCornerShape(20.dp),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp)
            ) {
                Icon(
                    Icons.Default.Add,
                    contentDescription = "Create",
                    modifier = Modifier.size(16.dp)
                )
                Spacer(modifier = Modifier.width(6.dp))
                Text(
                    "Create Playlist",
                    fontSize = 13.sp,
                    fontWeight = FontWeight.SemiBold
                )
            }
        }
        
        if (playlists.isEmpty()) {
            EmptyState(
                icon = Icons.Default.PlaylistPlay,
                title = "No playlists yet",
                subtitle = "Create your first playlist to get started"
            )
        } else {
            LazyVerticalGrid(
                columns = GridCells.Fixed(2),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
                contentPadding = PaddingValues(bottom = 16.dp)
            ) {
                items(playlists) { playlist ->
                    PlaylistCard(
                        playlist = playlist,
                        onClick = { onPlaylistClick(playlist.id) },
                        onPlayClick = { onPlayPlaylist(playlist) }
                    )
                }
            }
        }
    }
}

@Composable
fun PlaylistCard(
    playlist: com.musicserver.data.models.Playlist,
    onClick: () -> Unit,
    onPlayClick: () -> Unit = {}
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(1f)
            .clickable { onClick() },
        colors = CardDefaults.cardColors(containerColor = Surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
        shape = RoundedCornerShape(8.dp)
    ) {
        Box {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(12.dp)
            ) {
                // Playlist Art
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .aspectRatio(1f)
                        .background(
                            Brush.linearGradient(
                                colors = listOf(
                                    Color(0xFFff6b6b),
                                    Color(0xFFff8e8e)
                                )
                            )
                        )
                        .clip(RoundedCornerShape(8.dp)),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        Icons.Default.PlaylistPlay,
                        contentDescription = "Playlist",
                        tint = Color.White,
                        modifier = Modifier.size(48.dp)
                    )
                }
                
                Spacer(modifier = Modifier.height(16.dp))
                
                // Playlist Info
                Text(
                    text = playlist.name,
                    color = OnBackground,
                    fontSize = 13.sp,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                
                Text(
                    text = playlist.description.ifBlank { "No description" },
                    color = OnSurfaceVariant,
                    fontSize = 11.sp,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
                
                Text(
                    text = "${playlist.trackIds.size} tracks",
                    color = OnSurfaceVariant,
                    fontSize = 10.sp
                )
            }
            
            // Play Button (shown on hover/tap)
            Box(
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(12.dp)
                    .size(40.dp)
                    .background(
                        Primary,
                        CircleShape
                    )
                    .clickable { onPlayClick() },
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    Icons.Default.PlayArrow,
                    contentDescription = "Play",
                    tint = OnPrimary,
                    modifier = Modifier.size(20.dp)
                )
            }
        }
    }
}

@Composable
fun AlbumsTab(
    albums: List<com.musicserver.data.models.Album>,
    onAlbumClick: (String) -> Unit,
    onPlayAlbum: (com.musicserver.data.models.Album) -> Unit = {}
) {
    Column {
        Text(
            text = "Albums",
            color = OnBackground,
            fontSize = 20.sp,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.padding(bottom = 20.dp)
        )
        
        if (albums.isEmpty()) {
            EmptyState(
                icon = Icons.Default.Album,
                title = "No albums found",
                subtitle = "Add some music to see your albums here"
            )
        } else {
            LazyVerticalGrid(
                columns = GridCells.Fixed(2),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
                contentPadding = PaddingValues(bottom = 16.dp)
            ) {
                items(albums) { album ->
                    AlbumCard(
                        album = album,
                        onClick = { onAlbumClick(album.name) },
                        onPlayClick = { onPlayAlbum(album) }
                    )
                }
            }
        }
    }
}

@Composable
fun AlbumCard(
    album: com.musicserver.data.models.Album,
    onClick: () -> Unit,
    onPlayClick: () -> Unit = {}
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() },
        colors = CardDefaults.cardColors(containerColor = Surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
        shape = RoundedCornerShape(8.dp)
    ) {
        Box {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(12.dp)
            ) {
                // Album Art
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .aspectRatio(1f)
                        .background(
                            Brush.linearGradient(
                                colors = listOf(
                                    Color(0xFF4a90e2),
                                    Color(0xFF7bb3f0)
                                )
                            )
                        )
                        .clip(RoundedCornerShape(8.dp)),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        Icons.Default.Album,
                        contentDescription = "Album",
                        tint = Color.White,
                        modifier = Modifier.size(48.dp)
                    )
                }
                
                Spacer(modifier = Modifier.height(12.dp))
                
                // Album Info
                Text(
                    text = album.name,
                    color = OnBackground,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
                
                Spacer(modifier = Modifier.height(4.dp))
                
                Text(
                    text = album.artist,
                    color = OnSurfaceVariant,
                    fontSize = 12.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                
                Spacer(modifier = Modifier.height(2.dp))
                
                Text(
                    text = "${album.tracks.size} tracks • ${album.year}",
                    color = OnSurfaceVariant,
                    fontSize = 11.sp
                )
            }
            
            // Play Button
            Box(
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(12.dp)
                    .size(40.dp)
                    .background(
                        Primary,
                        CircleShape
                    )
                    .clickable { onPlayClick() },
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    Icons.Default.PlayArrow,
                    contentDescription = "Play",
                    tint = OnPrimary,
                    modifier = Modifier.size(20.dp)
                )
            }
        }
    }
}

@Composable
fun ArtistsTab(
    artists: List<String>,
    musicLibrary: List<com.musicserver.data.models.Music>,
    onArtistClick: (String) -> Unit,
    onPlayArtist: (String) -> Unit = {}
) {
    // Calculate track and album counts for each artist
    val artistStats = remember(musicLibrary) {
        artists.map { artistName ->
            val artistTracks = musicLibrary.filter { it.artist == artistName }
            val uniqueAlbums = artistTracks.map { it.album }.distinct()
            ArtistStats(
                name = artistName,
                trackCount = artistTracks.size,
                albumCount = uniqueAlbums.size
            )
        }
    }
    
    Column {
        Text(
            text = "Artists",
            color = OnBackground,
            fontSize = 20.sp,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.padding(bottom = 20.dp)
        )
        
        if (artists.isEmpty()) {
            EmptyState(
                icon = Icons.Default.Person,
                title = "No artists found",
                subtitle = "Add some music to see your artists here"
            )
        } else {
            LazyVerticalGrid(
                columns = GridCells.Fixed(2),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
                contentPadding = PaddingValues(bottom = 16.dp)
            ) {
                items(artistStats) { artist ->
                    ArtistCard(
                        artistName = artist.name,
                        trackCount = artist.trackCount,
                        albumCount = artist.albumCount,
                        onClick = { onArtistClick(artist.name) },
                        onPlayClick = { onPlayArtist(artist.name) }
                    )
                }
            }
        }
    }
}

data class ArtistStats(
    val name: String,
    val trackCount: Int,
    val albumCount: Int
)

@Composable
fun ArtistCard(
    artistName: String,
    trackCount: Int,
    albumCount: Int,
    onClick: () -> Unit,
    onPlayClick: () -> Unit = {}
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() },
        colors = CardDefaults.cardColors(containerColor = Surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
        shape = RoundedCornerShape(8.dp)
    ) {
        Box {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(12.dp)
            ) {
                // Artist Art
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .aspectRatio(1f)
                        .background(
                            Brush.linearGradient(
                                colors = listOf(
                                    Color(0xFF9b59b6),
                                    Color(0xFFc39bd3)
                                )
                            )
                        )
                        .clip(RoundedCornerShape(8.dp)),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        Icons.Default.Person,
                        contentDescription = "Artist",
                        tint = Color.White,
                        modifier = Modifier.size(48.dp)
                    )
                }
                
                Spacer(modifier = Modifier.height(12.dp))
                
                // Artist Info
                Text(
                    text = artistName,
                    color = OnBackground,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
                
                Spacer(modifier = Modifier.height(4.dp))
                
                Text(
                    text = "$trackCount tracks • $albumCount albums",
                    color = OnSurfaceVariant,
                    fontSize = 11.sp
                )
            }
            
            // Play Button
            Box(
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(12.dp)
                    .size(40.dp)
                    .background(
                        Primary,
                        CircleShape
                    )
                    .clickable { onPlayClick() },
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    Icons.Default.PlayArrow,
                    contentDescription = "Play",
                    tint = OnPrimary,
                    modifier = Modifier.size(20.dp)
                )
            }
        }
    }
}

@Composable
fun TracksTab(
    tracks: List<com.musicserver.data.models.Music>,
    searchQuery: String,
    onPlayTrack: (com.musicserver.data.models.Music) -> Unit = {},
    onPlayAll: (List<com.musicserver.data.models.Music>) -> Unit = {}
) {
    val filteredTracks = tracks.filter { track ->
        track.title.contains(searchQuery, ignoreCase = true) ||
        track.artist.contains(searchQuery, ignoreCase = true) ||
        track.album.contains(searchQuery, ignoreCase = true)
    }
    
    Column {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 20.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "All Tracks",
                color = OnBackground,
                fontSize = 20.sp,
                fontWeight = FontWeight.Bold
            )
            
            if (filteredTracks.isNotEmpty()) {
                Button(
                    onClick = { onPlayAll(filteredTracks) },
                    colors = ButtonDefaults.buttonColors(
                        containerColor = Primary,
                        contentColor = OnPrimary
                    ),
                    shape = RoundedCornerShape(20.dp),
                    contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp)
                ) {
                    Icon(
                        Icons.Default.PlayArrow,
                        contentDescription = "Play",
                        modifier = Modifier.size(16.dp)
                    )
                    Spacer(modifier = Modifier.width(6.dp))
                    Text(
                        "Play All",
                        fontSize = 13.sp,
                        fontWeight = FontWeight.SemiBold
                    )
                }
            }
        }
        
        if (filteredTracks.isEmpty()) {
            EmptyState(
                icon = Icons.Default.MusicNote,
                title = "No tracks found",
                subtitle = "Add some music to your library"
            )
        } else {
            LazyColumn(
                verticalArrangement = Arrangement.spacedBy(6.dp),
                contentPadding = PaddingValues(bottom = 16.dp)
            ) {
                items(filteredTracks.size) { index ->
                    TrackItem(
                        track = filteredTracks[index],
                        trackNumber = index + 1,
                        onTrackClick = { onPlayTrack(filteredTracks[index]) },
                        onMoreClick = { /* Show context menu */ }
                    )
                }
            }
        }
    }
}

@Composable
fun TrackItem(
    track: com.musicserver.data.models.Music,
    trackNumber: Int,
    onTrackClick: () -> Unit,
    onMoreClick: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onTrackClick() },
        colors = CardDefaults.cardColors(containerColor = Surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
        shape = RoundedCornerShape(8.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Track Number
            Box(
                modifier = Modifier.width(40.dp),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = trackNumber.toString(),
                    color = OnSurfaceVariant,
                    fontSize = 12.sp
                )
            }
            
            // Track Cover Art
            Box(
                modifier = Modifier
                    .size(40.dp)
                    .background(
                        Brush.linearGradient(
                            colors = listOf(
                                Color(0xFF4a90e2),
                                Color(0xFF7bb3f0)
                            )
                        )
                    )
                    .clip(RoundedCornerShape(4.dp)),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    Icons.Default.MusicNote,
                    contentDescription = "Track",
                    tint = Color.White,
                    modifier = Modifier.size(16.dp)
                )
            }
            
            Spacer(modifier = Modifier.width(12.dp))
            
            // Track Info
            Column(
                modifier = Modifier.weight(1f)
            ) {
                Text(
                    text = track.title,
                    color = OnBackground,
                    fontSize = 13.sp,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    text = track.artist,
                    color = OnSurfaceVariant,
                    fontSize = 11.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }
            
            // Duration
            Text(
                text = formatDuration(track.duration.toInt()),
                color = OnSurfaceVariant,
                fontSize = 11.sp
            )
        }
    }
}

@Composable
fun DownloadsTab(
    tracks: List<com.musicserver.data.models.Music>,
    onPlayTrack: (com.musicserver.data.models.Music) -> Unit = {}
) {
    Column {
        Text(
            text = "Downloads",
            color = OnBackground,
            fontSize = 20.sp,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.padding(bottom = 20.dp)
        )
        
        if (tracks.isEmpty()) {
            EmptyState(
                icon = Icons.Default.Download,
                title = "No downloaded songs",
                subtitle = "Download songs to listen offline"
            )
        } else {
            LazyColumn(
                verticalArrangement = Arrangement.spacedBy(6.dp),
                contentPadding = PaddingValues(bottom = 16.dp)
            ) {
                items(tracks.size) { index ->
                    TrackItem(
                        track = tracks[index],
                        trackNumber = index + 1,
                        onTrackClick = { onPlayTrack(tracks[index]) },
                        onMoreClick = { /* Show context menu */ }
                    )
                }
            }
        }
    }
}

@Composable
fun EmptyState(
    icon: ImageVector,
    title: String,
    subtitle: String
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(60.dp, 20.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Icon(
            icon,
            contentDescription = null,
            tint = OnSurfaceVariant.copy(alpha = 0.5f),
            modifier = Modifier.size(64.dp)
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = title,
            color = OnSurfaceVariant,
            fontSize = 16.sp
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = subtitle,
            color = OnSurfaceVariant.copy(alpha = 0.8f),
            fontSize = 13.sp
        )
    }
}

@Composable
fun CreatePlaylistDialog(
    onDismiss: () -> Unit,
    onCreate: (String, String) -> Unit
) {
    var playlistName by remember { mutableStateOf("") }
    var playlistDescription by remember { mutableStateOf("") }
    
    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Text(
                text = "Create Playlist",
                color = OnBackground
            )
        },
        text = {
            Column {
                OutlinedTextField(
                    value = playlistName,
                    onValueChange = { playlistName = it },
                    label = { Text("Playlist Name *", color = OnSurfaceVariant) },
                    colors = OutlinedTextFieldDefaults.colors(
                        unfocusedContainerColor = SurfaceVariant,
                        focusedContainerColor = SurfaceVariant,
                        unfocusedTextColor = OnBackground,
                        focusedTextColor = OnBackground,
                        unfocusedBorderColor = Secondary,
                        focusedBorderColor = Primary,
                        unfocusedLabelColor = OnSurfaceVariant,
                        focusedLabelColor = Primary
                    ),
                    modifier = Modifier.fillMaxWidth()
                )
                
                Spacer(modifier = Modifier.height(16.dp))
                
                OutlinedTextField(
                    value = playlistDescription,
                    onValueChange = { playlistDescription = it },
                    label = { Text("Description", color = OnSurfaceVariant) },
                    colors = OutlinedTextFieldDefaults.colors(
                        unfocusedContainerColor = SurfaceVariant,
                        focusedContainerColor = SurfaceVariant,
                        unfocusedTextColor = OnBackground,
                        focusedTextColor = OnBackground,
                        unfocusedBorderColor = Secondary,
                        focusedBorderColor = Primary,
                        unfocusedLabelColor = OnSurfaceVariant,
                        focusedLabelColor = Primary
                    ),
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 3
                )
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    if (playlistName.isNotBlank()) {
                        onCreate(playlistName, playlistDescription)
                    }
                },
                colors = ButtonDefaults.buttonColors(
                    containerColor = Primary,
                    contentColor = OnPrimary
                ),
                enabled = playlistName.isNotBlank()
            ) {
                Text("Create")
            }
        },
        dismissButton = {
            TextButton(
                onClick = onDismiss,
                colors = ButtonDefaults.textButtonColors(
                    contentColor = OnSurfaceVariant
                )
            ) {
                Text("Cancel")
            }
        },
        containerColor = SurfaceVariant,
        titleContentColor = OnBackground,
        textContentColor = OnBackground
    )
}

private fun formatDuration(seconds: Int): String {
    val minutes = seconds / 60
    val remainingSeconds = seconds % 60
    return "$minutes:${remainingSeconds.toString().padStart(2, '0')}"
}

// Helper functions to create fallback data from music library
private fun createAlbumsFromLibrary(musicLibrary: List<com.musicserver.data.models.Music>): List<com.musicserver.data.models.Album> {
    return musicLibrary
        .groupBy { it.album }
        .map { (albumName, tracks) ->
            val firstTrack = tracks.firstOrNull()
            com.musicserver.data.models.Album(
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

private fun createArtistsFromLibrary(musicLibrary: List<com.musicserver.data.models.Music>): List<String> {
    return musicLibrary
        .map { it.artist }
        .distinct()
        .sorted()
}
