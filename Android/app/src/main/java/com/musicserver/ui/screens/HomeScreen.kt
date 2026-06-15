package com.musicserver.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.musicserver.data.models.Music
import com.musicserver.player.MusicService
import com.musicserver.viewmodel.AuthViewModel
import com.musicserver.viewmodel.MusicViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    musicService: MusicService?,
    onServiceConnected: (MusicService) -> Unit,
    musicViewModel: MusicViewModel = hiltViewModel(),
    authViewModel: AuthViewModel = hiltViewModel(),
    onNavigateToLibrary: () -> Unit,
    onNavigateToSearch: () -> Unit,
    onNavigateToPlaylist: (String) -> Unit,
    onNavigateToAlbum: (String) -> Unit,
    onNavigateToArtist: (String) -> Unit,
    onNavigateToLikedSongs: () -> Unit,
    onLogout: () -> Unit
) {
    val musicLibrary by musicViewModel.musicLibrary.collectAsState()
    val isLoading by musicViewModel.isLoading.collectAsState()
    val currentUser by authViewModel.currentUser.collectAsState()
    val likedTrackIds by musicViewModel.likedTrackIds.collectAsState()
    val errorMessage by musicViewModel.errorMessage.collectAsState()
    
    LaunchedEffect(musicService) {
        musicService?.let { onServiceConnected(it) }
    }
    
    // Refresh music data when user becomes authenticated
    LaunchedEffect(currentUser) {
        if (currentUser != null) {
            android.util.Log.d("HomeScreen", "User authenticated, refreshing music library")
            musicViewModel.refreshMusicLibrary()
        }
    }
    
    // Show error message if any
    LaunchedEffect(errorMessage) {
        errorMessage?.let { message ->
            android.util.Log.e("HomeScreen", "Error: $message")
            // TODO: Show snackbar or toast to user
        }
    }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Music Server") },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    titleContentColor = MaterialTheme.colorScheme.onSurface,
                    actionIconContentColor = MaterialTheme.colorScheme.onSurface
                ),
                actions = {
                    IconButton(onClick = onNavigateToLibrary) {
                        Icon(Icons.Default.LibraryMusic, contentDescription = "Library")
                    }
                    IconButton(onClick = onNavigateToSearch) {
                        Icon(Icons.Default.Search, contentDescription = "Search")
                    }
                    IconButton(onClick = onLogout) {
                        Icon(Icons.Default.Logout, contentDescription = "Logout")
                    }
                }
            )
        },
        floatingActionButton = {
            FloatingActionButton(
                onClick = onNavigateToSearch,
                containerColor = MaterialTheme.colorScheme.primary,
                contentColor = MaterialTheme.colorScheme.onPrimary
            ) {
                Icon(Icons.Default.Search, contentDescription = "Search")
            }
        }
    ) { paddingValues ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            item {
                Text(
                    text = "Welcome back, ${currentUser?.username ?: "User"}!",
                    style = MaterialTheme.typography.headlineSmall
                )
            }
            
            item {
                Text(
                    text = "Recently played",
                    style = MaterialTheme.typography.titleLarge,
                    modifier = Modifier.padding(vertical = 8.dp)
                )
            }
            
            item {
                if (isLoading) {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(120.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        CircularProgressIndicator()
                    }
                } else {
                    LazyRow(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        contentPadding = PaddingValues(horizontal = 4.dp)
                    ) {
                        items(musicLibrary.take(6)) { music ->
                            RecentlyPlayedCard(
                                music = music,
                                onPlay = { 
                                    android.util.Log.d("HomeScreen", "Play clicked for track: ${music.title}")
                                    musicViewModel.playTrack(music)
                                },
                                onNavigateToAlbum = { onNavigateToAlbum(music.album) }
                            )
                        }
                    }
                }
            }
            
            item {
                Spacer(modifier = Modifier.height(24.dp))
                Text(
                    text = "Made for you",
                    style = MaterialTheme.typography.titleLarge,
                    modifier = Modifier.padding(vertical = 8.dp)
                )
            }
            
            item {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(120.dp),
                    contentAlignment = Alignment.Center
                ) {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(
                            text = "No personalized playlists available",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Text(
                            text = "Listen to more music to get personalized recommendations",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 4.dp)
                        )
                    }
                }
            }
            
            item {
                Spacer(modifier = Modifier.height(24.dp))
                Text(
                    text = "Recently Added",
                    style = MaterialTheme.typography.titleLarge,
                    modifier = Modifier.padding(vertical = 8.dp)
                )
            }
            
            if (isLoading) {
                item {
                    Box(
                        modifier = Modifier.fillMaxWidth(),
                        contentAlignment = Alignment.Center
                    ) {
                        CircularProgressIndicator()
                    }
                }
            } else {
                items(musicLibrary.take(10)) { music ->
                    MusicItem(
                        music = music,
                        isLiked = music.id in likedTrackIds,
                        onPlay = { 
                            android.util.Log.d("HomeScreen", "Play clicked for track: ${music.title}")
                            musicViewModel.playTrack(music)
                        },
                        onLike = { 
                            if (music.id in likedTrackIds) {
                                musicViewModel.unlikeTrack(music.id)
                            } else {
                                musicViewModel.likeTrack(music.id)
                            }
                        },
                        onNavigateToAlbum = { onNavigateToAlbum(music.album) },
                        onNavigateToArtist = { onNavigateToArtist(music.artist) }
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun MusicItem(
    music: Music,
    isLiked: Boolean,
    onPlay: () -> Unit,
    onLike: () -> Unit,
    onNavigateToAlbum: () -> Unit,
    onNavigateToArtist: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        ),
        onClick = onPlay
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = music.title,
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.onSurface,
                    maxLines = 1
                )
                Text(
                    text = music.artist,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1
                )
                Text(
                    text = music.album,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1
                )
            }
            
            Row {
                IconButton(onClick = onNavigateToArtist) {
                    Icon(
                        Icons.Default.Person,
                        contentDescription = "Go to artist",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                IconButton(onClick = onNavigateToAlbum) {
                    Icon(
                        Icons.Default.Album,
                        contentDescription = "Go to album",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                IconButton(onClick = onLike) {
                    Icon(
                        if (isLiked) Icons.Default.Favorite else Icons.Default.FavoriteBorder,
                        contentDescription = if (isLiked) "Unlike" else "Like",
                        tint = if (isLiked) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                IconButton(onClick = onPlay) {
                    Icon(
                        Icons.Default.PlayArrow,
                        contentDescription = "Play",
                        tint = MaterialTheme.colorScheme.primary
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun RecentlyPlayedCard(
    music: Music,
    onPlay: () -> Unit,
    onNavigateToAlbum: () -> Unit
) {
    Card(
        modifier = Modifier
            .width(160.dp)
            .height(200.dp),
        onClick = onPlay
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(12.dp)
        ) {
            // Album art placeholder
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(120.dp)
                    .padding(bottom = 8.dp),
                contentAlignment = Alignment.Center
            ) {
                Card(
                    modifier = Modifier.fillMaxSize(),
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.surfaceVariant
                    )
                ) {
                    Box(
                        modifier = Modifier.fillMaxSize(),
                        contentAlignment = Alignment.Center
                    ) {
                        Icon(
                            Icons.Default.Album,
                            contentDescription = "Album Art",
                            modifier = Modifier.size(48.dp),
                            tint = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            }
            
            Column(
                modifier = Modifier.fillMaxWidth()
            ) {
                Text(
                    text = music.title,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                    modifier = Modifier.padding(bottom = 2.dp)
                )
                Text(
                    text = music.artist,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1
                )
            }
        }
    }
}
