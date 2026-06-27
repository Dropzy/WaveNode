package org.wavenode.player.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ApiResponse<T>(
    val success: Boolean = false,
    val message: String? = null,
    val data: T? = null,
    val error: String? = null,
)

@Serializable
data class LoginRequest(
    val username: String,
    val password: String,
)

@Serializable
data class AuthResponse(
    val token: String,
    val user: User,
)

@Serializable
data class User(
    val id: String,
    val username: String,
    val email: String? = null,
    val role: String? = null,
)

@Serializable
data class Track(
    val id: String,
    val title: String,
    val artist: String,
    val album: String = "",
    val genre: String = "",
    val duration: Int = 0,
    @SerialName("release_date")
    val releaseDate: String? = null,
    @SerialName("file_name")
    val fileName: String = "",
    val format: String = "",
    @SerialName("image_url")
    val imageUrl: String = "",
    @SerialName("cover_art_url")
    val coverArtUrl: String = "",
    @SerialName("cover_art_small_url")
    val coverArtSmallUrl: String = "",
    @SerialName("cover_art_medium_url")
    val coverArtMediumUrl: String = "",
    @SerialName("cover_art_large_url")
    val coverArtLargeUrl: String = "",
    @SerialName("stream_url")
    val streamUrl: String = "",
    @SerialName("is_external")
    val isExternal: Boolean = false,
    @SerialName("external_kind")
    val externalKind: String = "",
    @SerialName("podcast_id")
    val podcastId: String = "",
    @SerialName("podcast_title")
    val podcastTitle: String = "",
    @SerialName("podcast_publisher")
    val podcastPublisher: String = "",
    @SerialName("podcast_episode_id")
    val podcastEpisodeId: String = "",
    @SerialName("podcast_description")
    val podcastDescription: String = "",
    @SerialName("podcast_website_url")
    val podcastWebsiteUrl: String = "",
    @SerialName("podcast_progress_seconds")
    val podcastProgressSeconds: Int = 0,
    @SerialName("podcast_completed")
    val podcastCompleted: Boolean = false,
    @SerialName("upload_order")
    val uploadOrder: Long = 0L,
    @SerialName("created_at")
    val createdAt: String = "",
)

@Serializable
data class Album(
    val id: String = "",
    val name: String,
    val artist: String = "",
    @SerialName("track_count")
    val trackCount: Int = 0,
    val year: Int = 0,
    @SerialName("cover_art_url")
    val coverArtUrl: String = "",
    @SerialName("cover_art_small_url")
    val coverArtSmallUrl: String = "",
    @SerialName("cover_art_medium_url")
    val coverArtMediumUrl: String = "",
    @SerialName("cover_art_large_url")
    val coverArtLargeUrl: String = "",
)

@Serializable
data class Artist(
    val id: String = "",
    val name: String = "",
    @SerialName("track_count")
    val trackCount: Int = 0,
    @SerialName("album_count")
    val albumCount: Int = 0,
    @SerialName("image_url")
    val imageUrl: String = "",
    @SerialName("image_small_url")
    val imageSmallUrl: String = "",
    @SerialName("image_medium_url")
    val imageMediumUrl: String = "",
    @SerialName("image_large_url")
    val imageLargeUrl: String = "",
)

@Serializable
data class Playlist(
    val id: String = "",
    val name: String = "",
    val description: String = "",
    @SerialName("image_url")
    val imageUrl: String = "",
    val type: String = "manual",
    @SerialName("track_ids")
    val trackIds: List<String> = emptyList(),
)

@Serializable
data class PlaylistRequest(
    val name: String,
    val description: String = "",
    @SerialName("image_url")
    val imageUrl: String = "",
    val type: String = "manual",
    @SerialName("track_ids")
    val trackIds: List<String> = emptyList(),
)

@Serializable
data class AlbumTracksResponse(
    val album: Album,
    val tracks: List<Track> = emptyList(),
)

@Serializable
data class ArtistTracksResponse(
    val artist: Artist,
    val tracks: List<Track> = emptyList(),
    val albums: List<Album> = emptyList(),
)

@Serializable
data class RadioMetadataResponse(
    @SerialName("station_title")
    val stationTitle: String = "",
    @SerialName("stream_title")
    val streamTitle: String = "",
    val error: String = "",
)

@Serializable
data class UserSession(
    val id: String = "",
    @SerialName("device_name")
    val deviceName: String = "",
    @SerialName("user_agent")
    val userAgent: String = "",
    @SerialName("ip_address")
    val ipAddress: String = "",
    @SerialName("last_seen_at")
    val lastSeenAt: String = "",
)

@Serializable
data class UserSessionsResponse(
    val sessions: List<UserSession> = emptyList(),
    @SerialName("current_session_id")
    val currentSessionId: String = "",
)

@Serializable
data class PlaybackHandoffRequest(
    @SerialName("target_session_id")
    val targetSessionId: String,
    @SerialName("track_ids")
    val trackIds: List<String> = emptyList(),
    @SerialName("start_index")
    val startIndex: Int = 0,
    val action: String = "play_queue",
    @SerialName("position_ms")
    val positionMs: Long? = null,
)

@Serializable
data class PlaylistTrackRequest(
    @SerialName("track_id")
    val trackId: String,
)

@Serializable
data class PlaylistTracksRequest(
    @SerialName("track_ids")
    val trackIds: List<String>,
)

@Serializable
data class PlaybackHandoffCommand(
    val id: String = "",
    @SerialName("source_session_id")
    val sourceSessionId: String = "",
    @SerialName("target_session_id")
    val targetSessionId: String = "",
    @SerialName("track_ids")
    val trackIds: List<String> = emptyList(),
    val tracks: List<Track> = emptyList(),
    @SerialName("start_index")
    val startIndex: Int = 0,
    val action: String = "play_queue",
    @SerialName("position_ms")
    val positionMs: Long = 0L,
)

@Serializable
data class PluginRowItem(
    val id: String = "",
    val title: String = "",
    val subtitle: String = "",
    val description: String = "",
    @SerialName("image_url")
    val imageUrl: String = "",
    @SerialName("stream_url")
    val streamUrl: String = "",
    @SerialName("homepage_url")
    val homepageUrl: String = "",
)

@Serializable
data class PluginHomeRow(
    @SerialName("plugin_id")
    val pluginId: String = "",
    val id: String = "",
    val title: String = "",
    val subtitle: String = "",
    val type: String = "",
    val items: List<PluginRowItem> = emptyList(),
)

@Serializable
data class Podcast(
    val id: String,
    val title: String = "",
    val publisher: String = "",
    val description: String = "",
    @SerialName("image_url")
    val imageUrl: String = "",
    @SerialName("thumbnail_url")
    val thumbnailUrl: String = "",
    @SerialName("website_url")
    val websiteUrl: String = "",
    @SerialName("total_episodes")
    val totalEpisodes: Int = 0,
    val explicit: Boolean = false,
)

@Serializable
data class PodcastSearchResponse(
    val query: String = "",
    val total: Int = 0,
    val count: Int = 0,
    val results: List<Podcast> = emptyList(),
)

@Serializable
data class PodcastEpisode(
    val id: String,
    val title: String = "",
    val description: String = "",
    @SerialName("audio_url")
    val audioUrl: String = "",
    @SerialName("website_url")
    val websiteUrl: String = "",
    @SerialName("image_url")
    val imageUrl: String = "",
    @SerialName("published_at")
    val publishedAt: String = "",
    val duration: Int = 0,
    val explicit: Boolean = false,
    @SerialName("progress_seconds")
    val progressSeconds: Int = 0,
    val completed: Boolean = false,
)

@Serializable
data class PodcastEpisodesResponse(
    val podcast: Podcast,
    val count: Int = 0,
    val episodes: List<PodcastEpisode> = emptyList(),
)

@Serializable
data class PodcastProgress(
    @SerialName("podcast_id") val podcastId: String,
    @SerialName("episode_id") val episodeId: String,
    @SerialName("podcast_title") val podcastTitle: String,
    val publisher: String = "",
    @SerialName("episode_title") val episodeTitle: String,
    val description: String = "",
    @SerialName("image_url") val imageUrl: String = "",
    @SerialName("audio_url") val audioUrl: String,
    @SerialName("website_url") val websiteUrl: String = "",
    @SerialName("published_at") val publishedAt: String? = null,
    @SerialName("duration_seconds") val durationSeconds: Int = 0,
    @SerialName("position_seconds") val positionSeconds: Int = 0,
    val completed: Boolean = false,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class PodcastHomeResponse(
    @SerialName("continue_listening") val continueListening: List<PodcastProgress> = emptyList(),
    @SerialName("top_podcasts") val topPodcasts: List<Podcast> = emptyList(),
)

data class SavedSession(
    val serverUrl: String,
    val token: String,
    val username: String,
)
