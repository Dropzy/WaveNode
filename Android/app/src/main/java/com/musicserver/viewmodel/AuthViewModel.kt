package com.musicserver.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.musicserver.data.models.User
import com.musicserver.data.repository.AuthRepository
import com.google.gson.Gson
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class AuthViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    private val gson: Gson
) : ViewModel() {
    
    private val _currentUser = MutableStateFlow<User?>(null)
    val currentUser: StateFlow<User?> = _currentUser.asStateFlow()
    
    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()
    
    private val _errorMessage = MutableStateFlow<String?>(null)
    val errorMessage: StateFlow<String?> = _errorMessage.asStateFlow()
    
    private val _isAuthenticated = MutableStateFlow(false)
    val isAuthenticated: StateFlow<Boolean> = _isAuthenticated.asStateFlow()
    
    init {
        checkAuthenticationStatus()
    }
    
    private fun checkAuthenticationStatus() {
        viewModelScope.launch {
            authRepository.getToken().collect { token ->
                if (!token.isNullOrEmpty()) {
                    // Token exists, try to get current user
                    authRepository.getCurrentUser()
                        .onSuccess { user ->
                            _currentUser.value = user
                            _isAuthenticated.value = true
                        }
                        .onFailure { _ ->
                            // Token might be invalid, clear it
                            authRepository.logout()
                            _isAuthenticated.value = false
                        }
                } else {
                    _isAuthenticated.value = false
                }
            }
        }
    }
    
    fun login(username: String, password: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _errorMessage.value = null
            
            println("Attempting login for user: $username")
            
            authRepository.login(username, password)
                .onSuccess { authResponse ->
                    println("Login successful for user: ${authResponse.user.username}")
                    _currentUser.value = authResponse.user
                    _isAuthenticated.value = true
                }
                .onFailure { error ->
                    println("Login failed: ${error.message}")
                    _errorMessage.value = error.message ?: "Login failed"
                }
            
            _isLoading.value = false
        }
    }
    
    fun register(username: String, email: String, password: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _errorMessage.value = null
            
            authRepository.register(username, email, password)
                .onSuccess { authResponse ->
                    _currentUser.value = authResponse.user
                    _isAuthenticated.value = true
                }
                .onFailure { error ->
                    _errorMessage.value = error.message ?: "Registration failed"
                }
            
            _isLoading.value = false
        }
    }
    
    fun logout() {
        viewModelScope.launch {
            authRepository.logout()
            _currentUser.value = null
            _isAuthenticated.value = false
        }
    }
    
    fun clearError() {
        _errorMessage.value = null
    }
}
