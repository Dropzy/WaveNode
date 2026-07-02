package router

import (
	"encoding/json"
	"net/http"

	"music-server/auth"

	"github.com/gorilla/mux"
)

func (r *Router) getRating(w http.ResponseWriter, req *http.Request) {
	userID, err := auth.GetUserFromContext(req)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}
	id := mux.Vars(req)["id"]
	track, err := r.db.GetMusic(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Track not found")
		return
	}
	allowed, accessErr := r.requestCanAccessMusic(req, track)
	if accessErr != nil || !allowed {
		writeJSONError(w, http.StatusNotFound, "Track not found")
		return
	}
	rating, err := r.db.GetMediaRating(userID, id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]int{"rating": rating}})
}

func (r *Router) setRating(w http.ResponseWriter, req *http.Request) {
	userID, err := auth.GetUserFromContext(req)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}
	id := mux.Vars(req)["id"]
	track, err := r.db.GetMusic(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Track not found")
		return
	}
	allowed, accessErr := r.requestCanAccessMusic(req, track)
	if accessErr != nil || !allowed {
		writeJSONError(w, http.StatusNotFound, "Track not found")
		return
	}
	var payload struct {
		Rating int `json:"rating"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload.Rating < 0 || payload.Rating > 5 {
		writeJSONError(w, http.StatusBadRequest, "Rating must be an integer between 0 and 5")
		return
	}
	if err := r.db.SetMediaRating(userID, id, "song", payload.Rating); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]int{"rating": payload.Rating}})
}
