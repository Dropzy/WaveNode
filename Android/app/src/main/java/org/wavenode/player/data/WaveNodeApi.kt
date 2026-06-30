package org.wavenode.player.data

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import java.util.concurrent.TimeUnit
import java.util.Locale

class WaveNodeApi(
    private val client: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build(),
) {
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }
    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    suspend fun login(serverUrl: String, username: String, password: String): SavedSession = withContext(Dispatchers.IO) {
        val normalizedServerUrl = normalizeServerUrl(serverUrl)
        val body = json.encodeToString(LoginRequest(username, password)).toRequestBody(jsonMediaType)
        val request = Request.Builder()
            .url("$normalizedServerUrl/api/auth/login")
            .addHeader("User-Agent", "WaveNode Android")
            .post(body)
            .build()

        client.newCall(request).execute().use { response ->
            val responseBody = response.body?.string().orEmpty()
            val apiResponse = json.decodeFromString<ApiResponse<AuthResponse>>(responseBody)
            if (!response.isSuccessful || !apiResponse.success || apiResponse.data == null) {
                throw IllegalStateException(apiResponse.error ?: "Login failed")
            }
            SavedSession(
                serverUrl = normalizedServerUrl,
                token = apiResponse.data.token,
                username = apiResponse.data.user.username,
            )
        }
    }

    suspend fun getTracks(session: SavedSession): List<Track> = withContext(Dispatchers.IO) {
        getList(session, "/api/music", "Could not load tracks")
    }

    suspend fun getAlbums(session: SavedSession): List<Album> = withContext(Dispatchers.IO) {
        getList(session, "/api/albums", "Could not load albums")
    }

    suspend fun getArtists(session: SavedSession): List<Artist> = withContext(Dispatchers.IO) {
        getList(session, "/api/artists", "Could not load artists")
    }

    suspend fun getPlaylists(session: SavedSession): List<Playlist> = withContext(Dispatchers.IO) {
        getList(session, "/api/playlists", "Could not load playlists")
    }

    suspend fun getPluginHomeRows(session: SavedSession): List<PluginHomeRow> = withContext(Dispatchers.IO) {
        getList(session, "/api/plugins/home-rows", "Could not load plugin rows")
    }

    suspend fun searchPodcasts(session: SavedSession, query: String): PodcastSearchResponse = withContext(Dispatchers.IO) {
        getObject(
            session,
            "/api/podcasts/search?q=${queryValue(query)}&page_size=20",
            "Could not search podcasts",
        )
    }

    suspend fun getPodcastHome(session: SavedSession): PodcastHomeResponse = withContext(Dispatchers.IO) {
        val country = Locale.getDefault().country.lowercase().takeIf { it.length == 2 } ?: "us"
        getObject(session, "/api/podcasts/home?country=$country", "Could not load podcasts")
    }

    suspend fun getPodcastEpisodes(session: SavedSession, podcastId: String): PodcastEpisodesResponse = withContext(Dispatchers.IO) {
        getObject(
            session,
            "/api/podcasts/${pathSegment(podcastId)}/episodes?limit=100",
            "Could not load podcast episodes",
        )
    }

    suspend fun updatePodcastProgress(session: SavedSession, progress: PodcastProgress): PodcastProgress = withContext(Dispatchers.IO) {
        val body = json.encodeToString(progress).toRequestBody(jsonMediaType)
        val request = authorizedRequest(session, "/api/podcasts/progress").put(body).build()
        client.newCall(request).execute().use { response ->
            val responseBody = response.body?.string().orEmpty()
            val apiResponse = json.decodeFromString<ApiResponse<PodcastProgress>>(responseBody)
            if (!response.isSuccessful || !apiResponse.success || apiResponse.data == null) {
                throw IllegalStateException(apiResponse.error ?: "Could not save podcast progress")
            }
            apiResponse.data
        }
    }

	suspend fun getPodcastPreferences(session: SavedSession): PodcastPreferences = withContext(Dispatchers.IO) {
		getObject(session, "/api/podcasts/preferences", "Could not load podcast preferences")
	}

	suspend fun updatePodcastPreferences(session: SavedSession, preferences: PodcastPreferences): PodcastPreferences = withContext(Dispatchers.IO) {
		putObject(session, "/api/podcasts/preferences", preferences, "Could not save podcast preferences")
	}

	suspend fun savePodcastSubscription(session: SavedSession, subscription: PodcastSubscription): PodcastSubscription = withContext(Dispatchers.IO) {
		val body = json.encodeToString(subscription).toRequestBody(jsonMediaType)
		val request = authorizedRequest(session, "/api/podcasts/subscriptions").post(body).build()
		client.newCall(request).execute().use { response ->
			val responseBody = response.body?.string().orEmpty()
			val apiResponse = json.decodeFromString<ApiResponse<PodcastSubscription>>(responseBody)
			if (!response.isSuccessful || !apiResponse.success || apiResponse.data == null) {
				throw IllegalStateException(apiResponse.error ?: "Could not follow podcast")
			}
			apiResponse.data
		}
	}

	suspend fun deletePodcastSubscription(session: SavedSession, podcastId: String) = withContext(Dispatchers.IO) {
		val request = authorizedRequest(session, "/api/podcasts/subscriptions/${pathSegment(podcastId)}")
			.delete()
			.build()
		client.newCall(request).execute().use { response ->
			if (!response.isSuccessful && response.code != 204) {
				throw IllegalStateException(apiErrorMessage(response.body?.string().orEmpty(), "Could not unfollow podcast"))
			}
		}
	}

    suspend fun getAlbumTracks(session: SavedSession, albumId: String): AlbumTracksResponse = withContext(Dispatchers.IO) {
        getObject(session, "/api/albums/${pathSegment(albumId)}/tracks", "Could not load album")
    }

	suspend fun createCastURL(session: SavedSession, trackId: String): CastURL = withContext(Dispatchers.IO) {
		val body = "{\"track_id\":${json.encodeToString(trackId)}}".toRequestBody(jsonMediaType)
		val request = authorizedRequest(session, "/api/outputs/cast-url").post(body).build()
		client.newCall(request).execute().use { response ->
			val payload = json.decodeFromString<ApiResponse<CastURL>>(response.body?.string().orEmpty())
			if (!response.isSuccessful || !payload.success || payload.data == null) throw IllegalStateException(payload.error ?: "Could not prepare cast playback")
			payload.data
		}
	}

	suspend fun discoverOutputDevices(session: SavedSession): List<OutputDevice> = withContext(Dispatchers.IO) {
		getList(session, "/api/outputs/devices", "Could not discover output devices")
	}

	suspend fun playOnDLNADevice(session: SavedSession, deviceId: String, mediaUrl: String, title: String): OutputDevice = withContext(Dispatchers.IO) {
		val body = json.encodeToString(OutputPlayRequest(deviceId, mediaUrl, title)).toRequestBody(jsonMediaType)
		val request = authorizedRequest(session, "/api/outputs/dlna/play").post(body).build()
		client.newCall(request).execute().use { response ->
			val payload = json.decodeFromString<ApiResponse<OutputDevice>>(response.body?.string().orEmpty())
			if (!response.isSuccessful || !payload.success || payload.data == null) throw IllegalStateException(payload.error ?: "Could not play on renderer")
			payload.data
		}
	}

    suspend fun getArtistTracks(session: SavedSession, artistIdOrName: String): ArtistTracksResponse = withContext(Dispatchers.IO) {
        getObject(session, "/api/artists/${pathSegment(artistIdOrName)}/tracks", "Could not load artist")
    }

    suspend fun getPlaylistTracks(session: SavedSession, playlistId: String): List<Track> = withContext(Dispatchers.IO) {
        getList(session, "/api/playlists/${pathSegment(playlistId)}/tracks", "Could not load playlist")
    }

    suspend fun createPlaylist(session: SavedSession, name: String, description: String): Playlist = withContext(Dispatchers.IO) {
        val body = json.encodeToString(
            PlaylistRequest(
                name = name,
                description = description,
            ),
        ).toRequestBody(jsonMediaType)
        val request = authorizedRequest(session, "/api/playlists")
            .post(body)
            .build()
        client.newCall(request).execute().use { response ->
            val responseBody = response.body?.string().orEmpty()
            val apiResponse = json.decodeFromString<ApiResponse<Playlist>>(responseBody)
            if (!response.isSuccessful || !apiResponse.success || apiResponse.data == null) {
                throw IllegalStateException(apiResponse.error ?: "Could not create playlist")
            }
            apiResponse.data
        }
    }

    suspend fun updatePlaylist(session: SavedSession, playlist: Playlist): Playlist = withContext(Dispatchers.IO) {
        val body = json.encodeToString(
            PlaylistRequest(
                name = playlist.name,
                description = playlist.description,
                imageUrl = playlist.imageUrl,
                type = playlist.type,
                trackIds = playlist.trackIds,
            ),
        ).toRequestBody(jsonMediaType)
        val request = authorizedRequest(session, "/api/playlists/${pathSegment(playlist.id)}")
            .put(body)
            .build()
        client.newCall(request).execute().use { response ->
            val responseBody = response.body?.string().orEmpty()
            val apiResponse = json.decodeFromString<ApiResponse<Playlist>>(responseBody)
            if (!response.isSuccessful || !apiResponse.success || apiResponse.data == null) {
                throw IllegalStateException(apiResponse.error ?: "Could not update playlist")
            }
            apiResponse.data
        }
    }

    suspend fun deletePlaylist(session: SavedSession, playlistId: String) = withContext(Dispatchers.IO) {
        val request = authorizedRequest(session, "/api/playlists/${pathSegment(playlistId)}")
            .delete()
            .build()
        client.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                val responseBody = response.body?.string().orEmpty()
                throw IllegalStateException(apiErrorMessage(responseBody, "Could not delete playlist"))
            }
        }
    }

    suspend fun addTrackToPlaylist(session: SavedSession, playlistId: String, trackId: String): Playlist = withContext(Dispatchers.IO) {
        val body = json.encodeToString(PlaylistTrackRequest(trackId)).toRequestBody(jsonMediaType)
        val request = authorizedRequest(session, "/api/playlists/${pathSegment(playlistId)}/tracks")
            .post(body)
            .build()
        client.newCall(request).execute().use { response ->
            val responseBody = response.body?.string().orEmpty()
            val apiResponse = json.decodeFromString<ApiResponse<Playlist>>(responseBody)
            if (!response.isSuccessful || !apiResponse.success || apiResponse.data == null) {
                throw IllegalStateException(apiResponse.error ?: "Could not add track to playlist")
            }
            apiResponse.data
        }
    }

    suspend fun addTracksToPlaylist(session: SavedSession, playlistId: String, trackIds: List<String>): Playlist = withContext(Dispatchers.IO) {
        val body = json.encodeToString(PlaylistTracksRequest(trackIds)).toRequestBody(jsonMediaType)
        val request = authorizedRequest(session, "/api/playlists/${pathSegment(playlistId)}/tracks/bulk")
            .post(body)
            .build()
        client.newCall(request).execute().use { response ->
            val responseBody = response.body?.string().orEmpty()
            val apiResponse = json.decodeFromString<ApiResponse<Playlist>>(responseBody)
            if (!response.isSuccessful || !apiResponse.success || apiResponse.data == null) {
                throw IllegalStateException(apiResponse.error ?: "Could not add tracks to playlist")
            }
            apiResponse.data
        }
    }

    suspend fun getRadioMetadata(session: SavedSession, streamUrl: String): RadioMetadataResponse = withContext(Dispatchers.IO) {
        getObject(session, "/api/plugins/radio-metadata?stream_url=${queryValue(streamUrl)}", "Could not load radio metadata")
    }

    suspend fun getSessions(session: SavedSession): UserSessionsResponse = withContext(Dispatchers.IO) {
        getObject(session, "/api/auth/sessions", "Could not load connected devices")
    }

    suspend fun createPlaybackHandoff(
        session: SavedSession,
        targetSessionId: String,
        trackIds: List<String>,
        startIndex: Int,
        action: String = "play_queue",
        positionMs: Long? = null,
    ) = withContext(Dispatchers.IO) {
        val body = json.encodeToString(
            PlaybackHandoffRequest(
                targetSessionId = targetSessionId,
                trackIds = trackIds,
                startIndex = startIndex,
                action = action,
                positionMs = positionMs,
            ),
        ).toRequestBody(jsonMediaType)
        val request = authorizedRequest(session, "/api/playback/connect")
            .post(body)
            .build()
        client.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                val responseBody = response.body?.string().orEmpty()
                throw IllegalStateException(apiErrorMessage(responseBody, "Could not connect to device"))
            }
        }
    }

    suspend fun consumePendingPlaybackHandoff(session: SavedSession): PlaybackHandoffCommand? = withContext(Dispatchers.IO) {
        val request = authorizedRequest(session, "/api/playback/connect/pending")
            .get()
            .build()
        client.newCall(request).execute().use { response ->
            val responseBody = response.body?.string().orEmpty()
            val apiResponse = json.decodeFromString<ApiResponse<PlaybackHandoffCommand?>>(responseBody)
            if (!response.isSuccessful || !apiResponse.success) {
                throw IllegalStateException(apiResponse.error ?: "Could not check connected playback")
            }
            apiResponse.data
        }
    }

    suspend fun addRecentlyPlayed(session: SavedSession, trackId: String) = withContext(Dispatchers.IO) {
        val request = authorizedRequest(
            session,
            "/api/recently-played/$trackId?source=mobile&device=Android",
        )
            .post(ByteArray(0).toRequestBody(null))
            .build()

        client.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                val responseBody = response.body?.string().orEmpty()
                throw IllegalStateException(apiErrorMessage(responseBody, "Could not record play"))
            }
        }
    }

    fun streamUrl(session: SavedSession, track: Track): String {
        if (track.isExternal && track.streamUrl.isNotBlank()) {
            return track.streamUrl
        }
        return "${session.serverUrl}/api/music/${track.id}/stream"
    }

    fun artworkUrl(session: SavedSession, track: Track): String? {
        val raw = listOf(
            track.imageUrl,
            track.coverArtLargeUrl,
            track.coverArtMediumUrl,
            track.coverArtSmallUrl,
            track.coverArtUrl,
        ).firstOrNull { it.isNotBlank() } ?: return null

        return artworkAbsoluteUrl(session, raw)
    }

    fun artworkUrl(session: SavedSession, album: Album): String? {
        val raw = listOf(
            album.coverArtLargeUrl,
            album.coverArtMediumUrl,
            album.coverArtSmallUrl,
            album.coverArtUrl,
        ).firstOrNull { it.isNotBlank() } ?: return null

        return artworkAbsoluteUrl(session, raw)
    }

    fun artworkUrl(session: SavedSession, artist: Artist): String? {
        val raw = listOf(
            artist.imageMediumUrl,
            artist.imageLargeUrl,
            artist.imageSmallUrl,
            artist.imageUrl,
        ).firstOrNull { it.isNotBlank() } ?: return null

        return artworkAbsoluteUrl(session, raw)
    }

    fun authHeaders(session: SavedSession): Map<String, String> {
        return mapOf("Authorization" to "Bearer ${session.token}")
    }

    fun playbackHeaders(session: SavedSession, track: Track): Map<String, String> {
        return if (track.isExternal) emptyMap() else authHeaders(session)
    }

    private fun authorizedRequest(session: SavedSession, path: String): Request.Builder {
        return Request.Builder()
            .url("${session.serverUrl}$path")
            .addHeader("Authorization", "Bearer ${session.token}")
            .addHeader("User-Agent", "WaveNode Android")
    }

    private inline fun <reified T> getList(session: SavedSession, path: String, fallbackError: String): List<T> {
        val request = authorizedRequest(session, path).get().build()
        client.newCall(request).execute().use { response ->
            val responseBody = response.body?.string().orEmpty()
            val apiResponse = json.decodeFromString<ApiResponse<List<T>>>(responseBody)
            if (!response.isSuccessful || !apiResponse.success) {
                throw IllegalStateException(apiResponse.error ?: fallbackError)
            }
            return apiResponse.data.orEmpty()
        }
    }

    private inline fun <reified T> getObject(session: SavedSession, path: String, fallbackError: String): T {
        val request = authorizedRequest(session, path).get().build()
        client.newCall(request).execute().use { response ->
            val responseBody = response.body?.string().orEmpty()
            val apiResponse = json.decodeFromString<ApiResponse<T>>(responseBody)
            if (!response.isSuccessful || !apiResponse.success || apiResponse.data == null) {
                throw IllegalStateException(apiResponse.error ?: fallbackError)
            }
            return apiResponse.data
        }
    }

	private inline fun <reified RequestType, reified ResponseType> putObject(
		session: SavedSession,
		path: String,
		value: RequestType,
		fallbackError: String,
	): ResponseType {
		val body = json.encodeToString(value).toRequestBody(jsonMediaType)
		val request = authorizedRequest(session, path).put(body).build()
		client.newCall(request).execute().use { response ->
			val responseBody = response.body?.string().orEmpty()
			val apiResponse = json.decodeFromString<ApiResponse<ResponseType>>(responseBody)
			if (!response.isSuccessful || !apiResponse.success || apiResponse.data == null) {
				throw IllegalStateException(apiResponse.error ?: fallbackError)
			}
			return apiResponse.data
		}
	}

    private fun absoluteUrl(session: SavedSession, raw: String): String {
        return if (raw.startsWith("http://") || raw.startsWith("https://")) {
            raw
        } else {
            "${session.serverUrl}/${raw.trimStart('/')}"
        }
    }

    private fun artworkAbsoluteUrl(session: SavedSession, raw: String): String? {
        val trimmed = raw.trim()
        if (trimmed.isBlank() || trimmed.lowercase().contains("default-track.png")) {
            return null
        }
        if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
            return trimmed
        }
        if (trimmed.startsWith("/artwork/")) {
            return "${session.serverUrl}/api${trimmed}"
        }
        if (trimmed.startsWith("/api/")) {
            return "${session.serverUrl}$trimmed"
        }
        if (trimmed.startsWith("artwork/")) {
            val filename = trimmed.substringAfterLast("/")
            return "${session.serverUrl}/api/artwork/${pathSegment(filename)}"
        }
        if (!trimmed.contains("/")) {
            return "${session.serverUrl}/api/artwork/${pathSegment(trimmed)}"
        }
        return absoluteUrl(session, trimmed)
    }

    private fun normalizeServerUrl(serverUrl: String): String {
        val withScheme = if (serverUrl.startsWith("http://") || serverUrl.startsWith("https://")) {
            serverUrl
        } else {
            "http://$serverUrl"
        }
        val uri = URI(withScheme.trim())
        return URI(uri.scheme, uri.authority, null, null, null).toString().trimEnd('/')
    }

    private fun pathSegment(value: String): String {
        return URLEncoder.encode(value, StandardCharsets.UTF_8.toString()).replace("+", "%20")
    }

    private fun queryValue(value: String): String {
        return URLEncoder.encode(value, StandardCharsets.UTF_8.toString()).replace("+", "%20")
    }

    private fun apiErrorMessage(responseBody: String, fallbackError: String): String {
        return runCatching {
            json.parseToJsonElement(responseBody)
                .jsonObject["error"]
                ?.jsonPrimitive
                ?.contentOrNull
        }.getOrNull()?.takeIf { it.isNotBlank() } ?: fallbackError
    }
}
