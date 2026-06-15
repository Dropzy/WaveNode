package com.musicserver.navigation

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.musicserver.ui.screens.LoginScreen
import com.musicserver.ui.screens.RegisterScreen
import com.musicserver.ui.screens.HomeScreen
import com.musicserver.ui.screens.LibraryScreen
import com.musicserver.ui.screens.SearchScreen
import com.musicserver.ui.screens.PlaylistScreen
import com.musicserver.ui.screens.AlbumScreen
import com.musicserver.ui.screens.ArtistScreen
import com.musicserver.ui.screens.LikedSongsScreen
import com.musicserver.ui.screens.NowPlayingScreen
import com.musicserver.viewmodel.AuthViewModel
import com.musicserver.viewmodel.MusicViewModel

@Composable
fun MusicNavigation(
    musicService: com.musicserver.player.MusicService?,
    musicViewModel: MusicViewModel,
    onServiceConnected: (com.musicserver.player.MusicService) -> Unit,
    modifier: Modifier = Modifier,
    navController: NavHostController = rememberNavController(),
    authViewModel: AuthViewModel = hiltViewModel()
) {
    val isAuthenticated by authViewModel.isAuthenticated.collectAsState()
    
    LaunchedEffect(isAuthenticated) {
        if (isAuthenticated) {
            navController.navigate("home") {
                popUpTo("login") { inclusive = true }
            }
        } else {
            navController.navigate("login") {
                popUpTo(0) { inclusive = true }
            }
        }
    }
    
    NavHost(
        navController = navController,
        startDestination = "login",
        modifier = modifier
    ) {
        // Authentication Screens
        composable("login") {
            LoginScreen(
                onLoginSuccess = { navController.navigate("home") },
                onNavigateToRegister = { navController.navigate("register") },
                authViewModel = authViewModel
            )
        }
        
        composable("register") {
            RegisterScreen(
                onRegisterSuccess = { navController.navigate("home") },
                onNavigateToLogin = { navController.popBackStack() },
                authViewModel = authViewModel
            )
        }
        
        // Main App Screens
        composable("home") {
            HomeScreen(
                musicService = null, // Using ViewModel instead
                onServiceConnected = onServiceConnected,
                musicViewModel = musicViewModel,
                authViewModel = authViewModel,
                onNavigateToLibrary = { navController.navigate("library") },
                onNavigateToSearch = { navController.navigate("search") },
                onNavigateToPlaylist = { playlistId -> 
                    navController.navigate("playlist/$playlistId") 
                },
                onNavigateToAlbum = { albumName -> 
                    navController.navigate("album/$albumName") 
                },
                onNavigateToArtist = { artistName -> 
                    navController.navigate("artist/$artistName") 
                },
                onNavigateToLikedSongs = { navController.navigate("liked_songs") },
                onLogout = {
                    authViewModel.logout()
                }
            )
        }
        
        composable("library") {
            LibraryScreen(
                musicService = null, // Using ViewModel instead
                musicViewModel = musicViewModel,
                onNavigateToPlaylist = { playlistId -> 
                    navController.navigate("playlist/$playlistId") 
                },
                onNavigateToAlbum = { albumName -> 
                    navController.navigate("album/$albumName") 
                },
                onNavigateToArtist = { artistName -> 
                    navController.navigate("artist/$artistName") 
                },
                onNavigateBack = { navController.popBackStack() }
            )
        }
        
        composable("search") {
            SearchScreen(
                musicService = null, // Using ViewModel instead
                musicViewModel = musicViewModel,
                onNavigateToPlaylist = { playlistId -> 
                    navController.navigate("playlist/$playlistId") 
                },
                onNavigateToAlbum = { albumName -> 
                    navController.navigate("album/$albumName") 
                },
                onNavigateToArtist = { artistName -> 
                    navController.navigate("artist/$artistName") 
                },
                onNavigateBack = { navController.popBackStack() }
            )
        }
        
        composable("playlist/{playlistId}") { backStackEntry ->
            val playlistId = backStackEntry.arguments?.getString("playlistId") ?: ""
            PlaylistScreen(
                playlistId = playlistId,
                musicService = null, // Using ViewModel instead
                musicViewModel = musicViewModel,
                onNavigateBack = { navController.popBackStack() }
            )
        }
        
        composable("album/{albumName}") { backStackEntry ->
            val albumName = backStackEntry.arguments?.getString("albumName") ?: ""
            AlbumScreen(
                albumName = albumName,
                musicService = null, // Using ViewModel instead
                musicViewModel = musicViewModel,
                onNavigateToArtist = { artistName -> 
                    navController.navigate("artist/$artistName") 
                },
                onNavigateBack = { navController.popBackStack() }
            )
        }
        
        composable("artist/{artistName}") { backStackEntry ->
            val artistName = backStackEntry.arguments?.getString("artistName") ?: ""
            ArtistScreen(
                artistName = artistName,
                musicService = null, // Using ViewModel instead
                musicViewModel = musicViewModel,
                onNavigateToAlbum = { albumName -> 
                    navController.navigate("album/$albumName") 
                },
                onNavigateBack = { navController.popBackStack() }
            )
        }
        
        composable("liked_songs") {
            LikedSongsScreen(
                musicService = null, // Using ViewModel instead
                musicViewModel = musicViewModel,
                onNavigateToAlbum = { albumName -> 
                    navController.navigate("album/$albumName") 
                },
                onNavigateToArtist = { artistName -> 
                    navController.navigate("artist/$artistName") 
                },
                onNavigateBack = { navController.popBackStack() }
            )
        }
        
        composable("now_playing") {
            NowPlayingScreen(
                musicViewModel = musicViewModel,
                onNavigateBack = { navController.popBackStack() }
            )
        }
        
    }
}
