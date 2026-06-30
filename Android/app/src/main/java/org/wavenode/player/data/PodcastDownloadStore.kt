package org.wavenode.player.data

import android.content.Context
import android.net.Uri
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.OkHttpClient
import okhttp3.Request
import java.io.File
import java.security.MessageDigest
import java.util.concurrent.TimeUnit

class PodcastDownloadStore(context: Context) {
    private val directory = File(context.filesDir, "podcasts").apply { mkdirs() }
    private val preferences = context.getSharedPreferences("wavenode_podcast_downloads", Context.MODE_PRIVATE)
    private val client = OkHttpClient.Builder()
        .connectTimeout(20, TimeUnit.SECONDS)
        .readTimeout(5, TimeUnit.MINUTES)
        .build()

    fun downloadedEpisodeIds(): Set<String> = preferences.getStringSet(KEY_EPISODES, emptySet()).orEmpty().toSet()

    fun isDownloaded(track: Track): Boolean = track.id in downloadedEpisodeIds() && episodeFile(track.id).isFile

    fun withLocalAudio(track: Track): Track {
        val file = episodeFile(track.id)
        return if (track.id in downloadedEpisodeIds() && file.isFile) {
            track.copy(
				streamUrl = Uri.fromFile(file).toString(),
				podcastAudioUrl = track.podcastAudioUrl.ifBlank { track.streamUrl },
			)
        } else {
            track
        }
    }

    suspend fun download(track: Track): Track = withContext(Dispatchers.IO) {
        require(track.externalKind == "podcast" && track.streamUrl.startsWith("http")) { "Episode is not downloadable" }
        val destination = episodeFile(track.id)
        if (!destination.isFile) {
            val temporary = File(destination.absolutePath + ".part")
            val request = Request.Builder().url(track.streamUrl).header("User-Agent", "WaveNode Android").build()
            client.newCall(request).execute().use { response ->
                if (!response.isSuccessful || response.body == null) {
                    throw IllegalStateException("Episode download failed (${response.code})")
                }
                temporary.outputStream().use { output -> response.body!!.byteStream().use { it.copyTo(output) } }
            }
            if (temporary.length() == 0L || (!temporary.renameTo(destination) && !temporary.copyTo(destination, overwrite = true).exists())) {
                temporary.delete()
                throw IllegalStateException("Episode download could not be saved")
            }
            temporary.delete()
        }
        saveEpisodeId(track.id, true)
        withLocalAudio(track)
    }

    fun delete(trackId: String) {
        episodeFile(trackId).delete()
        File(episodeFile(trackId).absolutePath + ".part").delete()
        saveEpisodeId(trackId, false)
    }

    private fun saveEpisodeId(trackId: String, downloaded: Boolean) {
        val updated = downloadedEpisodeIds().toMutableSet().apply {
            if (downloaded) add(trackId) else remove(trackId)
        }
        preferences.edit().putStringSet(KEY_EPISODES, updated).apply()
    }

    private fun episodeFile(trackId: String): File {
        val digest = MessageDigest.getInstance("SHA-256").digest(trackId.toByteArray())
        val name = digest.joinToString("") { "%02x".format(it) }
        return File(directory, "$name.audio")
    }

    private companion object {
        const val KEY_EPISODES = "episode_ids"
    }
}
