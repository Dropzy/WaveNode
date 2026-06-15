package com.musicserver.data.local

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.firstOrNull
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

val Context.tokenDataStore: DataStore<Preferences> by preferencesDataStore(name = "token_prefs")

@Singleton
class TokenManager @Inject constructor(
    @ApplicationContext private val context: Context
) {
    private val dataStore = context.tokenDataStore
    private val tokenKey = stringPreferencesKey("auth_token")
    private val userKey = stringPreferencesKey("user_data")

    suspend fun saveToken(token: String) {
        dataStore.edit { preferences ->
            preferences[tokenKey] = token
        }
    }

    suspend fun saveUser(userJson: String) {
        dataStore.edit { preferences ->
            preferences[userKey] = userJson
        }
    }

    fun getToken(): Flow<String?> {
        return dataStore.data.map { preferences ->
            preferences[tokenKey]
        }
    }

    suspend fun getTokenSync(): String? {
        return dataStore.data.map { preferences ->
            preferences[tokenKey]
        }.firstOrNull()
    }

    fun getUser(): Flow<String?> {
        return dataStore.data.map { preferences ->
            preferences[userKey]
        }
    }

    suspend fun clearToken() {
        dataStore.edit { preferences ->
            preferences.remove(tokenKey)
            preferences.remove(userKey)
        }
    }
}
