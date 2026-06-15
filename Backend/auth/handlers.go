package auth

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"music-server/database"

	"github.com/golang-jwt/jwt/v5"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// AuthRequest types
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string        `json:"token"`
	User  database.User `json:"user"`
}

// AuthHandler handles authentication requests
type AuthHandler struct {
	db                *database.DB
	jwtSecret         []byte
	allowRegistration bool
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *database.DB, jwtSecret []byte) *AuthHandler {
	return &AuthHandler{
		db:                db,
		jwtSecret:         jwtSecret,
		allowRegistration: true,
	}
}

func (h *AuthHandler) SetRegistrationEnabled(enabled bool) {
	h.allowRegistration = enabled
}

func (h *AuthHandler) RegistrationEnabled() bool {
	return h.allowRegistration
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if !h.allowRegistration {
		writeAuthResponse(w, http.StatusForbidden, APIResponse{
			Success: false,
			Error:   "Public registration is disabled. Ask an administrator to create or enable accounts.",
		})
		return
	}

	setupRequired, err := h.db.IsSetupRequired()
	if err != nil {
		writeAuthResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	if setupRequired {
		writeAuthResponse(w, http.StatusConflict, APIResponse{
			Success: false,
			Error:   "Complete the server setup before creating additional accounts",
		})
		return
	}

	var req RegisterRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response := APIResponse{
			Success: false,
			Error:   "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		response := APIResponse{
			Success: false,
			Error:   "Username, email, and password are required",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create user
	user := &database.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     "user",
	}

	err = h.db.CreateUser(user, req.Password)
	if err != nil {
		response := APIResponse{
			Success: false,
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate token
	token, err := h.issueJWT(user.ID, r)
	if err != nil {
		response := APIResponse{
			Success: false,
			Error:   "Failed to generate token",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	authResponse := AuthResponse{
		Token: token,
		User:  *user,
	}

	response := APIResponse{
		Success: true,
		Message: "User registered successfully",
		Data:    authResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) GenerateJWT(userID string, request ...*http.Request) (string, error) {
	if len(request) > 0 && request[0] != nil {
		return h.issueJWT(userID, request[0])
	}
	return h.generateJWT(userID, "")
}

func writeAuthResponse(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response := APIResponse{
			Success: false,
			Error:   "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		response := APIResponse{
			Success: false,
			Error:   "Username and password are required",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate password
	user, err := h.db.ValidatePassword(req.Username, req.Password)
	if err != nil {
		response := APIResponse{
			Success: false,
			Error:   "Invalid username or password",
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate token
	token, err := h.issueJWT(user.ID, r)
	if err != nil {
		response := APIResponse{
			Success: false,
			Error:   "Failed to generate token",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	authResponse := AuthResponse{
		Token: token,
		User:  *user,
	}

	response := APIResponse{
		Success: true,
		Message: "Login successful",
		Data:    authResponse,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetCurrentUser handles getting current user info
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userIDValue := r.Context().Value("user_id")
	if userIDValue == nil {
		response := APIResponse{
			Success: false,
			Error:   "User ID not found in context",
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	userID, ok := userIDValue.(string)
	if !ok {
		response := APIResponse{
			Success: false,
			Error:   "Invalid user ID in context",
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	user, err := h.db.GetUserByID(userID)
	if err != nil {
		response := APIResponse{
			Success: false,
			Error:   "User not found",
		}
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := APIResponse{
		Success: true,
		Message: "User retrieved successfully",
		Data:    user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserFromContext(r)
	if err != nil {
		writeAuthResponse(w, http.StatusUnauthorized, APIResponse{Success: false, Error: err.Error()})
		return
	}
	var request struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeAuthResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
		return
	}
	if len(request.NewPassword) < 8 || strings.TrimSpace(request.CurrentPassword) == "" {
		writeAuthResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Current password and a new password of at least 8 characters are required"})
		return
	}
	user, err := h.db.GetUserByID(userID)
	if err != nil || !h.db.ValidatePasswordForUser(user, request.CurrentPassword) {
		writeAuthResponse(w, http.StatusUnauthorized, APIResponse{Success: false, Error: "Current password is incorrect"})
		return
	}
	if err := h.db.UpdateUserPassword(userID, request.NewPassword); err != nil {
		writeAuthResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	writeAuthResponse(w, http.StatusOK, APIResponse{Success: true, Message: "Password changed successfully"})
}

// JWT Functions

// generateJWT generates a JWT token for a user
func (h *AuthHandler) issueJWT(userID string, r *http.Request) (string, error) {
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	userAgent := strings.TrimSpace(r.UserAgent())
	ipAddress := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ipAddress = host
	}
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		ipAddress = forwarded
	}
	if err := h.db.CreateSession(database.UserSession{
		ID: sessionID, UserID: userID, DeviceName: deviceNameFromUserAgent(userAgent),
		UserAgent: userAgent, IPAddress: ipAddress, ExpiresAt: expiresAt,
	}); err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	return h.generateJWT(userID, sessionID)
}

func deviceNameFromUserAgent(userAgent string) string {
	lower := strings.ToLower(userAgent)
	switch {
	case strings.Contains(lower, "android"):
		return "Android device"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		return "Apple mobile device"
	case strings.Contains(lower, "windows"):
		return "Windows browser"
	case strings.Contains(lower, "macintosh"):
		return "Mac browser"
	case strings.Contains(lower, "linux"):
		return "Linux browser"
	case userAgent == "":
		return "WaveNode client"
	default:
		return "Web browser"
	}
}

func (h *AuthHandler) generateJWT(userID, sessionID string) (string, error) {
	user, err := h.db.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to load user for token: %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":      userID,
		"user_version": user.UpdatedAt.Unix(),
		"session_id":   sessionID,
		"exp":          time.Now().Add(30 * 24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(h.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT validates a JWT token and returns the user ID
func (h *AuthHandler) ValidateJWT(tokenString string) (string, error) {
	userID, _, err := h.ValidateJWTSession(tokenString)
	return userID, err
}

func (h *AuthHandler) ValidateJWTSession(tokenString string) (string, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return h.jwtSecret, nil
	})

	if err != nil {
		return "", "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid user ID in token")
		}
		version, ok := claims["user_version"].(float64)
		if !ok {
			return "", "", fmt.Errorf("token must be refreshed")
		}
		user, err := h.db.GetUserByID(userID)
		if err != nil {
			return "", "", fmt.Errorf("user no longer exists")
		}
		if user.UpdatedAt.Unix() > int64(version) {
			return "", "", fmt.Errorf("token has been invalidated")
		}
		sessionID, _ := claims["session_id"].(string)
		if sessionID != "" && !h.db.IsSessionActive(sessionID, userID) {
			return "", "", fmt.Errorf("session has been revoked")
		}
		return userID, sessionID, nil
	}

	return "", "", fmt.Errorf("invalid token")
}
