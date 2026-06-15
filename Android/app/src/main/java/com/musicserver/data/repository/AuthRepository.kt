package com.musicserver.data.repository

import com.musicserver.data.api.MusicApiService
import com.musicserver.data.local.TokenManager
import com.musicserver.data.models.*
import com.google.gson.Gson
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AuthRepository @Inject constructor(
    private val apiService: MusicApiService,
    private val tokenManager: TokenManager,
    private val gson: Gson
) {
    
    suspend fun login(username: String, password: String): Result<AuthResponse> {
        return try {
            println("Making login request for user: $username")
            val response = apiService.login(LoginRequest(username, password))
            println("Login response code: ${response.code()}")
            println("Login response body: ${response.body()}")
            
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    println("Login successful, saving token")
                    // Save token and user data
                    tokenManager.saveToken(apiResponse.data.token)
                    val userJson = gson.toJson(apiResponse.data.user)
                    tokenManager.saveUser(userJson)
                    Result.success(apiResponse.data)
                } else {
                    println("Login failed: ${apiResponse?.error}")
                    Result.failure(Exception(apiResponse?.error ?: "Login failed"))
                }
            } else {
                println("Login HTTP error: ${response.code()}")
                Result.failure(Exception("Login failed: ${response.code()}"))
            }
        } catch (e: Exception) {
            println("Login exception: ${e.message}")
            Result.failure(e)
        }
    }
    
    suspend fun register(username: String, email: String, password: String): Result<AuthResponse> {
        return try {
            val response = apiService.register(RegisterRequest(username, email, password))
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    // Save token and user data
                    tokenManager.saveToken(apiResponse.data.token)
                    val userJson = gson.toJson(apiResponse.data.user)
                    tokenManager.saveUser(userJson)
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Registration failed"))
                }
            } else {
                Result.failure(Exception("Registration failed: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun getCurrentUser(): Result<User> {
        return try {
            val response = apiService.getCurrentUser()
            if (response.isSuccessful) {
                val apiResponse = response.body()
                if (apiResponse?.success == true && apiResponse.data != null) {
                    Result.success(apiResponse.data)
                } else {
                    Result.failure(Exception(apiResponse?.error ?: "Failed to get user"))
                }
            } else {
                Result.failure(Exception("Failed to get user: ${response.code()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
    
    suspend fun logout() {
        tokenManager.clearToken()
    }
    
    fun getToken() = tokenManager.getToken()
    fun getUser() = tokenManager.getUser()
}
