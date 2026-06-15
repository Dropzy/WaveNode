package com.musicserver.data.repository

import com.musicserver.data.api.MusicApiService
import com.musicserver.data.models.*
import okhttp3.ResponseBody
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class MusicRepository @Inject constructor(
    private val apiService: MusicApiService
) {
    
    suspend fun getAllMusic(): Result<List<Music>> {
        return try {
            val response = apiService.getAllMusic()
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to get music"))
                }
            } else {
                Result.failure(Exception("Failed to get music: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun getMusic(id: String): Result<Music> {
        return try {
            val response = apiService.getMusic(id)
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to get music"))
                }
            } else {
                Result.failure(Exception("Failed to get music: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun searchMusic(query: String): Result<List<Music>> {
        return try {
            val response = apiService.searchMusic(query)
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to search music"))
                }
            } else {
                Result.failure(Exception("Failed to search music: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun streamMusic(id: String): Result<ResponseBody> {
        return try {
            val response = apiService.streamMusic(id)
            if (response.isSuccessful) {
                Result.success(response.body()!!)
            } else {
                Result.failure(Exception("Failed to stream music: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun getAlbums(): Result<List<Album>> {
        return try {
            val response = apiService.getAlbums()
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to get albums"))
                }
            } else {
                Result.failure(Exception("Failed to get albums: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun getAlbumTracks(albumName: String): Result<AlbumTracksResponse> {
        return try {
            val response = apiService.getAlbumTracks(albumName)
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to get album tracks"))
                }
            } else {
                Result.failure(Exception("Failed to get album tracks: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun getArtists(): Result<List<String>> {
        return try {
            val response = apiService.getArtists()
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to get artists"))
                }
            } else {
                Result.failure(Exception("Failed to get artists: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun getArtistTracks(artistName: String): Result<ArtistTracksResponse> {
        return try {
            val response = apiService.getArtistTracks(artistName)
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to get artist tracks"))
                }
            } else {
                Result.failure(Exception("Failed to get artist tracks: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun getAllPlaylists(): Result<List<Playlist>> {
        return try {
            val response = apiService.getAllPlaylists()
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to get playlists"))
                }
            } else {
                Result.failure(Exception("Failed to get playlists: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun getPlaylist(id: String): Result<Playlist> {
        return try {
            val response = apiService.getPlaylist(id)
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to get playlist"))
                }
            } else {
                Result.failure(Exception("Failed to get playlist: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun createPlaylist(name: String, description: String): Result<Playlist> {
        return try {
            val response = apiService.createPlaylist(CreatePlaylistRequest(name, description))
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to create playlist"))
                }
            } else {
                Result.failure(Exception("Failed to create playlist: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun updatePlaylist(id: String, name: String?, description: String?): Result<Playlist> {
        return try {
            val response = apiService.updatePlaylist(id, UpdatePlaylistRequest(name, description))
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to update playlist"))
                }
            } else {
                Result.failure(Exception("Failed to update playlist: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun deletePlaylist(id: String): Result<Unit> {
        return try {
            val response = apiService.deletePlaylist(id)
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true) {
                    Result.success(Unit)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to delete playlist"))
                }
            } else {
                Result.failure(Exception("Failed to delete playlist: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun addTrackToPlaylist(playlistId: String, trackId: String): Result<Playlist> {
        return try {
            val response = apiService.addTrackToPlaylist(playlistId, AddTrackToPlaylistRequest(trackId))
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to add track to playlist"))
                }
            } else {
                Result.failure(Exception("Failed to add track to playlist: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun removeTrackFromPlaylist(playlistId: String, trackId: String): Result<Playlist> {
        return try {
            val response = apiService.removeTrackFromPlaylist(playlistId, trackId)
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to remove track from playlist"))
                }
            } else {
                Result.failure(Exception("Failed to remove track from playlist: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun getLikedTracks(): Result<List<Music>> {
        return try {
            val response = apiService.getLikedTracks()
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to get liked tracks"))
                }
            } else {
                Result.failure(Exception("Failed to get liked tracks: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun likeTrack(trackId: String): Result<Unit> {
        return try {
            val response = apiService.likeTrack(trackId)
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true) {
                    Result.success(Unit)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to like track"))
                }
            } else {
                Result.failure(Exception("Failed to like track: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun unlikeTrack(trackId: String): Result<Unit> {
        return try {
            val response = apiService.unlikeTrack(trackId)
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true) {
                    Result.success(Unit)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to unlike track"))
                }
            } else {
                Result.failure(Exception("Failed to unlike track: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun isTrackLiked(trackId: String): Result<Boolean> {
        return try {
            val response = apiService.isTrackLiked(trackId)
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data["is_liked"] ?: false)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to check if track is liked"))
                }
            } else {
                Result.failure(Exception("Failed to check if track is liked: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun healthCheck(): Result<HealthResponse> {
        return try {
            val response = apiService.healthCheck()
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Health check failed"))
                }
            } else {
                Result.failure(Exception("Health check failed: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
