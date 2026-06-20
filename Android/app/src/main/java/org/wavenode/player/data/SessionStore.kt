package org.wavenode.player.data

import android.content.Context

class SessionStore(context: Context) {
    private val preferences = context.getSharedPreferences("wavenode_session", Context.MODE_PRIVATE)

    fun load(): SavedSession? {
        val serverUrl = preferences.getString("server_url", null)?.takeIf { it.isNotBlank() } ?: return null
        val token = preferences.getString("token", null)?.takeIf { it.isNotBlank() } ?: return null
        val username = preferences.getString("username", null).orEmpty()
        return SavedSession(serverUrl = serverUrl, token = token, username = username)
    }

    fun save(session: SavedSession) {
        preferences.edit()
            .putString("server_url", session.serverUrl)
            .putString("token", session.token)
            .putString("username", session.username)
            .apply()
    }

    fun clear() {
        preferences.edit().clear().apply()
    }
}
