package com.musicserver.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
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
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.media3.common.util.UnstableApi
import com.musicserver.player.MusicService
import com.musicserver.ui.theme.*
import com.musicserver.viewmodel.MusicViewModel

@OptIn(ExperimentalMaterial3Api::class, UnstableApi::class)
@Composable
fun AlbumScreen(
    albumName: String,
    musicService: MusicService?,
    musicViewModel: MusicViewModel = hiltViewModel(),
    onNavigateToArtist: (String) -> Unit,
    onNavigateBack: () -> Unit
) {
    val currentAlbum by musicViewModel.currentAlbum.collectAsState()
    val isLoading by musicViewModel.isLoading.collectAsState()
    val likedTrackIds by musicViewModel.likedTrackIds.collectAsState()
    
    LaunchedEffect(albumName) {
        musicViewModel.loadAlbumTracks(albumName)
    }
    
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Background)
    ) {
        if (isLoading) {
            Box(
                modifier = Modifier.fillMaxSize(),
                contentAlignment = Alignment.Center
            ) {
                CircularProgressIndicator(color = Primary)
            }
        } else {
            currentAlbum?.let { album ->
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(bottom = 80.dp) // Space for play bar
                ) {
                    item {
                        AlbumHeader(
                            album = album.album,
                            tracks = album.tracks,
                            onPlayAlbum = { 
                                val mediaItems = album.tracks.map { musicViewModel.createMediaItem(it) }
                                musicService?.setPlaylist(mediaItems)
                            },
                            onNavigateBack = onNavigateBack
                        )
                    }
                    
                    item {
                        Spacer(modifier = Modifier.height(40.dp))
                        SectionTitle("Tracks")
                        Spacer(modifier = Modifier.height(20.dp))
                    }
                    
                    items(album.tracks) { music ->
                        AlbumTrackItem(
                            music = music,
                            trackNumber = album.tracks.indexOf(music) + 1,
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
                            onNavigateToArtist = { onNavigateToArtist(music.artist) }
                        )
                        Spacer(modifier = Modifier.height(4.dp))
                    }
                    
                    item {
                        Spacer(modifier = Modifier.height(40.dp))
                        AlbumDetailsSection(
                            album = album.album,
                            tracks = album.tracks
                        )
                        Spacer(modifier = Modifier.height(40.dp))
                    }
                }
            } ?: run {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(
                            text = "Album not found",
                            color = OnSurface,
                            style = MaterialTheme.typography.headlineSmall
                        )
                        Spacer(modifier = Modifier.height(16.dp))
                        Button(
                            onClick = onNavigateBack,
                            colors = ButtonDefaults.buttonColors(
                                containerColor = Color.Transparent
                            )
                        ) {
                            Icon(
                                Icons.Default.ArrowBack,
                                contentDescription = "Back",
                                tint = OnSurfaceVariant,
                                modifier = Modifier.padding(end = 8.dp)
                            )
                            Text(
                                "Back to Library",
                                color = OnSurfaceVariant
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun AlbumHeader(
    album: com.musicserver.data.models.AlbumInfo,
    tracks: List<com.musicserver.data.models.Music>,
    onPlayAlbum: () -> Unit,
    onNavigateBack: () -> Unit
) {
    val totalDuration = tracks.sumOf { it.duration }
    val uniqueGenres = tracks.map { it.genre }.distinct()
    
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(24.dp)
    ) {
        // Back button
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 24.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            IconButton(
                onClick = onNavigateBack,
                modifier = Modifier.padding(0.dp)
            ) {
                Icon(
                    Icons.Default.ArrowBack,
                    contentDescription = "Back",
                    tint = OnSurfaceVariant
                )
            }
            Text(
                text = "Back to Library",
                color = OnSurfaceVariant,
                fontSize = 14.sp,
                fontWeight = FontWeight.Medium
            )
        }
        
        // Album art and info
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(32.dp)
        ) {
            // Album art with overlay play button
            Box(
                modifier = Modifier
                    .size(320.dp)
                    .clip(RoundedCornerShape(8.dp))
                    .shadow(
                        elevation = 32.dp,
                        shape = RoundedCornerShape(8.dp)
                    )
                    .background(
                        brush = Brush.linearGradient(
                            colors = listOf(
                                Color(0xFF4A90E2),
                                Color(0xFF7BB3F0)
                            )
                        )
                    ),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    Icons.Default.Album,
                    contentDescription = "Album Art",
                    tint = Color.White,
                    modifier = Modifier.size(96.dp)
                )
                
                // Overlay play button
                Box(
                    modifier = Modifier
                        .align(Alignment.BottomEnd)
                        .padding(16.dp)
                        .size(56.dp)
                        .background(
                            Primary,
                            CircleShape
                        )
                        .shadow(
                            elevation = 12.dp,
                            shape = CircleShape
                        )
                        .clickable { onPlayAlbum() },
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        Icons.Default.PlayArrow,
                        contentDescription = "Play Album",
                        tint = OnPrimary,
                        modifier = Modifier.size(24.dp)
                    )
                }
            }
            
            // Album info
            Column(
                modifier = Modifier
                    .weight(1f)
                    .padding(top = 20.dp)
            ) {
                Text(
                    text = album.name,
                    color = OnSurface,
                    fontSize = 48.sp,
                    fontWeight = FontWeight.Bold,
                    lineHeight = 52.sp
                )
                Spacer(modifier = Modifier.height(12.dp))
                Text(
                    text = "by ${album.artist}",
                    color = OnSurfaceVariant,
                    fontSize = 18.sp
                )
                Spacer(modifier = Modifier.height(8.dp))
                
                // Album metadata
                Row(
                    horizontalArrangement = Arrangement.spacedBy(16.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = album.year.toString(),
                        color = OnSurfaceVariant,
                        fontSize = 14.sp
                    )
                    Text(
                        text = "•",
                        color = OnSurfaceVariant,
                        fontSize = 14.sp
                    )
                    Text(
                        text = "${tracks.size} tracks",
                        color = OnSurfaceVariant,
                        fontSize = 14.sp
                    )
                    Text(
                        text = "•",
                        color = OnSurfaceVariant,
                        fontSize = 14.sp
                    )
                    Text(
                        text = formatDuration(totalDuration),
                        color = OnSurfaceVariant,
                        fontSize = 14.sp
                    )
                }
            }
        }
    }
}

@Composable
private fun AlbumTrackItem(
    music: com.musicserver.data.models.Music,
    trackNumber: Int,
    isLiked: Boolean,
    onPlay: () -> Unit,
    onLike: () -> Unit,
    onNavigateToArtist: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 24.dp)
            .clickable { onPlay() },
        colors = CardDefaults.cardColors(
            containerColor = Surface
        ),
        shape = RoundedCornerShape(8.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Track number and info
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.weight(1f)
            ) {
                Text(
                    text = trackNumber.toString(),
                    color = OnSurfaceVariant,
                    fontSize = 14.sp,
                    modifier = Modifier.width(30.dp),
                    textAlign = androidx.compose.ui.text.style.TextAlign.Center
                )
                Spacer(modifier = Modifier.width(16.dp))
                Column(
                    modifier = Modifier.weight(1f)
                ) {
                    Text(
                        text = music.title,
                        color = OnSurface,
                        fontSize = 14.sp,
                        fontWeight = FontWeight.Medium,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                    Spacer(modifier = Modifier.height(2.dp))
                    Text(
                        text = music.artist,
                        color = OnSurfaceVariant,
                        fontSize = 12.sp,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
            }
            
            // Track metadata
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Genre
                Surface(
                    modifier = Modifier.clip(RoundedCornerShape(12.dp)),
                    color = SurfaceVariant
                ) {
                    Text(
                        text = music.genre,
                        color = OnSurfaceVariant,
                        fontSize = 11.sp,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
                    )
                }
                
                // Duration
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    Icon(
                        Icons.Default.Schedule,
                        contentDescription = "Duration",
                        tint = OnSurfaceVariant,
                        modifier = Modifier.size(12.dp)
                    )
                    Text(
                        text = formatDuration(music.duration),
                        color = OnSurfaceVariant,
                        fontSize = 12.sp
                    )
                }
            }
        }
    }
}

@Composable
private fun SectionTitle(title: String) {
    Text(
        text = title,
        color = OnSurface,
        fontSize = 24.sp,
        fontWeight = FontWeight.Bold,
        modifier = Modifier.padding(horizontal = 24.dp)
    )
}

@Composable
private fun AlbumDetailsSection(
    album: com.musicserver.data.models.AlbumInfo,
    tracks: List<com.musicserver.data.models.Music>
) {
    val totalDuration = tracks.sumOf { it.duration }
    val uniqueGenres = tracks.map { it.genre }.distinct()
    
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 24.dp),
        colors = CardDefaults.cardColors(
            containerColor = Surface
        ),
        shape = RoundedCornerShape(8.dp)
    ) {
        Column(
            modifier = Modifier.padding(24.dp)
        ) {
            SectionTitle("Album Details")
            Spacer(modifier = Modifier.height(20.dp))
            
            Column(
                verticalArrangement = Arrangement.spacedBy(20.dp)
            ) {
                DetailItem("Album", album.name)
                DetailItem("Artist", album.artist)
                DetailItem("Year", album.year.toString())
                DetailItem("Total Tracks", tracks.size.toString())
                DetailItem("Total Duration", formatDuration(totalDuration))
                DetailItem("Genres", uniqueGenres.joinToString(", "))
            }
        }
    }
}

@Composable
private fun DetailItem(label: String, value: String) {
    Column(
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        Text(
            text = label.uppercase(),
            color = OnSurfaceVariant,
            fontSize = 12.sp,
            fontWeight = FontWeight.Medium
        )
        Text(
            text = value,
            color = OnSurface,
            fontSize = 14.sp,
            fontWeight = FontWeight.Medium
        )
    }
}

private fun formatDuration(seconds: Long): String {
    val minutes = seconds / 60
    val remainingSeconds = seconds % 60
    return "${minutes}:${remainingSeconds.toString().padStart(2, '0')}"
}
