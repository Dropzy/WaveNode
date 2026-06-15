package com.musicserver.data.models

import android.os.Parcelable
import kotlinx.parcelize.Parcelize
import com.google.gson.annotations.SerializedName

@Parcelize
data class User(
    val id: String,
    val username: String,
    val email: String,
    val role: UserRole,
    @SerializedName("created_at")
    val createdAt: String,
    @SerializedName("updated_at")
    val updatedAt: String
) : Parcelable

enum class UserRole {
    @SerializedName("admin")
    ADMIN,
    @SerializedName("user")
    USER
}

@Parcelize
data class AuthResponse(
    val token: String,
    val user: User
) : Parcelable

data class LoginRequest(
    val username: String,
    val password: String
)

data class RegisterRequest(
    val username: String,
    val email: String,
    val password: String
)

data class APIResponse<T>(
    val success: Boolean,
    val message: String,
    val data: T?,
    val error: String?
)
