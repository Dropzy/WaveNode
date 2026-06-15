package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"music-server/database"
)

// AuthMiddleware creates middleware that validates JWT tokens
func (h *AuthHandler) AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response := APIResponse{
					Success: false,
					Error:   "Authorization header required",
				}
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			// Extract token from "Bearer <token>"
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				response := APIResponse{
					Success: false,
					Error:   "Invalid authorization header format",
				}
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			userID, sessionID, err := h.ValidateJWTSession(tokenParts[1])
			if err != nil {
				response := APIResponse{
					Success: false,
					Error:   "Invalid token",
				}
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			// Add user ID to request context
			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "session_id", sessionID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// AdminMiddleware creates middleware that requires admin role
func (h *AuthHandler) AdminMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			if user.Role != "admin" {
				response := APIResponse{
					Success: false,
					Error:   "Admin access required",
				}
				w.WriteHeader(http.StatusForbidden)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateJWTForWebSocket validates JWT token for WebSocket connections
func (h *AuthHandler) ValidateJWTForWebSocket(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("no token provided")
	}

	return h.ValidateJWT(token)
}

// GetUserFromContext extracts user ID from request context
func GetUserFromContext(r *http.Request) (string, error) {
	userIDValue := r.Context().Value("user_id")
	if userIDValue == nil {
		return "", fmt.Errorf("user ID not found in context")
	}

	userID, ok := userIDValue.(string)
	if !ok {
		return "", fmt.Errorf("invalid user ID in context")
	}

	return userID, nil
}

// ValidateUserInContext validates that user exists in database
func (h *AuthHandler) ValidateUserInContext(r *http.Request) (*database.User, error) {
	userID, err := GetUserFromContext(r)
	if err != nil {
		return nil, err
	}

	user, err := h.db.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}
