package router

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"music-server/database"
)

func (r *Router) subsonicSetRating(user *database.User, req *http.Request) *subsonicError {
	id := strings.TrimSpace(req.FormValue("id"))
	if id == "" {
		return missingSubsonicParameter("id")
	}
	rating, err := strconv.Atoi(req.FormValue("rating"))
	if err != nil || rating < 0 || rating > 5 {
		return &subsonicError{Code: 10, Message: "Rating must be an integer between 0 and 5"}
	}
	mediaType := "song"
	if _, err := r.db.GetMusic(id); err != nil {
		if _, err := r.db.GetAlbumByID(id); err == nil {
			mediaType = "album"
		} else if _, err := r.db.GetArtistByID(id); err == nil {
			mediaType = "artist"
		} else {
			return notFoundSubsonicError("Media")
		}
	}
	if err := r.db.SetMediaRating(user.ID, id, mediaType, rating); err != nil {
		return internalSubsonicError(err)
	}
	return nil
}

func (r *Router) subsonicBookmarks(user *database.User) (map[string]interface{}, *subsonicError) {
	bookmarks, err := r.db.GetBookmarks(user.ID)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	items := make([]interface{}, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		track, err := r.db.GetMusic(bookmark.TrackID)
		if err != nil {
			continue
		}
		items = append(items, map[string]interface{}{
			"position": bookmark.PositionMS, "username": user.Username,
			"comment": bookmark.Comment, "created": bookmark.UpdatedAt.Format(time.RFC3339),
			"changed": bookmark.UpdatedAt.Format(time.RFC3339), "entry": subsonicSongMap(*track),
		})
	}
	return map[string]interface{}{"bookmarks": map[string]interface{}{"bookmark": items}}, nil
}

func (r *Router) subsonicCreateBookmark(user *database.User, req *http.Request) *subsonicError {
	id := req.FormValue("id")
	if id == "" {
		return missingSubsonicParameter("id")
	}
	if _, err := r.db.GetMusic(id); err != nil {
		return notFoundSubsonicError("Song")
	}
	position, err := strconv.ParseInt(req.FormValue("position"), 10, 64)
	if err != nil || position < 0 {
		return &subsonicError{Code: 10, Message: "Bookmark position must be a non-negative integer"}
	}
	if err := r.db.SaveBookmark(user.ID, id, position, req.FormValue("comment")); err != nil {
		return internalSubsonicError(err)
	}
	return nil
}

func (r *Router) subsonicDeleteBookmark(user *database.User, id string) *subsonicError {
	if id == "" {
		return missingSubsonicParameter("id")
	}
	if err := r.db.DeleteBookmark(user.ID, id); err != nil {
		return internalSubsonicError(err)
	}
	return nil
}

func (r *Router) subsonicPlayQueue(user *database.User) (map[string]interface{}, *subsonicError) {
	queue, err := r.db.GetPlayQueue(user.ID)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	entries := make([]interface{}, 0, len(queue.TrackIDs))
	for _, id := range queue.TrackIDs {
		if track, err := r.db.GetMusic(id); err == nil {
			entries = append(entries, subsonicSongMap(*track))
		}
	}
	return map[string]interface{}{"playQueue": map[string]interface{}{
		"current": queue.CurrentTrackID, "position": queue.PositionMS,
		"username": user.Username, "changed": queue.ChangedAt.Format(time.RFC3339), "entry": entries,
	}}, nil
}

func (r *Router) subsonicSavePlayQueue(user *database.User, req *http.Request) *subsonicError {
	position, err := strconv.ParseInt(defaultString(req.FormValue("position"), "0"), 10, 64)
	if err != nil || position < 0 {
		return &subsonicError{Code: 10, Message: "Queue position must be a non-negative integer"}
	}
	queue := database.PlayQueue{
		TrackIDs: append([]string(nil), req.Form["id"]...), CurrentTrackID: req.FormValue("current"),
		PositionMS: position,
	}
	if err := r.db.SavePlayQueue(user.ID, queue); err != nil {
		return internalSubsonicError(err)
	}
	return nil
}

func (r *Router) subsonicTopSongs(req *http.Request) (map[string]interface{}, *subsonicError) {
	artist := strings.TrimSpace(req.FormValue("artist"))
	if artist == "" {
		return nil, missingSubsonicParameter("artist")
	}
	tracks, err := r.db.GetArtistTracks(artist)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	sort.SliceStable(tracks, func(i, j int) bool { return tracks[i].PlayCount > tracks[j].PlayCount })
	if count := queryInt(req, "count", 50, 1, 500); count < len(tracks) {
		tracks = tracks[:count]
	}
	return map[string]interface{}{"topSongs": map[string]interface{}{"song": subsonicSongs(tracks)}}, nil
}

func (r *Router) subsonicSimilarSongs(req *http.Request, method string) (map[string]interface{}, *subsonicError) {
	id := req.FormValue("id")
	source, err := r.db.GetMusic(id)
	if err != nil {
		return nil, notFoundSubsonicError("Song")
	}
	all, err := r.db.GetAllMusic()
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	similar := make([]database.Music, 0)
	for _, track := range all {
		if track.ID != source.ID && strings.EqualFold(track.Genre, source.Genre) {
			similar = append(similar, track)
		}
	}
	sort.SliceStable(similar, func(i, j int) bool { return similar[i].PlayCount > similar[j].PlayCount })
	if count := queryInt(req, "count", 50, 1, 500); count < len(similar) {
		similar = similar[:count]
	}
	key := "similarSongs2"
	if method == "getSimilarSongs" {
		key = "similarSongs"
	}
	return map[string]interface{}{key: map[string]interface{}{"song": subsonicSongs(similar)}}, nil
}

func (r *Router) subsonicArtistInfo(id, method string) (map[string]interface{}, *subsonicError) {
	artist, err := r.db.GetArtistByID(id)
	if err != nil {
		return nil, notFoundSubsonicError("Artist")
	}
	key := "artistInfo2"
	if method == "getArtistInfo" {
		key = "artistInfo"
	}
	return map[string]interface{}{key: map[string]interface{}{
		"biography": artist.Biography, "musicBrainzId": artist.MusicBrainzID,
		"smallImageUrl": artist.ImageSmallURL, "mediumImageUrl": artist.ImageMediumURL,
		"largeImageUrl": artist.ImageLargeURL, "similarArtist": []interface{}{},
	}}, nil
}

func (r *Router) subsonicAlbumInfo(id, method string) (map[string]interface{}, *subsonicError) {
	album, err := r.db.GetAlbumByID(id)
	if err != nil {
		return nil, notFoundSubsonicError("Album")
	}
	key := "albumInfo2"
	if method == "getAlbumInfo" {
		key = "albumInfo"
	}
	return map[string]interface{}{key: map[string]interface{}{
		"notes": "", "smallImageUrl": album.CoverArtSmallURL,
		"mediumImageUrl": album.CoverArtMediumURL, "largeImageUrl": album.CoverArtLargeURL,
	}}, nil
}

func (r *Router) subsonicUsers(requester *database.User) (map[string]interface{}, *subsonicError) {
	if requester.Role != "admin" {
		return nil, &subsonicError{Code: 50, Message: "Administrator access is required"}
	}
	users, err := r.db.GetAllUsers()
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	items := make([]interface{}, 0, len(users))
	for index := range users {
		items = append(items, subsonicUser(&users[index]))
	}
	return map[string]interface{}{"users": map[string]interface{}{"user": items}}, nil
}

func (r *Router) subsonicGetUser(requester *database.User, username string) (map[string]interface{}, *subsonicError) {
	if username == "" {
		username = requester.Username
	}
	if requester.Role != "admin" && username != requester.Username {
		return nil, &subsonicError{Code: 50, Message: "Not authorized to view this user"}
	}
	target, err := r.db.GetUserByUsername(username)
	if err != nil {
		return nil, notFoundSubsonicError("User")
	}
	return map[string]interface{}{"user": subsonicUser(target)}, nil
}

func (r *Router) subsonicCreateUser(requester *database.User, req *http.Request) *subsonicError {
	if requester.Role != "admin" {
		return &subsonicError{Code: 50, Message: "Administrator access is required"}
	}
	username := strings.TrimSpace(req.FormValue("username"))
	password, decodeErr := decodeSubsonicPassword(req.FormValue("password"))
	if decodeErr != nil {
		return &subsonicError{Code: 10, Message: "Password encoding is invalid"}
	}
	if username == "" || password == "" {
		return missingSubsonicParameter("username/password")
	}
	role := "user"
	if req.FormValue("adminRole") == "true" {
		role = "admin"
	}
	email := strings.TrimSpace(req.FormValue("email"))
	if email == "" {
		email = username + "@local.invalid"
	}
	if err := r.db.CreateUser(&database.User{Username: username, Email: email, Role: role}, password); err != nil {
		return internalSubsonicError(err)
	}
	return nil
}

func (r *Router) subsonicUpdateUser(requester *database.User, req *http.Request) *subsonicError {
	if requester.Role != "admin" {
		return &subsonicError{Code: 50, Message: "Administrator access is required"}
	}
	target, err := r.db.GetUserByUsername(req.FormValue("username"))
	if err != nil {
		return notFoundSubsonicError("User")
	}
	role := "user"
	if req.FormValue("adminRole") == "true" {
		role = "admin"
	}
	if err := r.db.UpdateUserRole(target.ID, role); err != nil {
		if errors.Is(err, database.ErrLastAdministrator) {
			return &subsonicError{Code: 50, Message: err.Error()}
		}
		return internalSubsonicError(err)
	}
	return nil
}

func (r *Router) subsonicDeleteUser(requester *database.User, username string) *subsonicError {
	if requester.Role != "admin" {
		return &subsonicError{Code: 50, Message: "Administrator access is required"}
	}
	target, err := r.db.GetUserByUsername(username)
	if err != nil {
		return notFoundSubsonicError("User")
	}
	if err := r.db.DeleteUser(target.ID); err != nil {
		if errors.Is(err, database.ErrLastAdministrator) {
			return &subsonicError{Code: 50, Message: err.Error()}
		}
		return internalSubsonicError(err)
	}
	return nil
}

func (r *Router) subsonicChangePassword(requester *database.User, req *http.Request) *subsonicError {
	username := req.FormValue("username")
	target, err := r.db.GetUserByUsername(username)
	if err != nil {
		return notFoundSubsonicError("User")
	}
	if requester.Role != "admin" && requester.ID != target.ID {
		return &subsonicError{Code: 50, Message: "Not authorized to change this password"}
	}
	password, decodeErr := decodeSubsonicPassword(req.FormValue("password"))
	if decodeErr != nil {
		return &subsonicError{Code: 10, Message: "Password encoding is invalid"}
	}
	if password == "" {
		return missingSubsonicParameter("password")
	}
	if err := r.db.UpdateUserPassword(target.ID, password); err != nil {
		return internalSubsonicError(err)
	}
	return nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
