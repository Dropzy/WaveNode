package com.musicserver.data.models

import android.os.Parcelable
import kotlinx.parcelize.Parcelize
import com.google.gson.annotations.SerializedName

@Parcelize
data class Music(
    val id: String,
    val title: String,
    val artist: String,
    val album: String,
    val genre: String,
    val duration: Long,
    @SerializedName("release_date")
    val releaseDate: String,
    @SerializedName("file_path")
    val filePath: String,
    @SerializedName("created_at")
    val createdAt: String,
    @SerializedName("updated_at")
    val updatedAt: String
) : Parcelable

@Parcelize
data class Playlist(
    val id: String,
    val name: String,
    val description: String,
    @SerializedName("track_ids")
    val trackIds: List<String>,
    @SerializedName("created_at")
    val createdAt: String,
    @SerializedName("updated_at")
    val updatedAt: String
) : Parcelable

data class PlaylistWithTracks(
    val playlist: Playlist,
    val tracks: List<Music>
)

@Parcelize
data class Album(
    val name: String,
    val artist: String,
    val year: Int,
    val tracks: List<Music>
) : Parcelable

@Parcelize
data class AlbumInfo(
    val name: String,
    val artist: String,
    val year: Int
) : Parcelable

data class AlbumTracksResponse(
    val album: AlbumInfo,
    val tracks: List<Music>
)

@Parcelize
data class ArtistInfo(
    val name: String,
    @SerializedName("track_count")
    val trackCount: Int,
    @SerializedName("album_count")
    val albumCount: Int
) : Parcelable

data class ArtistTracksResponse(
    val artist: ArtistInfo,
    val tracks: List<Music>,
    val albums: List<String>
)

data class CreatePlaylistRequest(
    val name: String,
    val description: String,
    @SerializedName("track_ids")
    val trackIds: List<String> = emptyList()
)

data class UpdatePlaylistRequest(
    val name: String? = null,
    val description: String? = null,
    @SerializedName("track_ids")
    val trackIds: List<String>? = null
)

data class AddTrackToPlaylistRequest(
    @SerializedName("track_id")
    val trackId: String
)

data class HealthResponse(
    val status: String,
    val timestamp: String,
    val version: String,
    @SerializedName("database_status")
    val databaseStatus: String,
    @SerializedName("database_error")
    val databaseError: String?
)
