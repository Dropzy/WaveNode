package com.musicserver.data.api

import com.musicserver.data.models.*
import okhttp3.ResponseBody
import retrofit2.Response
import retrofit2.http.*

interface MusicApiService {
    
    // Auth endpoints
    @POST("auth/login")
    suspend fun login(@Body request: LoginRequest): Response<APIResponse<AuthResponse>>
    
    @POST("auth/register")
    suspend fun register(@Body request: RegisterRequest): Response<APIResponse<AuthResponse>>
    
    @GET("auth/me")
    suspend fun getCurrentUser(): Response<APIResponse<User>>
    
    // Music endpoints
    @GET("music")
    suspend fun getAllMusic(): Response<APIResponse<List<Music>>>
    
    @GET("music/{id}")
    suspend fun getMusic(@Path("id") id: String): Response<APIResponse<Music>>
    
    @GET("music/search")
    suspend fun searchMusic(@Query("q") query: String): Response<APIResponse<List<Music>>>
    
    @POST("music")
    suspend fun addMusic(@Body music: Music): Response<APIResponse<Music>>
    
    @PUT("music/{id}")
    suspend fun updateMusic(@Path("id") id: String, @Body music: Music): Response<APIResponse<Music>>
    
    @DELETE("music/{id}")
    suspend fun deleteMusic(@Path("id") id: String): Response<APIResponse<Nothing>>
    
    @Streaming
    @GET("music/{id}/stream")
    suspend fun streamMusic(@Path("id") id: String): Response<ResponseBody>
    
    // Album endpoints
    @GET("albums")
    suspend fun getAlbums(): Response<APIResponse<List<Album>>>
    
    @GET("albums/{albumName}/tracks")
    suspend fun getAlbumTracks(@Path("albumName") albumName: String): Response<APIResponse<AlbumTracksResponse>>
    
    // Artist endpoints
    @GET("artists")
    suspend fun getArtists(): Response<APIResponse<List<String>>>
    
    @GET("artists/{artistName}/tracks")
    suspend fun getArtistTracks(@Path("artistName") artistName: String): Response<APIResponse<ArtistTracksResponse>>
    
    // Playlist endpoints
    @GET("playlists")
    suspend fun getAllPlaylists(): Response<APIResponse<List<Playlist>>>
    
    @GET("playlists/{id}")
    suspend fun getPlaylist(@Path("id") id: String): Response<APIResponse<Playlist>>
    
    @POST("playlists")
    suspend fun createPlaylist(@Body request: CreatePlaylistRequest): Response<APIResponse<Playlist>>
    
    @PUT("playlists/{id}")
    suspend fun updatePlaylist(@Path("id") id: String, @Body request: UpdatePlaylistRequest): Response<APIResponse<Playlist>>
    
    @DELETE("playlists/{id}")
    suspend fun deletePlaylist(@Path("id") id: String): Response<APIResponse<Nothing>>
    
    @POST("playlists/{id}/tracks")
    suspend fun addTrackToPlaylist(@Path("id") playlistId: String, @Body request: AddTrackToPlaylistRequest): Response<APIResponse<Playlist>>
    
    @DELETE("playlists/{id}/tracks/{trackId}")
    suspend fun removeTrackFromPlaylist(@Path("id") playlistId: String, @Path("trackId") trackId: String): Response<APIResponse<Playlist>>
    
    // Liked tracks endpoints
    @GET("liked-tracks")
    suspend fun getLikedTracks(): Response<APIResponse<List<Music>>>
    
    @POST("liked-tracks/{trackId}")
    suspend fun likeTrack(@Path("trackId") trackId: String): Response<APIResponse<Nothing>>
    
    @DELETE("liked-tracks/{trackId}")
    suspend fun unlikeTrack(@Path("trackId") trackId: String): Response<APIResponse<Nothing>>
    
    @GET("liked-tracks/{trackId}/check")
    suspend fun isTrackLiked(@Path("trackId") trackId: String): Response<APIResponse<Map<String, Boolean>>>
    
    // Health check
    @GET("health")
    suspend fun healthCheck(): Response<APIResponse<HealthResponse>>
}
