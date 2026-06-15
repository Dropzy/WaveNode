package auth

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (h *AuthHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(string)
	currentSessionID, _ := r.Context().Value("session_id").(string)
	sessions, err := h.db.GetUserSessions(userID)
	if err != nil {
		writeAuthResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	writeAuthResponse(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"sessions": sessions, "current_session_id": currentSessionID,
	}})
}

func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(string)
	if err := h.db.RevokeSession(userID, mux.Vars(r)["id"]); err != nil {
		writeAuthResponse(w, http.StatusNotFound, APIResponse{Success: false, Error: err.Error()})
		return
	}
	writeAuthResponse(w, http.StatusOK, APIResponse{Success: true, Message: "Session revoked"})
}

func (h *AuthHandler) RevokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(string)
	currentSessionID, _ := r.Context().Value("session_id").(string)
	if err := h.db.RevokeOtherSessions(userID, currentSessionID); err != nil {
		writeAuthResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	writeAuthResponse(w, http.StatusOK, APIResponse{Success: true, Message: "Other sessions revoked"})
}
