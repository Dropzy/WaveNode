package org.wavenode.player.data

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withPermit
import okhttp3.OkHttpClient
import okhttp3.Request
import java.net.Inet4Address
import java.net.NetworkInterface
import java.net.URI
import java.util.concurrent.TimeUnit

data class DiscoveredServer(
    val name: String,
    val url: String,
)

class ServerDiscovery(
    private val client: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(600, TimeUnit.MILLISECONDS)
        .readTimeout(900, TimeUnit.MILLISECONDS)
        .callTimeout(1200, TimeUnit.MILLISECONDS)
        .build(),
) {
    suspend fun discover(): List<DiscoveredServer> = coroutineScope {
        val candidates = buildCandidates()
        val limiter = Semaphore(48)

        candidates.map { url ->
            async(Dispatchers.IO) {
                limiter.withPermit { validate(url) }
            }
        }
            .awaitAll()
            .filterNotNull()
            .distinctBy { it.url }
            .sortedBy { it.url }
    }

    private fun buildCandidates(): List<String> {
        val urls = linkedSetOf("http://10.0.2.2:8080")
        localSubnetPrefixes().forEach { prefix ->
            (1..254).forEach { host ->
                urls.add("http://$prefix.$host:8080")
                urls.add("http://$prefix.$host")
            }
        }
        return urls.toList()
    }

    private fun localSubnetPrefixes(): Set<String> {
        return NetworkInterface.getNetworkInterfaces()
            .toList()
            .filter { it.isUp && !it.isLoopback }
            .flatMap { networkInterface ->
                networkInterface.inetAddresses.toList()
                    .filterIsInstance<Inet4Address>()
                    .mapNotNull { address ->
                        val parts = address.hostAddress.orEmpty().split(".")
                        if (parts.size == 4 && isPrivateAddress(parts)) {
                            parts.take(3).joinToString(".")
                        } else {
                            null
                        }
                    }
            }
            .toSet()
    }

    private fun isPrivateAddress(parts: List<String>): Boolean {
        val first = parts.getOrNull(0)?.toIntOrNull() ?: return false
        val second = parts.getOrNull(1)?.toIntOrNull() ?: return false
        return first == 10 || (first == 172 && second in 16..31) || (first == 192 && second == 168)
    }

    private fun validate(url: String): DiscoveredServer? {
        val request = Request.Builder()
            .url("$url/health")
            .get()
            .build()

        return runCatching {
            client.newCall(request).execute().use { response ->
                val body = response.body?.string().orEmpty()
                if (!response.isSuccessful || !body.contains("WaveNode", ignoreCase = true)) {
                    return null
                }

                val uri = URI(url)
                val host = uri.host ?: url
                val port = if (uri.port > 0) ":${uri.port}" else ""
                DiscoveredServer(
                    name = "WaveNode at $host$port",
                    url = url,
                )
            }
        }.getOrNull()
    }
}
