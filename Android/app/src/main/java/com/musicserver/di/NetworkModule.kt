package com.musicserver.di

import android.content.Context
import com.google.gson.Gson
import com.google.gson.GsonBuilder
import com.musicserver.data.api.ApiClient
import com.musicserver.data.api.MusicApiService
import com.musicserver.data.local.TokenManager
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {
    
    @Provides
    @Singleton
    fun provideGson(): Gson {
        return GsonBuilder()
            .setDateFormat("yyyy-MM-dd'T'HH:mm:ss")
            .create()
    }
    
    @Provides
    @Singleton
    fun provideTokenManager(@ApplicationContext context: Context): TokenManager {
        return TokenManager(context)
    }
    
    @Provides
    @Singleton
    fun provideApiClient(tokenManager: TokenManager): ApiClient {
        return ApiClient(tokenManager)
    }
    
    @Provides
    @Singleton
    fun provideMusicApiService(apiClient: ApiClient): MusicApiService {
        return apiClient.musicApiService
    }
}
