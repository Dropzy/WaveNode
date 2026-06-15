package com.musicserver

import android.content.Context
import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import androidx.activity.viewModels
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.navigation.compose.rememberNavController
import com.musicserver.navigation.MusicNavigation
import com.musicserver.player.MusicService
import com.musicserver.ui.components.MiniPlayBar
import com.musicserver.ui.theme.MusicServerTheme
import com.musicserver.viewmodel.MusicViewModel
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    
    private val musicViewModel: MusicViewModel by viewModels()
    
    override fun onStart() {
        super.onStart()
        android.util.Log.d("MainActivity", "Starting and binding to MusicService")
        Intent(this, MusicService::class.java).also { intent ->
            val started = startService(intent)
            android.util.Log.d("MainActivity", "Service started: $started")
        }
        
        // Connect to service using ViewModel
        musicViewModel.connectToService(this)
    }
    
    override fun onStop() {
        super.onStop()
        // Disconnect from service using ViewModel
        musicViewModel.disconnectFromService(this)
    }
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        
        setContent {
            val currentTrack by musicViewModel.currentTrack.collectAsState()
            val isPlaying by musicViewModel.isPlaying.collectAsState()
            val isServiceConnected by musicViewModel.isServiceConnected.collectAsState()
            val navController = rememberNavController()
            
            MusicServerTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    Scaffold(
                        bottomBar = {
                            MiniPlayBar(
                                musicService = null, // We'll use ViewModel methods instead
                                currentTrack = currentTrack,
                                isPlaying = isPlaying,
                                onPlayPause = {
                                    if (isPlaying) {
                                        musicViewModel.pause()
                                    } else {
                                        musicViewModel.play()
                                    }
                                },
                                onSkipToNext = {
                                    musicViewModel.skipToNext()
                                },
                                onSkipToPrevious = {
                                    musicViewModel.skipToPrevious()
                                },
                                onExpand = {
                                    navController.navigate("now_playing")
                                }
                            )
                        }
                    ) { paddingValues ->
                        MusicNavigation(
                            musicService = null, // We'll use ViewModel methods instead
                            musicViewModel = musicViewModel,
                            onServiceConnected = { service ->
                                // Service connection is handled by ViewModel
                            },
                            modifier = Modifier.padding(paddingValues),
                            navController = navController
                        )
                    }
                }
            }
        }
    }
}
