package com.musicserver.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.musicserver.player.MusicService
import com.musicserver.viewmodel.MusicViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ArtistScreen(
    artistName: String,
    musicService: MusicService?,
    musicViewModel: MusicViewModel = hiltViewModel(),
    onNavigateToAlbum: (String) -> Unit,
    onNavigateBack: () -> Unit
) {
    val currentArtist by musicViewModel.currentArtist.collectAsState()
    val isLoading by musicViewModel.isLoading.collectAsState()
    val likedTrackIds by musicViewModel.likedTrackIds.collectAsState()
    
    LaunchedEffect(artistName) {
        musicViewModel.loadArtistTracks(artistName)
    }
    
    if (isLoading) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(Color.Black),
            contentAlignment = Alignment.Center
        ) {
            CircularProgressIndicator(color = Color.White)
        }
    } else {
        currentArtist?.let { artist ->
            LazyColumn(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Color.Black)
                    .padding(horizontal = 16.dp)
            ) {
                // Artist Header Section
                item {
                    ArtistHeaderSection(
                        artist = artist,
                        onPlayAll = { 
                            val mediaItems = artist.tracks.map { musicViewModel.createMediaItem(it) }
                            musicService?.setPlaylist(mediaItems)
                        },
                        onNavigateBack = onNavigateBack
                    )
                }
                
                // Popular Tracks Section
                item {
                    Spacer(modifier = Modifier.height(32.dp))
                    SectionTitle(title = "Popular Tracks")
                    Spacer(modifier = Modifier.height(16.dp))
                }
                
                items(artist.tracks) { music ->
                    val trackIndex = artist.tracks.indexOf(music) + 1
                    ArtistTrackItem(
                        music = music,
                        trackNumber = trackIndex,
                        isLiked = music.id in likedTrackIds,
                        onPlay = { 
                            musicService?.playTrack(musicViewModel.createMediaItem(music))
                        },
                        onLike = { 
                            if (music.id in likedTrackIds) {
                                musicViewModel.unlikeTrack(music.id)
                            } else {
                                musicViewModel.likeTrack(music.id)
                            }
                        },
                        onNavigateToAlbum = { onNavigateToAlbum(music.album) }
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                }
                
                // Albums Section
                item {
                    Spacer(modifier = Modifier.height(32.dp))
                    SectionTitle(title = "Albums")
                    Spacer(modifier = Modifier.height(16.dp))
                }
                
                item {
                    LazyVerticalGrid(
                        columns = GridCells.Fixed(2),
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(200.dp * ((artist.albums.size + 1) / 2)), // Estimate height
                        horizontalArrangement = Arrangement.spacedBy(16.dp),
                        verticalArrangement = Arrangement.spacedBy(16.dp)
                    ) {
                        items(artist.albums) { albumName ->
                            AlbumCard(
                                albumName = albumName,
                                tracks = artist.tracks.filter { it.album == albumName },
                                onClick = { onNavigateToAlbum(albumName) }
                            )
                        }
                    }
                    Spacer(modifier = Modifier.height(32.dp))
                }
            }
        } ?: run {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Color.Black),
                contentAlignment = Alignment.Center
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(
                        text = "Artist not found",
                        color = Color.White,
                        fontSize = 18.sp
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                    Button(
                        onClick = onNavigateBack,
                        colors = ButtonDefaults.buttonColors(
                            containerColor = Color(0xFF1DB954)
                        )
                    ) {
                        Icon(
                            Icons.Default.ArrowBack,
                            contentDescription = null,
                            modifier = Modifier.padding(end = 8.dp)
                        )
                        Text("Back to Library")
                    }
                }
            }
        }
    }
}

@Composable
private fun ArtistHeaderSection(
    artist: com.musicserver.data.models.ArtistTracksResponse,
    onPlayAll: () -> Unit,
    onNavigateBack: () -> Unit
) {
    Column {
        Spacer(modifier = Modifier.height(24.dp))
        
        // Back Button
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.Start
        ) {
            IconButton(
                onClick = onNavigateBack
            ) {
                Icon(
                    Icons.Default.ArrowBack,
                    contentDescription = "Back",
                    tint = Color(0xFFB3B3B3)
                )
            }
        }
        
        Spacer(modifier = Modifier.height(16.dp))
        
        // Artist Header
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(24.dp)
        ) {
            // Artist Avatar
            Box(
                modifier = Modifier
                    .size(120.dp)
                    .clip(RoundedCornerShape(8.dp))
                    .background(
                        Brush.linearGradient(
                            colors = listOf(
                                Color(0xFF9B59B6),
                                Color(0xFFC39BD3)
                            )
                        )
                    ),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    Icons.Default.Person,
                    contentDescription = null,
                    tint = Color.White,
                    modifier = Modifier.size(60.dp)
                )
            }
            
            // Artist Info
            Column(
                modifier = Modifier.weight(1f)
            ) {
                Text(
                    text = artist.artist.name,
                    color = Color.White,
                    fontSize = 32.sp,
                    fontWeight = FontWeight.Bold,
                    lineHeight = 36.sp
                )
                
                Spacer(modifier = Modifier.height(16.dp))
                
                // Stats
                Row(
                    horizontalArrangement = Arrangement.spacedBy(24.dp)
                ) {
                    StatItem(
                        value = artist.artist.trackCount.toString(),
                        label = "Tracks"
                    )
                    StatItem(
                        value = artist.artist.albumCount.toString(),
                        label = "Albums"
                    )
                }
                
                Spacer(modifier = Modifier.height(24.dp))
                
                // Play All Button
                Button(
                    onClick = onPlayAll,
                    modifier = Modifier
                        .height(48.dp)
                        .defaultMinSize(minWidth = 120.dp),
                    colors = ButtonDefaults.buttonColors(
                        containerColor = Color(0xFF1DB954)
                    ),
                    shape = RoundedCornerShape(24.dp)
                ) {
                    Icon(
                        Icons.Default.PlayArrow,
                        contentDescription = null,
                        modifier = Modifier.padding(end = 8.dp)
                    )
                    Text(
                        text = "Play All",
                        color = Color.White,
                        fontWeight = FontWeight.Bold
                    )
                }
            }
        }
    }
}

@Composable
private fun StatItem(
    value: String,
    label: String
) {
    Column {
        Text(
            text = value,
            color = Color.White,
            fontSize = 24.sp,
            fontWeight = FontWeight.Bold
        )
        Text(
            text = label,
            color = Color(0xFFB3B3B3),
            fontSize = 14.sp
        )
    }
}

@Composable
private fun SectionTitle(
    title: String
) {
    Text(
        text = title,
        color = Color.White,
        fontSize = 24.sp,
        fontWeight = FontWeight.Bold
    )
}

@Composable
private fun ArtistTrackItem(
    music: com.musicserver.data.models.Music,
    trackNumber: Int,
    isLiked: Boolean,
    onPlay: () -> Unit,
    onLike: () -> Unit,
    onNavigateToAlbum: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onPlay() },
        colors = CardDefaults.cardColors(
            containerColor = Color(0xFF181818)
        ),
        shape = RoundedCornerShape(8.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Track Number
            Text(
                text = "$trackNumber",
                color = Color(0xFFB3B3B3),
                fontSize = 14.sp,
                modifier = Modifier.width(30.dp)
            )
            
            // Track Info
            Column(
                modifier = Modifier.weight(1f)
            ) {
                Text(
                    text = music.title,
                    color = Color.White,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1
                )
                Spacer(modifier = Modifier.height(2.dp))
                Text(
                    text = "${music.album} • ${music.releaseDate.take(4)} • ${music.genre}",
                    color = Color(0xFFB3B3B3),
                    fontSize = 12.sp,
                    maxLines = 1
                )
            }
            
            // Duration
            Text(
                text = formatDuration(music.duration),
                color = Color(0xFFB3B3B3),
                fontSize = 14.sp,
                modifier = Modifier.padding(horizontal = 8.dp)
            )
            
            // Action Buttons
            Row {
                IconButton(
                    onClick = onNavigateToAlbum,
                    modifier = Modifier.size(32.dp)
                ) {
                    Icon(
                        Icons.Default.Album,
                        contentDescription = "Go to album",
                        tint = Color(0xFFB3B3B3),
                        modifier = Modifier.size(20.dp)
                    )
                }
                IconButton(
                    onClick = onLike,
                    modifier = Modifier.size(32.dp)
                ) {
                    Icon(
                        if (isLiked) Icons.Default.Favorite else Icons.Default.FavoriteBorder,
                        contentDescription = if (isLiked) "Unlike" else "Like",
                        tint = if (isLiked) Color(0xFF1DB954) else Color(0xFFB3B3B3),
                        modifier = Modifier.size(20.dp)
                    )
                }
            }
        }
    }
}

@Composable
private fun AlbumCard(
    albumName: String,
    tracks: List<com.musicserver.data.models.Music>,
    onClick: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() },
        colors = CardDefaults.cardColors(
            containerColor = Color(0xFF181818)
        ),
        shape = RoundedCornerShape(8.dp)
    ) {
        Column(
            modifier = Modifier.padding(12.dp)
        ) {
            // Album Art Placeholder
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .aspectRatio(1f)
                    .clip(RoundedCornerShape(8.dp))
                    .background(
                        Brush.linearGradient(
                            colors = listOf(
                                Color(0xFF4A90E2),
                                Color(0xFF7BB3F0)
                            )
                        )
                    ),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    Icons.Default.MusicNote,
                    contentDescription = null,
                    tint = Color.White,
                    modifier = Modifier.size(32.dp)
                )
            }
            
            Spacer(modifier = Modifier.height(12.dp))
            
            // Album Name
            Text(
                text = albumName,
                color = Color.White,
                fontSize = 14.sp,
                fontWeight = FontWeight.SemiBold,
                maxLines = 1
            )
            
            Spacer(modifier = Modifier.height(4.dp))
            
            // Album Year
            Text(
                text = tracks.firstOrNull()?.releaseDate?.take(4) ?: "",
                color = Color(0xFFB3B3B3),
                fontSize = 12.sp
            )
        }
    }
}

private fun formatDuration(seconds: Long): String {
    val minutes = seconds / 60
    val remainingSeconds = seconds % 60
    return "${minutes}:${remainingSeconds.toString().padStart(2, '0')}"
}
