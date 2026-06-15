package com.musicserver.player

import android.content.Context
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultDataSource
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.datasource.HttpDataSource
import androidx.media3.datasource.DataSource
import com.musicserver.data.local.TokenManager
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.runBlocking
import javax.inject.Inject
import javax.inject.Singleton

@UnstableApi
@Singleton
class AuthenticatedDataSourceFactory @Inject constructor(
    @ApplicationContext private val context: Context,
    private val tokenManager: TokenManager
) : HttpDataSource.Factory {

    private var defaultRequestProperties: Map<String, String> = emptyMap()

    override fun setDefaultRequestProperties(defaultRequestProperties: Map<String, String>): HttpDataSource.Factory {
        this.defaultRequestProperties = defaultRequestProperties
        return this
    }

    override fun createDataSource(): HttpDataSource {
        return try {
            val token = runBlocking { tokenManager.getTokenSync() }
            android.util.Log.d("AuthenticatedDataSourceFactory", "Creating data source with token: ${if (token != null) "present" else "null"}")
            val dataSourceFactory = DefaultHttpDataSource.Factory()
            
            // Set timeouts
            dataSourceFactory.setConnectTimeoutMs(30000)
            dataSourceFactory.setReadTimeoutMs(30000)
            
            // Set authentication header if token is available
            if (token != null) {
                val headers = mutableMapOf(
                    "Authorization" to "Bearer $token",
                    "User-Agent" to "MusicServer-Android"
                )
                // Add any default properties
                headers.putAll(defaultRequestProperties)
                android.util.Log.d("AuthenticatedDataSourceFactory", "Setting headers: $headers")
                dataSourceFactory.setDefaultRequestProperties(headers)
            } else {
                // Fallback without token
                val headers = mutableMapOf(
                    "User-Agent" to "MusicServer-Android"
                )
                // Add any default properties
                headers.putAll(defaultRequestProperties)
                android.util.Log.w("AuthenticatedDataSourceFactory", "No token available, using headers: $headers")
                dataSourceFactory.setDefaultRequestProperties(headers)
            }
            
            dataSourceFactory.createDataSource()
        } catch (e: Exception) {
            android.util.Log.e("AuthenticatedDataSourceFactory", "Error creating authenticated data source", e)
            // Fallback to basic data source if token retrieval fails
            DefaultHttpDataSource.Factory().apply {
                setConnectTimeoutMs(30000)
                setReadTimeoutMs(30000)
                val headers = mutableMapOf(
                    "User-Agent" to "MusicServer-Android"
                )
                headers.putAll(defaultRequestProperties)
                setDefaultRequestProperties(headers)
            }.createDataSource()
        }
    }
}

@UnstableApi
@Singleton
class AuthenticatedMediaDataSourceFactory @Inject constructor(
    @ApplicationContext private val context: Context,
    private val authenticatedDataSourceFactory: AuthenticatedDataSourceFactory
) : DataSource.Factory {
    
    override fun createDataSource(): DataSource {
        return DefaultDataSource.Factory(context, authenticatedDataSourceFactory).createDataSource()
    }
}
