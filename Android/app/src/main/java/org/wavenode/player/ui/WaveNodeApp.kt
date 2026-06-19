package org.wavenode.player.ui

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.media.AudioDeviceInfo
import android.media.AudioManager
import android.os.Build
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.automirrored.filled.PlaylistPlay
import androidx.compose.material.icons.automirrored.filled.QueueMusic
import androidx.compose.material.icons.automirrored.filled.VolumeUp
import androidx.compose.material.icons.filled.Album
import androidx.compose.material.icons.filled.Bluetooth
import androidx.compose.material.icons.filled.Computer
import androidx.compose.material.icons.filled.Devices
import androidx.compose.material.icons.filled.Groups
import androidx.compose.material.icons.filled.Headphones
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.LibraryMusic
import androidx.compose.material.icons.filled.Radio
import androidx.compose.material.icons.filled.MusicNote
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.PhoneAndroid
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Repeat
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Shuffle
import androidx.compose.material.icons.filled.SkipNext
import androidx.compose.material.icons.filled.SkipPrevious
import androidx.compose.material.icons.filled.Wifi
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import kotlin.random.Random
import java.time.Instant
import org.wavenode.player.AppState
import org.wavenode.player.LibraryDetail
import org.wavenode.player.data.Album
import org.wavenode.player.data.Artist
import org.wavenode.player.data.DiscoveredServer
import org.wavenode.player.data.Playlist
import org.wavenode.player.data.PluginHomeRow
import org.wavenode.player.data.PluginRowItem
import org.wavenode.player.data.SavedSession
import org.wavenode.player.data.Track
import org.wavenode.player.data.UserSession
import org.wavenode.player.playback.PlayerState
import org.wavenode.player.playback.WaveRepeatMode

private enum class WaveTab(val label: String, val icon: ImageVector) {
    Home("Home", Icons.Default.Home),
    Tracks("Tracks", Icons.Default.LibraryMusic),
    Playlists("Playlists", Icons.AutoMirrored.Filled.PlaylistPlay),
    Albums("Albums", Icons.Default.Album),
    Artists("Artists", Icons.Default.Groups),
    Radio("Radio", Icons.Default.Radio),
}

private enum class TrackSortOption(val label: String) {
    RecentlyUploaded("Recently uploaded"),
    Title("Title"),
    Artist("Artist"),
    Album("Album"),
    Duration("Duration"),
}

@Composable
fun WaveNodeApp(
    state: AppState,
    playerState: PlayerState,
    onLogin: (String, String, String) -> Unit,
    onDiscoverServers: () -> Unit,
    onRefresh: () -> Unit,
    onLogout: () -> Unit,
    onPlayFromHere: (Track, List<Track>) -> Unit,
    onPlayQueueTrack: (Track) -> Unit,
    onOpenAlbum: (Album) -> Unit,
    onOpenArtist: (Artist) -> Unit,
    onOpenPlaylist: (Playlist) -> Unit,
    onCloseDetail: () -> Unit,
    onTogglePlayPause: () -> Unit,
    onToggleShuffle: () -> Unit,
    onCycleRepeatMode: () -> Unit,
    onSkipNext: () -> Unit,
    onSkipPrevious: () -> Unit,
    onSeekTo: (Long) -> Unit,
    onRefreshConnectSessions: () -> Unit,
    onConnectPlaybackTo: (String) -> Unit,
    onAddTrackToPlaylist: (Track, Playlist) -> Unit,
    trackArtworkUrl: (Track) -> String?,
    albumArtworkUrl: (Album) -> String?,
    artistArtworkUrl: (Artist) -> String?,
) {
    if (state.session == null) {
        LoginScreen(
            isLoading = state.isLoading,
            error = state.error,
            discoveredServers = state.discoveredServers,
            isDiscoveringServers = state.isDiscoveringServers,
            onDiscoverServers = onDiscoverServers,
            onLogin = onLogin,
        )
    } else {
        MainShell(
            state = state,
            playerState = playerState,
            onRefresh = onRefresh,
            onLogout = onLogout,
            onPlayFromHere = onPlayFromHere,
            onPlayQueueTrack = onPlayQueueTrack,
            onOpenAlbum = onOpenAlbum,
            onOpenArtist = onOpenArtist,
            onOpenPlaylist = onOpenPlaylist,
            onCloseDetail = onCloseDetail,
            onTogglePlayPause = onTogglePlayPause,
            onToggleShuffle = onToggleShuffle,
            onCycleRepeatMode = onCycleRepeatMode,
            onSkipNext = onSkipNext,
            onSkipPrevious = onSkipPrevious,
            onSeekTo = onSeekTo,
            onRefreshConnectSessions = onRefreshConnectSessions,
            onConnectPlaybackTo = onConnectPlaybackTo,
            onAddTrackToPlaylist = onAddTrackToPlaylist,
            trackArtworkUrl = trackArtworkUrl,
            albumArtworkUrl = albumArtworkUrl,
            artistArtworkUrl = artistArtworkUrl,
        )
    }
}

@Composable
private fun LoginScreen(
    isLoading: Boolean,
    error: String?,
    discoveredServers: List<DiscoveredServer>,
    isDiscoveringServers: Boolean,
    onDiscoverServers: () -> Unit,
    onLogin: (String, String, String) -> Unit,
) {
    var serverUrl by remember { mutableStateOf("") }
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }

    LaunchedEffect(discoveredServers) {
        if (serverUrl.isBlank() && discoveredServers.size == 1) {
            serverUrl = discoveredServers.first().url
        }
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(WaveBackground)
            .padding(24.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            verticalArrangement = Arrangement.spacedBy(14.dp),
            modifier = Modifier.fillMaxWidth(),
        ) {
            WaveNodeMark()
            Text(
                text = "WaveNode",
                color = WaveText,
                fontSize = 38.sp,
                fontWeight = FontWeight.Black,
            )
            Text(
                text = "Your self-hosted music player, native on Android.",
                color = WaveSubtle,
                fontSize = 15.sp,
            )
            Spacer(modifier = Modifier.height(10.dp))
            OutlinedTextField(
                value = serverUrl,
                onValueChange = { serverUrl = it },
                label = { Text("Server address") },
                placeholder = { Text("http://192.168.1.70:8080") },
                leadingIcon = { Icon(Icons.Default.Wifi, contentDescription = null) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            ServerDiscoveryPanel(
                servers = discoveredServers,
                isDiscovering = isDiscoveringServers,
                onRefresh = onDiscoverServers,
                onSelect = { serverUrl = it.url },
            )
            OutlinedTextField(
                value = username,
                onValueChange = { username = it },
                label = { Text("Username") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = password,
                onValueChange = { password = it },
                label = { Text("Password") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                modifier = Modifier.fillMaxWidth(),
            )
            Button(
                onClick = { onLogin(serverUrl, username, password) },
                enabled = !isLoading && serverUrl.isNotBlank() && username.isNotBlank() && password.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(if (isLoading) "Connecting..." else "Sign In")
            }
            if (error != null) {
                Text(text = error, color = MaterialTheme.colorScheme.error)
            }
        }
    }
}

@Composable
private fun ServerDiscoveryPanel(
    servers: List<DiscoveredServer>,
    isDiscovering: Boolean,
    onRefresh: () -> Unit,
    onSelect: (DiscoveredServer) -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(
                text = if (isDiscovering) "Searching for WaveNode..." else "Nearby WaveNode servers",
                color = WaveSubtle,
                fontSize = 13.sp,
                modifier = Modifier.weight(1f),
            )
            OutlinedButton(
                onClick = onRefresh,
                enabled = !isDiscovering,
            ) {
                Text(if (isDiscovering) "Searching" else "Scan")
            }
        }

        servers.forEach { server ->
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { onSelect(server) },
                colors = CardDefaults.cardColors(containerColor = WaveSurface),
                shape = RoundedCornerShape(10.dp),
            ) {
                Row(
                    modifier = Modifier.padding(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    WaveNodeMark(size = 34)
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = server.name,
                            color = WaveText,
                            fontWeight = FontWeight.SemiBold,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                        Text(
                            text = server.url,
                            color = WaveSubtle,
                            fontSize = 12.sp,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
            }
        }

        if (!isDiscovering && servers.isEmpty()) {
            Text(
                text = "No server found yet. You can still enter the address manually.",
                color = WaveSubtle,
                fontSize = 12.sp,
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun MainShell(
    state: AppState,
    playerState: PlayerState,
    onRefresh: () -> Unit,
    onLogout: () -> Unit,
    onPlayFromHere: (Track, List<Track>) -> Unit,
    onPlayQueueTrack: (Track) -> Unit,
    onOpenAlbum: (Album) -> Unit,
    onOpenArtist: (Artist) -> Unit,
    onOpenPlaylist: (Playlist) -> Unit,
    onCloseDetail: () -> Unit,
    onTogglePlayPause: () -> Unit,
    onToggleShuffle: () -> Unit,
    onCycleRepeatMode: () -> Unit,
    onSkipNext: () -> Unit,
    onSkipPrevious: () -> Unit,
    onSeekTo: (Long) -> Unit,
    onRefreshConnectSessions: () -> Unit,
    onConnectPlaybackTo: (String) -> Unit,
    onAddTrackToPlaylist: (Track, Playlist) -> Unit,
    trackArtworkUrl: (Track) -> String?,
    albumArtworkUrl: (Album) -> String?,
    artistArtworkUrl: (Artist) -> String?,
) {
    var activeTab by remember { mutableStateOf(WaveTab.Home) }
    var searchQuery by remember { mutableStateOf("") }
    var showQueue by remember { mutableStateOf(false) }
    var showNowPlaying by remember { mutableStateOf(false) }
    var showAccountSheet by remember { mutableStateOf(false) }

    val filteredTracks = remember(state.tracks, searchQuery) {
        state.tracks.filterByQuery(searchQuery) { listOf(title, artist, album) }
    }
    val filteredAlbums = remember(state.albums, searchQuery) {
        state.albums.filterByQuery(searchQuery) { listOf(name, artist, year.takeIf { it > 0 }?.toString().orEmpty()) }
    }
    val filteredArtists = remember(state.artists, searchQuery) {
        state.artists.filterByQuery(searchQuery) { listOf(name) }
    }
    val filteredPlaylists = remember(state.playlists, searchQuery) {
        state.playlists.filterByQuery(searchQuery) { listOf(name, description, type) }
    }
    val filteredPluginRows = remember(state.pluginRows, searchQuery) {
        if (searchQuery.isBlank()) {
            state.pluginRows
        } else {
            state.pluginRows.mapNotNull { row ->
                val items = row.items.filterByQuery(searchQuery) { listOf(title, subtitle, description) }
                if (items.isEmpty()) null else row.copy(items = items)
            }
        }
    }

    LaunchedEffect(state.session?.token) {
        if (state.tracks.isEmpty() && !state.isLoading) {
            onRefresh()
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                navigationIcon = {
                    AccountMenuButton(
                        session = state.session,
                        onOpenAccount = { showAccountSheet = true },
                        onRefresh = onRefresh,
                        onLogout = onLogout,
                    )
                },
                title = {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        Column {
                            Text("WaveNode", fontWeight = FontWeight.Black)
                            Text(
                                text = state.session?.serverUrl.orEmpty(),
                                color = WaveSubtle,
                                fontSize = 12.sp,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                    }
                },
                actions = {
                    IconButton(onClick = onRefresh) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = WaveBackground,
                    titleContentColor = WaveText,
                    actionIconContentColor = WaveText,
                ),
            )
        },
        bottomBar = {
            Column {
                MiniPlayer(
                    playerState = playerState,
                    onTogglePlayPause = onTogglePlayPause,
                    onSkipNext = onSkipNext,
                    onSkipPrevious = onSkipPrevious,
                    onOpenQueue = { showQueue = true },
                    onOpenPlayer = { showNowPlaying = true },
                    artworkUrl = playerState.currentTrack?.let(trackArtworkUrl),
                )
                NavigationBar(
                    containerColor = WaveSurface,
                    contentColor = WaveText,
                    modifier = Modifier.navigationBarsPadding(),
                ) {
                    WaveTab.entries.forEach { tab ->
                        NavigationBarItem(
                            selected = activeTab == tab,
                            onClick = {
                                activeTab = tab
                                if (state.activeDetail != null) {
                                    onCloseDetail()
                                }
                            },
                            icon = { Icon(tab.icon, contentDescription = tab.label) },
                            label = { Text(tab.label) },
                        )
                    }
                }
            }
        },
        containerColor = WaveBackground,
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize(),
        ) {
            if (state.activeDetail == null) {
                OutlinedTextField(
                    value = searchQuery,
                    onValueChange = { searchQuery = it },
                    leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
                    placeholder = { Text("Search WaveNode") },
                    singleLine = true,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 8.dp),
                )
            }

            if (state.isLoading || state.isDetailLoading) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(20.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator()
                }
            }

            val visibleError = state.detailError ?: state.error
            if (visibleError != null) {
                Text(
                    text = visibleError,
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                )
            }

            val detail = state.activeDetail
            if (detail != null) {
                LibraryDetailScreen(
                    detail = detail,
                    playerState = playerState,
                    onBack = onCloseDetail,
                    onPlayFromHere = onPlayFromHere,
                    playlists = state.playlists,
                    onAddTrackToPlaylist = onAddTrackToPlaylist,
                    trackArtworkUrl = trackArtworkUrl,
                    albumArtworkUrl = albumArtworkUrl,
                    artistArtworkUrl = artistArtworkUrl,
                    onOpenAlbum = onOpenAlbum,
                )
            } else when (activeTab) {
                WaveTab.Home -> HomeScreen(
                    state = state,
                    recentTracks = filteredTracks.take(8),
                    featuredAlbums = filteredAlbums.take(10),
                    pluginRows = filteredPluginRows,
                    onPlayFromHere = onPlayFromHere,
                    onOpenAlbum = onOpenAlbum,
                    playlists = state.playlists,
                    onAddTrackToPlaylist = onAddTrackToPlaylist,
                    trackArtworkUrl = trackArtworkUrl,
                    albumArtworkUrl = albumArtworkUrl,
                )
                WaveTab.Tracks -> TracksScreen(
                    tracks = filteredTracks,
                    playerState = playerState,
                    onPlayFromHere = onPlayFromHere,
                    playlists = state.playlists,
                    onAddTrackToPlaylist = onAddTrackToPlaylist,
                    trackArtworkUrl = trackArtworkUrl,
                    emptyMessage = emptyMessage("tracks", searchQuery),
                )
                WaveTab.Albums -> AlbumsScreen(
                    albums = filteredAlbums,
                    tracks = state.tracks,
                    albumArtworkUrl = albumArtworkUrl,
                    trackArtworkUrl = trackArtworkUrl,
                    onOpenAlbum = onOpenAlbum,
                    emptyMessage = emptyMessage("albums", searchQuery),
                )
                WaveTab.Playlists -> PlaylistsScreen(
                    playlists = filteredPlaylists,
                    onOpenPlaylist = onOpenPlaylist,
                    emptyMessage = emptyMessage("playlists", searchQuery),
                )
                WaveTab.Artists -> ArtistsScreen(
                    artists = filteredArtists,
                    tracks = state.tracks,
                    artistArtworkUrl = artistArtworkUrl,
                    trackArtworkUrl = trackArtworkUrl,
                    onOpenArtist = onOpenArtist,
                    emptyMessage = emptyMessage("artists", searchQuery),
                )
                WaveTab.Radio -> RadioScreen(
                    rows = filteredPluginRows,
                    onPlayFromHere = onPlayFromHere,
                    emptyMessage = if (searchQuery.isBlank()) {
                        "No enabled radio plugins found."
                    } else {
                        "No radio stations match \"$searchQuery\"."
                    },
                )
            }
        }
    }

    if (showQueue) {
        QueueSheet(
            playerState = playerState,
            onDismiss = { showQueue = false },
            onPlayTrack = onPlayQueueTrack,
            playlists = state.playlists,
            onAddTrackToPlaylist = onAddTrackToPlaylist,
            trackArtworkUrl = trackArtworkUrl,
        )
    }

    if (showNowPlaying) {
        NowPlayingSheet(
            playerState = playerState,
            artworkUrl = playerState.currentTrack?.let(trackArtworkUrl),
            onDismiss = { showNowPlaying = false },
            onTogglePlayPause = onTogglePlayPause,
            onSkipNext = onSkipNext,
            onSkipPrevious = onSkipPrevious,
            onSeekTo = onSeekTo,
            onOpenQueue = { showQueue = true },
            onToggleShuffle = onToggleShuffle,
            onCycleRepeatMode = onCycleRepeatMode,
            connectSessions = state.connectSessions,
            currentSessionId = state.currentSessionId,
            connectedPlaybackSessionId = state.connectedPlaybackSessionId,
            connectedPlaybackDeviceName = state.connectedPlaybackDeviceName,
            isLoadingConnectSessions = state.isLoadingConnectSessions,
            connectMessage = state.connectMessage,
            onRefreshConnectSessions = onRefreshConnectSessions,
            onConnectPlaybackTo = onConnectPlaybackTo,
        )
    }

    if (showAccountSheet) {
        AccountSettingsSheet(
            session = state.session,
            onDismiss = { showAccountSheet = false },
            onLogout = onLogout,
        )
    }
}

@Composable
private fun AccountMenuButton(
    session: SavedSession?,
    onOpenAccount: () -> Unit,
    onRefresh: () -> Unit,
    onLogout: () -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    Box {
        IconButton(onClick = { expanded = true }) {
            AccountAvatar(username = session?.username.orEmpty())
        }
        DropdownMenu(
            expanded = expanded,
            onDismissRequest = { expanded = false },
            modifier = Modifier.background(WaveSurface),
        ) {
            DropdownMenuItem(
                text = { Text("Account settings", color = WaveText) },
                leadingIcon = { Icon(Icons.Default.Person, contentDescription = null, tint = WaveAccent) },
                onClick = {
                    expanded = false
                    onOpenAccount()
                },
            )
            DropdownMenuItem(
                text = { Text("Refresh library", color = WaveText) },
                leadingIcon = { Icon(Icons.Default.Refresh, contentDescription = null, tint = WaveSubtle) },
                onClick = {
                    expanded = false
                    onRefresh()
                },
            )
            DropdownMenuItem(
                text = { Text("Sign out", color = WaveText) },
                leadingIcon = { Icon(Icons.AutoMirrored.Filled.Logout, contentDescription = null, tint = WaveSubtle) },
                onClick = {
                    expanded = false
                    onLogout()
                },
            )
        }
    }
}

@Composable
private fun AccountAvatar(username: String) {
    val initial = username.trim().firstOrNull()?.uppercaseChar()?.toString() ?: "?"
    Surface(
        modifier = Modifier.size(38.dp),
        shape = CircleShape,
        color = WaveAccent,
        contentColor = WaveBackground,
    ) {
        Box(contentAlignment = Alignment.Center) {
            Text(
                text = initial,
                fontWeight = FontWeight.Black,
                fontSize = 18.sp,
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AccountSettingsSheet(
    session: SavedSession?,
    onDismiss: () -> Unit,
    onLogout: () -> Unit,
) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        containerColor = WaveSurface,
        contentColor = WaveText,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .navigationBarsPadding()
                .padding(24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                AccountAvatar(username = session?.username.orEmpty())
                Column {
                    Text("Account settings", fontSize = 22.sp, fontWeight = FontWeight.Black)
                    Text(session?.username.orEmpty().ifBlank { "Signed in" }, color = WaveSubtle)
                }
            }
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text("Server", color = WaveSubtle, fontSize = 12.sp, fontWeight = FontWeight.Bold)
                Text(
                    text = session?.serverUrl.orEmpty(),
                    color = WaveText,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            OutlinedButton(
                onClick = {
                    onDismiss()
                    onLogout()
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Icon(Icons.AutoMirrored.Filled.Logout, contentDescription = null)
                Spacer(modifier = Modifier.size(8.dp))
                Text("Sign out")
            }
            Spacer(modifier = Modifier.height(12.dp))
        }
    }
}

@Composable
private fun HomeScreen(
    state: AppState,
    recentTracks: List<Track>,
    featuredAlbums: List<Album>,
    pluginRows: List<PluginHomeRow>,
    onPlayFromHere: (Track, List<Track>) -> Unit,
    onOpenAlbum: (Album) -> Unit,
    playlists: List<Playlist>,
    onAddTrackToPlaylist: (Track, Playlist) -> Unit,
    trackArtworkUrl: (Track) -> String?,
    albumArtworkUrl: (Album) -> String?,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item {
            Column(
                modifier = Modifier.padding(horizontal = 16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text("Home", color = WaveText, fontSize = 28.sp, fontWeight = FontWeight.Black)
                Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    StatTile("Tracks", state.tracks.size.toString(), Modifier.weight(1f))
                    StatTile("Albums", state.albums.size.toString(), Modifier.weight(1f))
                    StatTile("Playlists", state.playlists.size.toString(), Modifier.weight(1f))
                }
            }
        }

        pluginRows.forEach { row ->
            item {
                SectionHeader(row.title.ifBlank { "Radio" }, row.subtitle.takeIf { it.isNotBlank() })
                LazyRow(
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.padding(start = 16.dp),
                ) {
                    items(row.items, key = { "${row.pluginId}:${it.id}" }) { item ->
                        val rowTracks = row.items.map { pluginItemToTrack(row.pluginId, it) }
                        RadioCard(
                            item = item,
                            onClick = { onPlayFromHere(pluginItemToTrack(row.pluginId, item), rowTracks) },
                        )
                    }
                }
            }
        }

        if (featuredAlbums.isNotEmpty()) {
            item {
                SectionHeader("Albums")
                LazyRow(
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.padding(start = 16.dp),
                ) {
                    items(featuredAlbums, key = { it.id.ifBlank { it.name } }) { album ->
                        AlbumCard(
                            album = album,
                            artworkUrl = albumArtworkFor(album, state.tracks, albumArtworkUrl, trackArtworkUrl),
                            onClick = { onOpenAlbum(album) },
                        )
                    }
                }
            }
        }

        item { SectionHeader("Recently Uploaded") }
        if (recentTracks.isEmpty()) {
            item { EmptyLibraryMessage("No tracks available yet.") }
        } else {
            items(recentTracks, key = { it.id }) { track ->
                TrackRow(
                    track = track,
                    isCurrent = false,
                    artworkUrl = trackArtworkUrl(track),
                    playlists = playlists,
                    onClick = { onPlayFromHere(track, recentTracks) },
                    onAddToPlaylist = onAddTrackToPlaylist,
                )
            }
        }
    }
}

@Composable
private fun PlaylistsScreen(
    playlists: List<Playlist>,
    onOpenPlaylist: (Playlist) -> Unit,
    emptyMessage: String,
) {
    LazyColumn(modifier = Modifier.fillMaxSize()) {
        item { SectionHeader("Playlists", "${playlists.size}") }
        if (playlists.isEmpty()) {
            item { EmptyLibraryMessage(emptyMessage) }
        }
        items(playlists, key = { it.id.ifBlank { it.name } }) { playlist ->
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { onOpenPlaylist(playlist) }
                    .padding(horizontal = 16.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Box(
                    modifier = Modifier
                        .size(54.dp)
                        .clip(RoundedCornerShape(8.dp))
                        .background(WaveSurfaceRaised),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.AutoMirrored.Filled.PlaylistPlay, contentDescription = null, tint = WaveAccent)
                }
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = playlist.name,
                        color = WaveText,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Text(
                        text = playlistSubtitle(playlist),
                        color = WaveSubtle,
                        fontSize = 13.sp,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}

@Composable
private fun RadioScreen(
    rows: List<PluginHomeRow>,
    onPlayFromHere: (Track, List<Track>) -> Unit,
    emptyMessage: String,
) {
    LazyColumn(modifier = Modifier.fillMaxSize(), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        item { SectionHeader("Radio") }
        if (rows.isEmpty()) {
            item { EmptyLibraryMessage(emptyMessage) }
        }
        rows.forEach { row ->
            item { SectionHeader(row.title.ifBlank { "Stations" }, row.subtitle.takeIf { it.isNotBlank() }) }
            items(row.items, key = { "${row.pluginId}:${it.id}" }) { item ->
                val rowTracks = row.items.map { pluginItemToTrack(row.pluginId, it) }
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { onPlayFromHere(pluginItemToTrack(row.pluginId, item), rowTracks) }
                        .padding(horizontal = 16.dp, vertical = 9.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Artwork(url = item.imageUrl.takeIf { it.isNotBlank() }, size = 54)
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = item.title,
                            color = WaveText,
                            fontWeight = FontWeight.SemiBold,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                        Text(
                            text = item.subtitle.ifBlank { "Live radio" },
                            color = WaveSubtle,
                            fontSize = 13.sp,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                    Icon(Icons.Default.PlayArrow, contentDescription = null, tint = WaveSubtle)
                }
            }
        }
    }
}

@Composable
private fun TracksScreen(
    tracks: List<Track>,
    playerState: PlayerState,
    onPlayFromHere: (Track, List<Track>) -> Unit,
    playlists: List<Playlist>,
    onAddTrackToPlaylist: (Track, Playlist) -> Unit,
    trackArtworkUrl: (Track) -> String?,
    emptyMessage: String,
) {
    var selectedSort by remember { mutableStateOf(TrackSortOption.RecentlyUploaded) }
    val sortedTracks = remember(tracks, selectedSort) {
        tracks.sortedFor(selectedSort)
    }

    LazyColumn(modifier = Modifier.fillMaxSize()) {
        item {
            TrackSortHeader(
                count = sortedTracks.size,
                selectedSort = selectedSort,
                onSortSelected = { selectedSort = it },
            )
        }
        if (sortedTracks.isEmpty()) {
            item { EmptyLibraryMessage(emptyMessage) }
        }
        items(sortedTracks, key = { it.id }) { track ->
            TrackRow(
                track = track,
                isCurrent = playerState.currentTrack?.id == track.id,
                artworkUrl = trackArtworkUrl(track),
                playlists = playlists,
                onClick = { onPlayFromHere(track, sortedTracks) },
                onAddToPlaylist = onAddTrackToPlaylist,
            )
        }
    }
}

@Composable
private fun TrackSortHeader(
    count: Int,
    selectedSort: TrackSortOption,
    onSortSelected: (TrackSortOption) -> Unit,
) {
    var menuOpen by remember { mutableStateOf(false) }
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = "Tracks ($count)",
                color = WaveText,
                fontSize = 22.sp,
                fontWeight = FontWeight.Black,
            )
            Text(
                text = "Sorted by ${selectedSort.label.lowercase()}",
                color = WaveSubtle,
                fontSize = 12.sp,
            )
        }
        Box {
            Row(
                modifier = Modifier
                    .clip(RoundedCornerShape(999.dp))
                    .background(WaveSurfaceRaised)
                    .clickable { menuOpen = true }
                    .padding(horizontal = 14.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                Text(
                    text = selectedSort.label,
                    color = WaveText,
                    fontSize = 13.sp,
                    fontWeight = FontWeight.SemiBold,
                )
                Icon(
                    Icons.Default.KeyboardArrowDown,
                    contentDescription = null,
                    tint = WaveSubtle,
                    modifier = Modifier.size(18.dp),
                )
            }
            DropdownMenu(
                expanded = menuOpen,
                onDismissRequest = { menuOpen = false },
                modifier = Modifier.background(WaveSurface),
            ) {
                TrackSortOption.entries.forEach { option ->
                    DropdownMenuItem(
                        text = {
                            Text(
                                text = if (option == selectedSort) "${option.label} selected" else option.label,
                                color = WaveText,
                            )
                        },
                        onClick = {
                            onSortSelected(option)
                            menuOpen = false
                        },
                    )
                }
            }
        }
    }
}

@Composable
private fun AlbumsScreen(
    albums: List<Album>,
    tracks: List<Track>,
    albumArtworkUrl: (Album) -> String?,
    trackArtworkUrl: (Track) -> String?,
    onOpenAlbum: (Album) -> Unit,
    emptyMessage: String,
) {
    LazyColumn(modifier = Modifier.fillMaxSize()) {
        item { SectionHeader("Albums", "${albums.size}") }
        if (albums.isEmpty()) {
            item { EmptyLibraryMessage(emptyMessage) }
        }
        items(albums, key = { it.id.ifBlank { it.name } }) { album ->
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { onOpenAlbum(album) }
                    .padding(horizontal = 16.dp, vertical = 9.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Artwork(url = albumArtworkFor(album, tracks, albumArtworkUrl, trackArtworkUrl), size = 54)
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = album.name,
                        color = WaveText,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Text(
                        text = albumSubtitle(album),
                        color = WaveSubtle,
                        fontSize = 13.sp,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}

@Composable
private fun ArtistsScreen(
    artists: List<Artist>,
    tracks: List<Track>,
    artistArtworkUrl: (Artist) -> String?,
    trackArtworkUrl: (Track) -> String?,
    onOpenArtist: (Artist) -> Unit,
    emptyMessage: String,
) {
    LazyColumn(modifier = Modifier.fillMaxSize()) {
        item { SectionHeader("Artists", "${artists.size}") }
        if (artists.isEmpty()) {
            item { EmptyLibraryMessage(emptyMessage) }
        }
        items(artists, key = { it.id.ifBlank { it.name } }) { artist ->
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { onOpenArtist(artist) }
                    .padding(horizontal = 16.dp, vertical = 9.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Artwork(url = artistArtworkFor(artist, tracks, artistArtworkUrl, trackArtworkUrl), size = 54, rounded = 27)
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = artist.name,
                        color = WaveText,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Text(
                        text = "${artist.trackCount} tracks - ${artist.albumCount} albums",
                        color = WaveSubtle,
                        fontSize = 13.sp,
                    )
                }
            }
        }
    }
}

@Composable
private fun TrackRow(
    track: Track,
    isCurrent: Boolean,
    artworkUrl: String?,
    playlists: List<Playlist>,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
    onAddToPlaylist: (Track, Playlist) -> Unit,
) {
    var menuOpen by remember { mutableStateOf(false) }
    val editablePlaylists = remember(playlists) {
        playlists.filter { it.id.isNotBlank() && !it.type.equals("smart", ignoreCase = true) }
    }
    Row(
        modifier = modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 9.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Artwork(url = artworkUrl, size = 52)
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = track.title,
                color = if (isCurrent) WaveAccent else WaveText,
                fontWeight = FontWeight.SemiBold,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = listOf(track.artist, track.album).filter { it.isNotBlank() }.joinToString(" - "),
                color = WaveSubtle,
                fontSize = 13.sp,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Box {
            IconButton(onClick = { menuOpen = true }) {
                Icon(
                    imageVector = Icons.Default.MoreVert,
                    contentDescription = "Track options",
                    tint = if (isCurrent) WaveAccent else WaveSubtle,
                )
            }
            DropdownMenu(
                expanded = menuOpen,
                onDismissRequest = { menuOpen = false },
                modifier = Modifier.background(WaveSurface),
            ) {
                DropdownMenuItem(
                    text = { Text("Play", color = WaveText) },
                    leadingIcon = { Icon(Icons.Default.PlayArrow, contentDescription = null, tint = WaveAccent) },
                    onClick = {
                        menuOpen = false
                        onClick()
                    },
                )
                DropdownMenuItem(
                    text = { Text("Add to playlist", color = WaveSubtle, fontWeight = FontWeight.Bold) },
                    enabled = false,
                    onClick = {},
                )
                if (track.isExternal) {
                    DropdownMenuItem(
                        text = { Text("Radio streams cannot be added", color = WaveSubtle) },
                        enabled = false,
                        onClick = {},
                    )
                } else if (editablePlaylists.isEmpty()) {
                    DropdownMenuItem(
                        text = { Text("No manual playlists", color = WaveSubtle) },
                        enabled = false,
                        onClick = {},
                    )
                } else {
                    editablePlaylists.forEach { playlist ->
                        DropdownMenuItem(
                            text = {
                                Text(
                                    text = playlist.name.ifBlank { "Untitled playlist" },
                                    color = WaveText,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            },
                            leadingIcon = {
                                Icon(Icons.AutoMirrored.Filled.PlaylistPlay, contentDescription = null, tint = WaveAccent)
                            },
                            onClick = {
                                menuOpen = false
                                onAddToPlaylist(track, playlist)
                            },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun LibraryDetailScreen(
    detail: LibraryDetail,
    playerState: PlayerState,
    onBack: () -> Unit,
    onPlayFromHere: (Track, List<Track>) -> Unit,
    playlists: List<Playlist>,
    onAddTrackToPlaylist: (Track, Playlist) -> Unit,
    trackArtworkUrl: (Track) -> String?,
    albumArtworkUrl: (Album) -> String?,
    artistArtworkUrl: (Artist) -> String?,
    onOpenAlbum: (Album) -> Unit,
) {
    val title: String
    val subtitle: String
    val artworkUrl: String?
    val tracks: List<Track>
    val albums: List<Album>
    var shuffleSeed by remember { mutableStateOf(0) }

    when (detail) {
        is LibraryDetail.AlbumPage -> {
            title = detail.album.name
            subtitle = albumSubtitle(detail.album)
            tracks = detail.tracks
            artworkUrl = albumArtworkFor(detail.album, tracks, albumArtworkUrl, trackArtworkUrl)
            albums = emptyList()
        }
        is LibraryDetail.ArtistPage -> {
            title = detail.artist.name
            subtitle = "${detail.tracks.size} tracks - ${detail.albums.size} albums"
            tracks = detail.tracks
            artworkUrl = artistArtworkFor(detail.artist, tracks, artistArtworkUrl, trackArtworkUrl)
            albums = detail.albums
        }
        is LibraryDetail.PlaylistPage -> {
            title = detail.playlist.name
            subtitle = playlistSubtitle(detail.playlist).replace("${detail.playlist.trackIds.size} tracks", "${detail.tracks.size} tracks")
            tracks = detail.tracks
            artworkUrl = tracks.firstOrNull()?.let(trackArtworkUrl)
            albums = emptyList()
        }
    }
    val visibleTracks = remember(tracks, shuffleSeed) {
        if (shuffleSeed == 0 || tracks.size < 2) {
            tracks
        } else {
            tracks.shuffled(Random(shuffleSeed))
        }
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        item {
            Column(
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                }
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(16.dp),
                ) {
                    Artwork(url = artworkUrl, size = 116, rounded = if (detail is LibraryDetail.ArtistPage) 58 else 12)
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = title,
                            color = WaveText,
                            fontSize = 28.sp,
                            fontWeight = FontWeight.Black,
                            maxLines = 3,
                            overflow = TextOverflow.Ellipsis,
                        )
                        Text(
                            text = subtitle,
                            color = WaveSubtle,
                            fontSize = 14.sp,
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
                if (tracks.isNotEmpty()) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        Button(
                            onClick = { onPlayFromHere(visibleTracks.first(), visibleTracks) },
                            modifier = Modifier.weight(1f),
                        ) {
                            Icon(Icons.Default.PlayArrow, contentDescription = null)
                            Text("Play All")
                        }
                        IconButton(
                            onClick = { shuffleSeed = Random.nextInt(1, Int.MAX_VALUE) },
                            modifier = Modifier
                                .size(50.dp)
                                .clip(CircleShape)
                                .background(WaveSurfaceRaised),
                        ) {
                            Icon(
                                Icons.Default.Shuffle,
                                contentDescription = "Shuffle tracks",
                                tint = WaveAccent,
                            )
                        }
                    }
                }
            }
        }

        if (albums.isNotEmpty()) {
            item { SectionHeader("Albums", "${albums.size}") }
            item {
                LazyRow(
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.padding(start = 16.dp),
                ) {
                    items(albums, key = { it.id.ifBlank { "${it.name}:${it.artist}" } }) { album ->
                        AlbumCard(
                            album = album,
                            artworkUrl = albumArtworkFor(album, tracks, albumArtworkUrl, trackArtworkUrl),
                            onClick = { onOpenAlbum(album) },
                        )
                    }
                }
            }
        }

        item { SectionHeader("Tracks", "${tracks.size}") }
        if (tracks.isEmpty()) {
            item { EmptyLibraryMessage("No tracks found.") }
        } else {
            itemsIndexed(visibleTracks, key = { _, track -> track.id }) { _, track ->
                TrackRow(
                    track = track,
                    isCurrent = playerState.currentTrack?.id == track.id,
                    artworkUrl = trackArtworkUrl(track),
                    playlists = playlists,
                    modifier = Modifier.animateItem(),
                    onClick = { onPlayFromHere(track, visibleTracks) },
                    onAddToPlaylist = onAddTrackToPlaylist,
                )
            }
        }
    }
}

@Composable
private fun MiniPlayer(
    playerState: PlayerState,
    onTogglePlayPause: () -> Unit,
    onSkipNext: () -> Unit,
    onSkipPrevious: () -> Unit,
    onOpenQueue: () -> Unit,
    onOpenPlayer: () -> Unit,
    artworkUrl: String?,
) {
    val track = playerState.currentTrack ?: return
    val progress = playbackProgress(playerState)
    val audioOutput = rememberAudioOutputLabel()
    Surface(
        color = WaveSurface,
        tonalElevation = 4.dp,
        modifier = Modifier
            .fillMaxWidth()
            .height(74.dp)
            .padding(horizontal = 10.dp, vertical = 6.dp)
            .clip(RoundedCornerShape(12.dp)),
    ) {
        Box {
            Row(
                modifier = Modifier
                    .fillMaxSize()
                    .clickable(onClick = onOpenPlayer)
                    .padding(horizontal = 10.dp, vertical = 7.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Artwork(url = artworkUrl, size = 46)
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "${track.title} - ${track.artist}".trim(' ', '-'),
                        color = WaveText,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(5.dp),
                    ) {
                        Icon(
                            imageVector = if (audioOutput.isBluetooth) Icons.Default.Bluetooth else Icons.Default.Headphones,
                            contentDescription = null,
                            tint = WaveAccent,
                            modifier = Modifier.size(15.dp),
                        )
                        Text(
                            text = audioOutput.label,
                            color = WaveAccent,
                            fontSize = 12.sp,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
                IconButton(
                    onClick = onOpenQueue,
                    modifier = Modifier.size(40.dp),
                ) {
                    Icon(Icons.AutoMirrored.Filled.QueueMusic, contentDescription = "Queue")
                }
                IconButton(
                    onClick = onTogglePlayPause,
                    modifier = Modifier
                        .size(44.dp)
                        .clip(CircleShape)
                        .background(WaveAccent),
                ) {
                    Icon(
                        imageVector = if (playerState.isPlaying) Icons.Default.Pause else Icons.Default.PlayArrow,
                        contentDescription = if (playerState.isPlaying) "Pause" else "Play",
                        tint = WaveBackground,
                    )
                }
            }
            LinearProgressIndicator(
                progress = { progress },
                modifier = Modifier
                    .align(Alignment.BottomCenter)
                    .fillMaxWidth()
                    .height(3.dp),
                color = WaveAccent,
                trackColor = WaveSurfaceRaised,
            )
        }
    }
}

@Composable
private fun NowPlayingSheet(
    playerState: PlayerState,
    artworkUrl: String?,
    onDismiss: () -> Unit,
    onTogglePlayPause: () -> Unit,
    onToggleShuffle: () -> Unit,
    onCycleRepeatMode: () -> Unit,
    onSkipNext: () -> Unit,
    onSkipPrevious: () -> Unit,
    onSeekTo: (Long) -> Unit,
    onOpenQueue: () -> Unit,
    connectSessions: List<UserSession>,
    currentSessionId: String,
    connectedPlaybackSessionId: String,
    connectedPlaybackDeviceName: String,
    isLoadingConnectSessions: Boolean,
    connectMessage: String?,
    onRefreshConnectSessions: () -> Unit,
    onConnectPlaybackTo: (String) -> Unit,
) {
    val track = playerState.currentTrack ?: return
    val durationMs = effectiveDurationMs(playerState)
    val positionMs = playerState.positionMs.coerceAtMost(durationMs.coerceAtLeast(playerState.positionMs))
    val isRadio = track.isExternal
    val connectLabel = connectedPlaybackDeviceName
        .takeIf { connectedPlaybackSessionId.isNotBlank() && it.isNotBlank() }
        ?.let { "Playing on $it" }
        ?: "Connect to a device"
    var showOutputPicker by remember { mutableStateOf(false) }

    Surface(
        color = WaveBackground,
        contentColor = WaveText,
        modifier = Modifier.fillMaxSize(),
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .statusBarsPadding()
                .navigationBarsPadding()
                .padding(horizontal = 24.dp, vertical = 10.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                IconButton(onClick = onDismiss) {
                    Icon(Icons.Default.KeyboardArrowDown, contentDescription = "Collapse")
                }
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    modifier = Modifier.weight(1f),
                ) {
                    Text(
                        text = "PLAYING FROM WAVENODE",
                        color = WaveSubtle,
                        fontSize = 11.sp,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        text = if (isRadio) "Live radio" else track.album.ifBlank { "Your Library" },
                        color = WaveText,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                IconButton(onClick = { }) {
                    Icon(Icons.Default.MoreVert, contentDescription = "More")
                }
            }

            ArtworkHero(
                url = artworkUrl,
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f),
            )

            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = track.title,
                        color = WaveText,
                        fontSize = 25.sp,
                        fontWeight = FontWeight.Black,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                    Text(
                        text = track.artist,
                        color = WaveSubtle,
                        fontSize = 18.sp,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }

            if (!isRadio) {
                CleanSeekBar(
                    positionMs = positionMs,
                    durationMs = durationMs,
                    onSeekTo = onSeekTo,
                )
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                if (isRadio) {
                    Spacer(modifier = Modifier.size(58.dp))
                } else {
                    ShuffleModeControl(
                        isEnabled = playerState.isShuffleEnabled,
                        onToggle = onToggleShuffle,
                    )
                }
                if (isRadio) {
                    Spacer(modifier = Modifier.size(58.dp))
                } else {
                    IconButton(
                        onClick = onSkipPrevious,
                        enabled = playerState.currentIndex > 0 || playerState.repeatMode == WaveRepeatMode.All,
                        modifier = Modifier.size(58.dp),
                    ) {
                        Icon(Icons.Default.SkipPrevious, contentDescription = "Previous", modifier = Modifier.size(38.dp))
                    }
                }
                IconButton(
                    onClick = onTogglePlayPause,
                    modifier = Modifier
                        .size(76.dp)
                        .clip(CircleShape)
                        .background(WaveText),
                ) {
                    Icon(
                        imageVector = if (playerState.isPlaying) Icons.Default.Pause else Icons.Default.PlayArrow,
                        contentDescription = if (playerState.isPlaying) "Pause" else "Play",
                        tint = WaveBackground,
                        modifier = Modifier.size(42.dp),
                    )
                }
                if (isRadio) {
                    Spacer(modifier = Modifier.size(58.dp))
                } else {
                    IconButton(
                        onClick = onSkipNext,
                        enabled = playerState.currentIndex < playerState.queue.lastIndex || playerState.repeatMode == WaveRepeatMode.All,
                        modifier = Modifier.size(58.dp),
                    ) {
                        Icon(Icons.Default.SkipNext, contentDescription = "Next", modifier = Modifier.size(38.dp))
                    }
                }
                if (isRadio) {
                    Spacer(modifier = Modifier.size(48.dp))
                } else {
                    RepeatModeControl(
                        repeatMode = playerState.repeatMode,
                        onCycle = onCycleRepeatMode,
                    )
                }
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Row(
                    modifier = Modifier
                        .weight(1f)
                        .clickable {
                            onRefreshConnectSessions()
                            showOutputPicker = true
                        },
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Icon(
                        imageVector = Icons.Default.Devices,
                        contentDescription = null,
                        tint = WaveAccent,
                    )
                    Text(
                        text = connectLabel,
                        color = WaveAccent,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                IconButton(
                    onClick = onOpenQueue,
                    enabled = !isRadio,
                ) {
                    Icon(
                        Icons.AutoMirrored.Filled.QueueMusic,
                        contentDescription = "Queue",
                        tint = if (isRadio) WaveSubtle.copy(alpha = 0.35f) else WaveSubtle,
                    )
                }
            }
            if (showOutputPicker) {
                ConnectDeviceSheet(
                    sessions = connectSessions,
                    currentSessionId = currentSessionId,
                    connectedPlaybackSessionId = connectedPlaybackSessionId,
                    isLoading = isLoadingConnectSessions,
                    message = connectMessage,
                    onRefresh = onRefreshConnectSessions,
                    onConnect = onConnectPlaybackTo,
                    onDismiss = { showOutputPicker = false },
                )
            }
            Spacer(modifier = Modifier.height(2.dp))
        }
    }
}

@Composable
private fun ShuffleModeControl(
    isEnabled: Boolean,
    onToggle: () -> Unit,
) {
    val label = if (isEnabled) "Shuffle on" else "Shuffle off"
    val tint = if (isEnabled) WaveAccent else WaveSubtle
    IconButton(onClick = onToggle, modifier = Modifier.size(58.dp)) {
        Icon(
            Icons.Default.Shuffle,
            contentDescription = label,
            tint = tint,
            modifier = Modifier.size(32.dp),
        )
    }
}

@Composable
private fun RepeatModeControl(
    repeatMode: WaveRepeatMode,
    onCycle: () -> Unit,
) {
    val label = when (repeatMode) {
        WaveRepeatMode.Off -> "Repeat off"
        WaveRepeatMode.All -> "Repeat all"
        WaveRepeatMode.One -> "Repeat 1"
    }
    val tint = if (repeatMode == WaveRepeatMode.Off) WaveSubtle else WaveAccent
    IconButton(onClick = onCycle, modifier = Modifier.size(58.dp)) {
        Box(contentAlignment = Alignment.Center) {
            Icon(
                Icons.Default.Repeat,
                contentDescription = label,
                tint = tint,
                modifier = Modifier.size(32.dp),
            )
            if (repeatMode == WaveRepeatMode.One) {
                Box(
                    modifier = Modifier
                        .align(Alignment.BottomEnd)
                        .size(16.dp)
                        .clip(CircleShape)
                        .background(WaveAccent),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = "1",
                        color = WaveBackground,
                        fontSize = 9.sp,
                        fontWeight = FontWeight.Black,
                        lineHeight = 9.sp,
                    )
                }
            }
        }
    }
}

@Composable
private fun CleanSeekBar(
    positionMs: Long,
    durationMs: Long,
    onSeekTo: (Long) -> Unit,
) {
    var widthPx by remember { mutableStateOf(1) }
    val progress = if (durationMs > 0L) {
        (positionMs.toFloat() / durationMs.toFloat()).coerceIn(0f, 1f)
    } else {
        0f
    }

    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(22.dp)
                .onSizeChanged { widthPx = it.width.coerceAtLeast(1) }
                .pointerInput(durationMs) {
                    detectTapGestures { offset ->
                        if (durationMs > 0L) {
                            val tappedProgress = (offset.x / widthPx.toFloat()).coerceIn(0f, 1f)
                            onSeekTo((durationMs * tappedProgress).toLong())
                        }
                    }
                },
            contentAlignment = Alignment.CenterStart,
        ) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(6.dp)
                    .clip(CircleShape)
                    .background(WaveSurfaceRaised),
            )
            Box(
                modifier = Modifier
                    .fillMaxWidth(progress)
                    .height(6.dp)
                    .clip(CircleShape)
                    .background(WaveAccent),
            )
        }
        Row(modifier = Modifier.fillMaxWidth()) {
            Text(formatDuration(positionMs), color = WaveSubtle, fontSize = 12.sp)
            Spacer(modifier = Modifier.weight(1f))
            Text(formatDuration(durationMs), color = WaveSubtle, fontSize = 12.sp)
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ConnectDeviceSheet(
    sessions: List<UserSession>,
    currentSessionId: String,
    connectedPlaybackSessionId: String,
    isLoading: Boolean,
    message: String?,
    onRefresh: () -> Unit,
    onConnect: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        containerColor = WaveSurface,
        contentColor = WaveText,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .navigationBarsPadding()
                .padding(horizontal = 24.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = "Connect",
                    color = WaveText,
                    fontSize = 26.sp,
                    fontWeight = FontWeight.Black,
                    modifier = Modifier.weight(1f),
                )
                IconButton(onClick = onRefresh) {
                    Icon(Icons.Default.Refresh, contentDescription = "Refresh devices")
                }
            }
            if (message != null) {
                Text(
                    text = message,
                    color = if (message.startsWith("Playback sent") || message.startsWith("Controlling") || message.startsWith("Playback switched")) WaveAccent else MaterialTheme.colorScheme.error,
                    fontSize = 13.sp,
                )
            }
            if (isLoading) {
                Row(
                    modifier = Modifier.fillMaxWidth().padding(vertical = 16.dp),
                    horizontalArrangement = Arrangement.Center,
                ) {
                    CircularProgressIndicator(color = WaveAccent, modifier = Modifier.size(26.dp))
                }
            }
            val activeSessions = sessions.filter { it.id.isNotBlank() }
            if (!isLoading && activeSessions.isEmpty()) {
                EmptyLibraryMessage("No WaveNode devices found.")
            }
            activeSessions.forEach { session ->
                val isCurrent = session.id == currentSessionId
                val isConnectedRemote = session.id == connectedPlaybackSessionId
                val canSelect = !isCurrent || connectedPlaybackSessionId.isNotBlank()
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(14.dp))
                        .background(if (isCurrent || isConnectedRemote) WaveSurfaceRaised else Color.Transparent)
                        .clickable(enabled = canSelect) { onConnect(session.id) }
                        .padding(14.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Icon(
                        imageVector = connectDeviceIcon(session, isCurrent),
                        contentDescription = null,
                        tint = if (isCurrent || isConnectedRemote) WaveAccent else WaveSubtle,
                        modifier = Modifier.size(34.dp),
                    )
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = if (isCurrent) "This phone" else session.deviceName.ifBlank { "WaveNode device" },
                            color = if (isCurrent || isConnectedRemote) WaveAccent else WaveText,
                            fontWeight = FontWeight.Bold,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                        Text(
                            text = when {
                                isCurrent && connectedPlaybackSessionId.isNotBlank() -> "Tap to play on this phone"
                                isCurrent -> "Current playback device"
                                isConnectedRemote -> "Current playback host"
                                else -> session.ipAddress.ifBlank { "Signed in to WaveNode" }
                            },
                            color = WaveSubtle,
                            fontSize = 12.sp,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
            }
            Text(
                text = "Open WaveNode on another signed-in device to make it available here.",
                color = WaveSubtle,
                fontSize = 12.sp,
            )
            Spacer(modifier = Modifier.height(12.dp))
        }
    }
}

private fun connectDeviceIcon(session: UserSession, isCurrent: Boolean): ImageVector {
    if (isCurrent) {
        return Icons.Default.PhoneAndroid
    }
    val lower = "${session.deviceName} ${session.userAgent}".lowercase()
    return when {
        "android" in lower || "iphone" in lower || "mobile" in lower -> Icons.Default.PhoneAndroid
        else -> Icons.Default.Computer
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AudioOutputSheet(
    onDismiss: () -> Unit,
) {
    val context = LocalContext.current
    val outputs = remember { availableAudioOutputs(context) }
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        containerColor = WaveSurface,
        contentColor = WaveText,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .navigationBarsPadding()
                .padding(horizontal = 18.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            SectionHeader("Audio Output")
            outputs.forEach { output ->
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(10.dp))
                        .clickable {
                            selectAudioOutput(context, output)
                            onDismiss()
                        }
                        .padding(14.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Icon(
                        imageVector = if (output.isBluetooth) Icons.Default.Bluetooth else Icons.AutoMirrored.Filled.VolumeUp,
                        contentDescription = null,
                        tint = WaveAccent,
                    )
                    Text(
                        text = output.label,
                        color = WaveText,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            Text(
                text = "Android controls media routing. If a device does not switch here, use the system media output picker.",
                color = WaveSubtle,
                fontSize = 12.sp,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            )
        }
    }
}

@Composable
private fun ArtworkHero(url: String?, modifier: Modifier = Modifier) {
    Box(
        modifier = modifier
            .clip(RoundedCornerShape(18.dp))
            .background(WaveSurface),
        contentAlignment = Alignment.Center,
    ) {
        if (url.isNullOrBlank()) {
            Icon(
                imageVector = Icons.Default.MusicNote,
                contentDescription = null,
                tint = WaveAccent,
                modifier = Modifier.size(110.dp),
            )
        } else {
            AsyncImage(
                model = url,
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxSize(),
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun QueueSheet(
    playerState: PlayerState,
    onDismiss: () -> Unit,
    onPlayTrack: (Track) -> Unit,
    playlists: List<Playlist>,
    onAddTrackToPlaylist: (Track, Playlist) -> Unit,
    trackArtworkUrl: (Track) -> String?,
) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        containerColor = WaveSurface,
        contentColor = WaveText,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .navigationBarsPadding()
                .padding(bottom = 16.dp),
        ) {
            SectionHeader("Queue", "${playerState.queue.size}")
            if (playerState.queue.isEmpty()) {
                EmptyLibraryMessage("Nothing is queued.")
            } else {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(420.dp),
                ) {
                    items(playerState.queue, key = { it.id }) { track ->
                        TrackRow(
                            track = track,
                            isCurrent = playerState.currentTrack?.id == track.id,
                            artworkUrl = trackArtworkUrl(track),
                            playlists = playlists,
                            onClick = { onPlayTrack(track) },
                            onAddToPlaylist = onAddTrackToPlaylist,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun StatTile(label: String, value: String, modifier: Modifier = Modifier) {
    Card(
        modifier = modifier,
        colors = CardDefaults.cardColors(containerColor = WaveSurface),
        shape = RoundedCornerShape(10.dp),
    ) {
        Column(modifier = Modifier.padding(14.dp)) {
            Text(value, color = WaveText, fontSize = 24.sp, fontWeight = FontWeight.Black)
            Text(label, color = WaveSubtle, fontSize = 12.sp)
        }
    }
}

@Composable
private fun AlbumCard(album: Album, artworkUrl: String?, onClick: () -> Unit) {
    Column(
        modifier = Modifier
            .size(width = 132.dp, height = 184.dp)
            .clickable(onClick = onClick),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Artwork(url = artworkUrl, size = 132)
        Text(
            text = album.name,
            color = WaveText,
            fontWeight = FontWeight.SemiBold,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        Text(
            text = album.artist,
            color = WaveSubtle,
            fontSize = 12.sp,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun RadioCard(item: PluginRowItem, onClick: () -> Unit) {
    Column(
        modifier = Modifier
            .size(width = 146.dp, height = 184.dp)
            .clip(RoundedCornerShape(10.dp))
            .clickable(onClick = onClick)
            .padding(end = 6.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Artwork(url = item.imageUrl.takeIf { it.isNotBlank() }, size = 132)
        Text(
            text = item.title,
            color = WaveText,
            fontWeight = FontWeight.SemiBold,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        Text(
            text = item.subtitle.ifBlank { "Live radio" },
            color = WaveSubtle,
            fontSize = 12.sp,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun SectionHeader(title: String, trailing: String? = null) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = title,
            color = WaveText,
            fontSize = 22.sp,
            fontWeight = FontWeight.Black,
            modifier = Modifier.weight(1f),
        )
        if (trailing != null) {
            Text(text = trailing, color = WaveSubtle, fontSize = 13.sp)
        }
    }
}

@Composable
private fun Artwork(url: String?, size: Int, rounded: Int = 8) {
    Box(
        modifier = Modifier
            .size(size.dp)
            .clip(if (rounded >= size / 2) CircleShape else RoundedCornerShape(rounded.dp))
            .background(WaveSurfaceRaised),
        contentAlignment = Alignment.Center,
    ) {
        if (url.isNullOrBlank()) {
            Icon(
                imageVector = Icons.Default.MusicNote,
                contentDescription = null,
                tint = WaveAccent,
            )
        } else {
            AsyncImage(
                model = url,
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxSize(),
            )
        }
    }
}

@Composable
private fun WaveNodeMark(size: Int = 46) {
    Box(
        modifier = Modifier
            .size(size.dp)
            .clip(RoundedCornerShape((size / 4).dp))
            .background(WaveAccent),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            imageVector = Icons.Default.LibraryMusic,
            contentDescription = null,
            tint = WaveBackground,
            modifier = Modifier.size((size * 0.58f).dp),
        )
    }
}

@Composable
private fun EmptyLibraryMessage(message: String) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(32.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(text = message, color = WaveSubtle, fontSize = 15.sp)
    }
}

private data class AudioOutputLabel(
    val label: String,
    val isBluetooth: Boolean,
)

private data class AudioOutputOption(
    val label: String,
    val isBluetooth: Boolean,
    val device: AudioDeviceInfo? = null,
)

@Composable
private fun rememberAudioOutputLabel(): AudioOutputLabel {
    val context = LocalContext.current
    return remember {
        currentAudioOutputLabel(context)
    }
}

@Suppress("DEPRECATION")
private fun currentAudioOutputLabel(context: Context): AudioOutputLabel {
    val bluetoothOutput = availableAudioOutputs(context).firstOrNull { it.isBluetooth }
    if (bluetoothOutput != null) {
        return AudioOutputLabel(bluetoothOutput.label, true)
    }

    return AudioOutputLabel("This device", false)
}

@Suppress("DEPRECATION")
private fun availableAudioOutputs(context: Context): List<AudioOutputOption> {
    val audioManager = context.getSystemService(Context.AUDIO_SERVICE) as? AudioManager
        ?: return listOf(AudioOutputOption("This device", false))
    val options = mutableListOf(AudioOutputOption("This device", false))
    val canReadBluetoothNames = Build.VERSION.SDK_INT < Build.VERSION_CODES.S ||
        context.checkSelfPermission(Manifest.permission.BLUETOOTH_CONNECT) == PackageManager.PERMISSION_GRANTED

    if (canReadBluetoothNames) {
        audioManager.getDevices(AudioManager.GET_DEVICES_OUTPUTS)
            .filter { it.isBluetoothOutput() }
            .forEach { device ->
                val name = device.productName?.toString()?.takeIf { it.isNotBlank() } ?: "Bluetooth headphones"
                options.add(AudioOutputOption(name, true, device))
            }
    } else if (Build.VERSION.SDK_INT < Build.VERSION_CODES.S && audioManager.isBluetoothA2dpOn) {
        options.add(AudioOutputOption("Bluetooth headphones", true))
    }

    return options.distinctBy { "${it.label}:${it.isBluetooth}" }
}

private fun selectAudioOutput(context: Context, output: AudioOutputOption) {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.S) {
        return
    }
    val audioManager = context.getSystemService(Context.AUDIO_SERVICE) as? AudioManager ?: return
    if (context.checkSelfPermission(Manifest.permission.BLUETOOTH_CONNECT) != PackageManager.PERMISSION_GRANTED) {
        return
    }
    if (output.device == null) {
        audioManager.clearCommunicationDevice()
    } else {
        audioManager.setCommunicationDevice(output.device)
    }
}

private fun AudioDeviceInfo.isBluetoothOutput(): Boolean {
    return type == AudioDeviceInfo.TYPE_BLUETOOTH_A2DP ||
        type == AudioDeviceInfo.TYPE_BLUETOOTH_SCO ||
        (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S && type == AudioDeviceInfo.TYPE_BLE_HEADSET) ||
        (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S && type == AudioDeviceInfo.TYPE_BLE_SPEAKER)
}

private fun playbackProgress(playerState: PlayerState): Float {
    val duration = effectiveDurationMs(playerState)
    if (duration <= 0L) {
        return 0f
    }
    return (playerState.positionMs.toFloat() / duration.toFloat()).coerceIn(0f, 1f)
}

private fun effectiveDurationMs(playerState: PlayerState): Long {
    if (playerState.currentTrack?.isExternal == true) {
        return 0L
    }
    val measuredDuration = playerState.durationMs
    if (measuredDuration > 0L) {
        return measuredDuration
    }
    val trackDurationSeconds = playerState.currentTrack?.duration ?: 0
    return if (trackDurationSeconds > 0) trackDurationSeconds * 1000L else 0L
}

private fun formatDuration(durationMs: Long): String {
    val totalSeconds = (durationMs / 1000L).coerceAtLeast(0L)
    val minutes = totalSeconds / 60L
    val seconds = totalSeconds % 60L
    return "$minutes:${seconds.toString().padStart(2, '0')}"
}

private fun <T> List<T>.filterByQuery(query: String, fields: T.() -> List<String>): List<T> {
    val normalized = query.trim()
    if (normalized.isBlank()) return this
    return filter { item -> item.fields().any { it.contains(normalized, ignoreCase = true) } }
}

private fun emptyMessage(type: String, searchQuery: String): String {
    return if (searchQuery.isBlank()) {
        "No $type found on this server."
    } else {
        "No $type match \"$searchQuery\"."
    }
}

private fun albumSubtitle(album: Album): String {
    return listOf(
        album.artist,
        album.year.takeIf { it > 0 }?.toString().orEmpty(),
        "${album.trackCount} tracks",
    ).filter { it.isNotBlank() }.joinToString(" - ")
}

private fun playlistSubtitle(playlist: Playlist): String {
    val type = if (playlist.type == "smart") "Smart playlist" else "Playlist"
    return listOf(
        type,
        "${playlist.trackIds.size} tracks",
        playlist.description,
    ).filter { it.isNotBlank() }.joinToString(" - ")
}

private fun List<Track>.sortedFor(sortOption: TrackSortOption): List<Track> {
    return when (sortOption) {
        TrackSortOption.Title -> sortedWith(compareByText<Track> { it.title })
        TrackSortOption.Artist -> sortedWith(
            compareByText<Track> { it.artist }
                .then(compareByText { it.title }),
        )
        TrackSortOption.Album -> sortedWith(
            compareByText<Track> { it.album }
                .then(compareByText { it.title }),
        )
        TrackSortOption.Duration -> sortedWith(
            compareBy<Track> { it.duration }
                .then(compareByText { it.title }),
        )
        TrackSortOption.RecentlyUploaded -> sortedWith(
            compareByDescending<Track> { it.createdAt.toEpochMillis() }
                .thenByDescending { it.uploadOrder }
                .then(compareByText { it.title }),
        )
    }
}

private fun <T> compareByText(selector: (T) -> String): Comparator<T> {
    return compareBy(String.CASE_INSENSITIVE_ORDER, selector)
}

private fun String.toEpochMillis(): Long {
    if (isBlank()) {
        return 0L
    }
    return runCatching { Instant.parse(this).toEpochMilli() }.getOrDefault(0L)
}

private fun albumArtworkFor(
    album: Album,
    tracks: List<Track>,
    albumArtworkUrl: (Album) -> String?,
    trackArtworkUrl: (Track) -> String?,
): String? {
    return albumArtworkUrl(album)
        ?: tracks.firstOrNull { track ->
            track.album.equals(album.name, ignoreCase = true) &&
                (album.artist.isBlank() || track.artist.equals(album.artist, ignoreCase = true))
        }?.let(trackArtworkUrl)
}

private fun artistArtworkFor(
    artist: Artist,
    tracks: List<Track>,
    artistArtworkUrl: (Artist) -> String?,
    trackArtworkUrl: (Track) -> String?,
): String? {
    return artistArtworkUrl(artist)
        ?: tracks.firstOrNull { track ->
            track.artist.equals(artist.name, ignoreCase = true)
        }?.let(trackArtworkUrl)
}

private fun pluginItemToTrack(pluginId: String, item: PluginRowItem): Track {
    return Track(
        id = "plugin:$pluginId:${item.id}",
        title = item.title,
        artist = item.description.ifBlank { item.subtitle.ifBlank { "Live stream" } },
        album = "Live radio",
        genre = "Radio",
        duration = 0,
        imageUrl = item.imageUrl,
        streamUrl = item.streamUrl,
        isExternal = true,
    )
}
