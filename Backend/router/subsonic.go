package router

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"music-server/database"
	"music-server/handlers"
	"music-server/streaming"
	"music-server/utils"

	"github.com/gorilla/mux"
)

const subsonicAPIVersion = "1.16.1"

type subsonicError struct {
	Code    int
	Message string
}

type subsonicAuthCacheEntry struct {
	UserID      string
	UserVersion time.Time
	ExpiresAt   time.Time
}

func (r *Router) subsonicAPI(w http.ResponseWriter, req *http.Request) {
	method := strings.TrimSuffix(mux.Vars(req)["method"], ".view")
	format := strings.ToLower(req.FormValue("f"))
	if format == "" {
		format = "xml"
	}

	user, authErr := r.authenticateSubsonic(req)
	if authErr != nil {
		writeSubsonicError(w, format, authErr.Code, authErr.Message)
		return
	}
	req = req.WithContext(context.WithValue(req.Context(), "user_id", user.ID))
	if method != "stream" && method != "download" && method != "getCoverArt" {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}

	var data map[string]interface{}
	var err *subsonicError

	switch method {
	case "ping":
	case "getLicense":
		data = map[string]interface{}{"license": map[string]interface{}{"valid": true}}
	case "getOpenSubsonicExtensions":
		data = map[string]interface{}{"openSubsonicExtensions": map[string]interface{}{"openSubsonicExtension": []interface{}{}}}
	case "tokenInfo":
		err = &subsonicError{Code: 41, Message: "Token authentication is not supported"}
	case "getMusicFolders":
		data, err = r.subsonicMusicFolders(user)
	case "getGenres":
		data, err = r.subsonicGenres(req)
	case "getArtists":
		data, err = r.subsonicArtists(req)
	case "getIndexes":
		data, err = r.subsonicIndexes(req)
	case "getArtist":
		data, err = r.subsonicArtist(req, req.FormValue("id"))
	case "getAlbum":
		data, err = r.subsonicAlbum(req, req.FormValue("id"))
	case "getSong":
		data, err = r.subsonicSong(req, req.FormValue("id"))
	case "getMusicDirectory":
		data, err = r.subsonicMusicDirectory(req, req.FormValue("id"))
	case "getAlbumList", "getAlbumList2":
		data, err = r.subsonicAlbumList(req)
	case "getRandomSongs":
		data, err = r.subsonicRandomSongs(req)
	case "getSongsByGenre":
		data, err = r.subsonicSongsByGenre(req)
	case "search", "search2", "search3":
		data, err = r.subsonicSearch(req)
	case "getPlaylists":
		data, err = r.subsonicPlaylists(user)
	case "getPlaylist":
		data, err = r.subsonicPlaylist(user, req.FormValue("id"))
	case "createPlaylist":
		data, err = r.subsonicCreatePlaylist(user, req)
	case "updatePlaylist":
		err = r.subsonicUpdatePlaylist(user, req)
	case "deletePlaylist":
		err = r.subsonicDeletePlaylist(user, req.FormValue("id"))
	case "getStarred", "getStarred2":
		data, err = r.subsonicStarred(user, method)
	case "star":
		err = r.subsonicStar(user, req)
	case "unstar":
		err = r.subsonicUnstar(user, req)
	case "scrobble":
		err = r.subsonicScrobble(user, req.Form["id"])
	case "getNowPlaying":
		data = map[string]interface{}{"nowPlaying": map[string]interface{}{"entry": []interface{}{}}}
	case "getTopSongs":
		data, err = r.subsonicTopSongs(req)
	case "getSimilarSongs", "getSimilarSongs2":
		data, err = r.subsonicSimilarSongs(req, method)
	case "getArtistInfo", "getArtistInfo2":
		data, err = r.subsonicArtistInfo(req, req.FormValue("id"), method)
	case "getAlbumInfo", "getAlbumInfo2":
		data, err = r.subsonicAlbumInfo(req, req.FormValue("id"), method)
	case "getVideos":
		data = map[string]interface{}{"videos": map[string]interface{}{"video": []interface{}{}}}
	case "getVideoInfo":
		err = notFoundSubsonicError("Video")
	case "getLyrics":
		data = r.subsonicLyrics(req)
	case "getLyricsBySongId":
		data = r.subsonicStructuredLyrics(req)
	case "setRating":
		err = r.subsonicSetRating(user, req)
	case "getBookmarks":
		data, err = r.subsonicBookmarks(user)
	case "createBookmark":
		err = r.subsonicCreateBookmark(user, req)
	case "deleteBookmark":
		err = r.subsonicDeleteBookmark(user, req.FormValue("id"))
	case "getPlayQueue":
		data, err = r.subsonicPlayQueue(user)
	case "savePlayQueue":
		err = r.subsonicSavePlayQueue(user, req)
	case "getUsers":
		data, err = r.subsonicUsers(user)
	case "createUser":
		err = r.subsonicCreateUser(user, req)
	case "updateUser":
		err = r.subsonicUpdateUser(user, req)
	case "deleteUser":
		err = r.subsonicDeleteUser(user, req.FormValue("username"))
	case "changePassword":
		err = r.subsonicChangePassword(user, req)
	case "getShares":
		data = map[string]interface{}{"shares": map[string]interface{}{"share": []interface{}{}}}
	case "getPodcasts":
		data = map[string]interface{}{"podcasts": map[string]interface{}{"channel": []interface{}{}}}
	case "getNewestPodcasts":
		data = map[string]interface{}{"newestPodcasts": map[string]interface{}{"episode": []interface{}{}}}
	case "getInternetRadioStations":
		data, err = r.subsonicInternetRadioStations()
	case "getChatMessages":
		data = map[string]interface{}{"chatMessages": map[string]interface{}{"chatMessage": []interface{}{}}}
	case "getAvatar":
		err = notFoundSubsonicError("Avatar")
	case "hls", "getCaptions":
		err = &subsonicError{Code: 0, Message: "Video and transcoding are not enabled"}
	case "jukeboxControl":
		err = &subsonicError{Code: 50, Message: "Jukebox control is not enabled"}
	case "createShare", "updateShare", "deleteShare",
		"refreshPodcasts", "createPodcastChannel", "deletePodcastChannel", "deletePodcastEpisode", "downloadPodcastEpisode",
		"createInternetRadioStation", "updateInternetRadioStation", "deleteInternetRadioStation", "addChatMessage":
		err = &subsonicError{Code: 50, Message: "This optional server feature is not enabled"}
	case "getScanStatus":
		data, err = r.subsonicScanStatus()
	case "startScan":
		if user.Role != "admin" {
			err = &subsonicError{Code: 50, Message: "Administrator access is required"}
		} else {
			data, err = r.subsonicStartScan()
		}
	case "getUser":
		data, err = r.subsonicGetUser(user, req.FormValue("username"))
	case "stream", "download":
		r.subsonicStream(w, req, user, req.FormValue("id"), method == "download")
		return
	case "getCoverArt":
		r.subsonicCoverArt(w, req, req.FormValue("id"))
		return
	default:
		err = &subsonicError{Code: 0, Message: fmt.Sprintf("Unsupported Subsonic method: %s", method)}
	}

	if err != nil {
		writeSubsonicError(w, format, err.Code, err.Message)
		return
	}
	if data != nil && subsonicResponseIncludesMedia(method) {
		ratings, ratingErr := r.db.GetMediaRatings(user.ID)
		averageRatings, averageRatingErr := r.db.GetMediaAverageRatings()
		if ratingErr == nil && averageRatingErr == nil {
			applySubsonicRatings(data, ratings, averageRatings)
		}
	}
	writeSubsonicResponse(w, format, data)
}

func (r *Router) authenticateSubsonic(req *http.Request) (*database.User, *subsonicError) {
	username := strings.TrimSpace(req.FormValue("u"))
	version := strings.TrimSpace(req.FormValue("v"))
	client := strings.TrimSpace(req.FormValue("c"))
	if username == "" || version == "" || client == "" {
		return nil, &subsonicError{Code: 10, Message: "Required authentication parameter is missing"}
	}
	if !supportedSubsonicVersion(version) {
		return nil, &subsonicError{Code: 30, Message: "Client is using an unsupported Subsonic API version"}
	}
	if req.FormValue("t") != "" || req.FormValue("s") != "" {
		return nil, &subsonicError{Code: 41, Message: "Token authentication is not supported; use password authentication over HTTPS"}
	}

	password, err := decodeSubsonicPassword(req.FormValue("p"))
	if err != nil || password == "" {
		return nil, &subsonicError{Code: 40, Message: "Wrong username or password"}
	}
	cacheKey := subsonicCredentialCacheKey(username, password)
	if cachedValue, exists := r.subsonicAuthCache.Load(cacheKey); exists {
		cached := cachedValue.(subsonicAuthCacheEntry)
		if time.Now().Before(cached.ExpiresAt) {
			if user, userErr := r.db.GetUserByID(cached.UserID); userErr == nil && user.UpdatedAt.Equal(cached.UserVersion) {
				return user, nil
			}
		}
		r.subsonicAuthCache.Delete(cacheKey)
	}
	user, validationErr := r.db.ValidatePassword(username, password)
	if validationErr != nil {
		return nil, &subsonicError{Code: 40, Message: "Wrong username or password"}
	}
	r.subsonicAuthCache.Store(cacheKey, subsonicAuthCacheEntry{
		UserID:      user.ID,
		UserVersion: user.UpdatedAt,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	})
	return user, nil
}

func subsonicCredentialCacheKey(username, password string) string {
	sum := sha256.Sum256([]byte(username + "\x00" + password))
	return hex.EncodeToString(sum[:])
}

func supportedSubsonicVersion(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return false
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	return majorErr == nil && minorErr == nil && major == 1 && minor <= 16
}

func decodeSubsonicPassword(value string) (string, error) {
	if !strings.HasPrefix(value, "enc:") {
		return value, nil
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "enc:"))
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func (r *Router) subsonicMusicFolders(user *database.User) (map[string]interface{}, *subsonicError) {
	sources, err := r.db.GetMusicSources()
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	folders := make([]interface{}, 0, len(sources))
	for index, source := range sources {
		if user.Role != "admin" && user.LibraryRestricted && !stringSliceContains(user.MusicSourceIDs, source.ID) {
			continue
		}
		folders = append(folders, map[string]interface{}{
			"id":   index + 1,
			"name": filepath.Base(source.Path),
		})
	}
	return map[string]interface{}{"musicFolders": map[string]interface{}{"musicFolder": folders}}, nil
}

func (r *Router) subsonicGenres(req *http.Request) (map[string]interface{}, *subsonicError) {
	tracks, err := r.db.GetAllMusic()
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	tracks, err = r.filterMusicForRequest(req, tracks)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	counts := make(map[string]int)
	for _, track := range tracks {
		if genre := strings.TrimSpace(track.Genre); genre != "" {
			counts[genre]++
		}
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	genres := make([]interface{}, 0, len(names))
	for _, name := range names {
		genres = append(genres, map[string]interface{}{"value": name, "songCount": counts[name], "albumCount": 0})
	}
	return map[string]interface{}{"genres": map[string]interface{}{"genre": genres}}, nil
}

func (r *Router) subsonicArtists(req *http.Request) (map[string]interface{}, *subsonicError) {
	artists, err := r.db.GetAllArtistsForLibrary()
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	allowedTracks, filterErr := r.db.GetAllMusic()
	if filterErr == nil {
		allowedTracks, filterErr = r.filterMusicForRequest(req, allowedTracks)
	}
	if filterErr != nil {
		return nil, internalSubsonicError(filterErr)
	}
	allowedNames := make(map[string]bool)
	for _, track := range allowedTracks {
		allowedNames[strings.ToLower(strings.TrimSpace(track.Artist))] = true
	}
	indexes := make(map[string][]interface{})
	for _, artist := range artists {
		name := stringValue(artist["name"])
		if !allowedNames[strings.ToLower(strings.TrimSpace(name))] {
			continue
		}
		key := "#"
		if name != "" {
			first := strings.ToUpper(string([]rune(name)[0]))
			if first >= "A" && first <= "Z" {
				key = first
			}
		}
		indexes[key] = append(indexes[key], map[string]interface{}{
			"id":         stringValue(artist["id"]),
			"name":       name,
			"albumCount": intValue(artist["album_count"]),
			"coverArt":   stringValue(artist["id"]),
		})
	}
	keys := make([]string, 0, len(indexes))
	for key := range indexes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		result = append(result, map[string]interface{}{"name": key, "artist": indexes[key]})
	}
	return map[string]interface{}{"artists": map[string]interface{}{"ignoredArticles": "The El La Los Las Le Les", "index": result}}, nil
}

func (r *Router) subsonicIndexes(req *http.Request) (map[string]interface{}, *subsonicError) {
	artists, err := r.subsonicArtists(req)
	if err != nil {
		return nil, err
	}
	artistRoot := artists["artists"].(map[string]interface{})
	return map[string]interface{}{"indexes": map[string]interface{}{
		"ignoredArticles": artistRoot["ignoredArticles"],
		"lastModified":    time.Now().UnixMilli(),
		"index":           artistRoot["index"],
	}}, nil
}

func (r *Router) subsonicArtist(req *http.Request, id string) (map[string]interface{}, *subsonicError) {
	if id == "" {
		return nil, missingSubsonicParameter("id")
	}
	artist, err := r.db.GetArtistByID(id)
	if err != nil {
		return nil, notFoundSubsonicError("Artist")
	}
	artistTracks, trackErr := r.db.GetArtistTracks(artist.Name)
	if trackErr == nil {
		artistTracks, trackErr = r.filterMusicForRequest(req, artistTracks)
	}
	if trackErr != nil || len(artistTracks) == 0 {
		return nil, notFoundSubsonicError("Artist")
	}
	albums, dbErr := r.db.GetAllAlbums()
	if dbErr != nil {
		return nil, internalSubsonicError(dbErr)
	}
	albums, dbErr = r.filterAlbumsForRequest(req, albums)
	if dbErr != nil {
		return nil, internalSubsonicError(dbErr)
	}
	items := make([]interface{}, 0)
	for _, album := range albums {
		if strings.EqualFold(album.Artist, artist.Name) {
			items = append(items, subsonicAlbumMap(album))
		}
	}
	return map[string]interface{}{"artist": map[string]interface{}{
		"id": artist.ID, "name": artist.Name, "albumCount": len(items),
		"coverArt": artist.ID, "album": items,
	}}, nil
}

func (r *Router) subsonicAlbum(req *http.Request, id string) (map[string]interface{}, *subsonicError) {
	if id == "" {
		return nil, missingSubsonicParameter("id")
	}
	album, err := r.db.GetAlbumByID(id)
	if err != nil {
		return nil, notFoundSubsonicError("Album")
	}
	tracks, trackErr := r.db.GetAlbumTracksByID(id)
	if trackErr != nil {
		return nil, internalSubsonicError(trackErr)
	}
	tracks, trackErr = r.filterMusicForRequest(req, tracks)
	if trackErr != nil || len(tracks) == 0 {
		return nil, notFoundSubsonicError("Album")
	}
	item := subsonicAlbumMap(*album)
	songs := make([]interface{}, 0, len(tracks))
	duration := 0
	for _, track := range tracks {
		songs = append(songs, subsonicSongMap(track))
		duration += track.Duration
	}
	item["duration"] = duration
	item["song"] = songs
	return map[string]interface{}{"album": item}, nil
}

func (r *Router) subsonicSong(req *http.Request, id string) (map[string]interface{}, *subsonicError) {
	if id == "" {
		return nil, missingSubsonicParameter("id")
	}
	track, err := r.db.GetMusic(id)
	if err != nil {
		return nil, notFoundSubsonicError("Song")
	}
	allowed, accessErr := r.requestCanAccessMusic(req, track)
	if accessErr != nil || !allowed {
		return nil, notFoundSubsonicError("Song")
	}
	return map[string]interface{}{"song": subsonicSongMap(*track)}, nil
}

func (r *Router) subsonicMusicDirectory(req *http.Request, id string) (map[string]interface{}, *subsonicError) {
	if id == "" {
		return nil, missingSubsonicParameter("id")
	}
	if artist, err := r.db.GetArtistByID(id); err == nil {
		albums, albumErr := r.db.GetAllAlbums()
		if albumErr != nil {
			return nil, internalSubsonicError(albumErr)
		}
		albums, albumErr = r.filterAlbumsForRequest(req, albums)
		if albumErr != nil {
			return nil, internalSubsonicError(albumErr)
		}
		children := make([]interface{}, 0)
		for _, album := range albums {
			if strings.EqualFold(album.Artist, artist.Name) {
				children = append(children, subsonicAlbumMap(album))
			}
		}
		return map[string]interface{}{"directory": map[string]interface{}{"id": id, "name": artist.Name, "child": children}}, nil
	}
	if album, err := r.db.GetAlbumByID(id); err == nil {
		tracks, trackErr := r.db.GetAlbumTracksByID(id)
		if trackErr != nil {
			return nil, internalSubsonicError(trackErr)
		}
		tracks, trackErr = r.filterMusicForRequest(req, tracks)
		if trackErr != nil || len(tracks) == 0 {
			return nil, notFoundSubsonicError("Directory")
		}
		children := make([]interface{}, 0, len(tracks))
		for _, track := range tracks {
			children = append(children, subsonicSongMap(track))
		}
		return map[string]interface{}{"directory": map[string]interface{}{"id": id, "name": album.Name, "parent": album.ID, "child": children}}, nil
	}
	return nil, notFoundSubsonicError("Directory")
}

func (r *Router) subsonicAlbumList(req *http.Request) (map[string]interface{}, *subsonicError) {
	albums, err := r.db.GetAllAlbums()
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	albums, err = r.filterAlbumsForRequest(req, albums)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	listType := req.FormValue("type")
	switch listType {
	case "alphabeticalByArtist":
		sort.SliceStable(albums, func(i, j int) bool {
			return strings.ToLower(albums[i].Artist+" "+albums[i].Name) < strings.ToLower(albums[j].Artist+" "+albums[j].Name)
		})
	case "random":
		rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(albums), func(i, j int) { albums[i], albums[j] = albums[j], albums[i] })
	default:
		sort.SliceStable(albums, func(i, j int) bool { return strings.ToLower(albums[i].Name) < strings.ToLower(albums[j].Name) })
	}
	offset := queryInt(req, "offset", 0, 0, len(albums))
	size := queryInt(req, "size", 10, 1, 500)
	albums = sliceAlbums(albums, offset, size)
	items := make([]interface{}, 0, len(albums))
	for _, album := range albums {
		items = append(items, subsonicAlbumMap(album))
	}
	key := "albumList2"
	if strings.TrimSuffix(mux.Vars(req)["method"], ".view") == "getAlbumList" {
		key = "albumList"
	}
	return map[string]interface{}{key: map[string]interface{}{"album": items}}, nil
}

func (r *Router) subsonicRandomSongs(req *http.Request) (map[string]interface{}, *subsonicError) {
	tracks, err := r.db.GetAllMusic()
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	tracks, err = r.filterMusicForRequest(req, tracks)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(tracks), func(i, j int) { tracks[i], tracks[j] = tracks[j], tracks[i] })
	size := queryInt(req, "size", 10, 1, 500)
	if size < len(tracks) {
		tracks = tracks[:size]
	}
	return map[string]interface{}{"randomSongs": map[string]interface{}{"song": subsonicSongs(tracks)}}, nil
}

func (r *Router) subsonicSongsByGenre(req *http.Request) (map[string]interface{}, *subsonicError) {
	genre := strings.TrimSpace(req.FormValue("genre"))
	if genre == "" {
		return nil, missingSubsonicParameter("genre")
	}
	tracks, err := r.db.GetAllMusic()
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	tracks, err = r.filterMusicForRequest(req, tracks)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	filtered := make([]database.Music, 0)
	for _, track := range tracks {
		if strings.EqualFold(track.Genre, genre) {
			filtered = append(filtered, track)
		}
	}
	offset := queryInt(req, "offset", 0, 0, len(filtered))
	count := queryInt(req, "count", 10, 1, 500)
	filtered = sliceTracks(filtered, offset, count)
	return map[string]interface{}{"songsByGenre": map[string]interface{}{"song": subsonicSongs(filtered)}}, nil
}

func (r *Router) subsonicSearch(req *http.Request) (map[string]interface{}, *subsonicError) {
	query := strings.TrimSpace(req.FormValue("query"))
	if query == "" {
		query = strings.TrimSpace(req.FormValue("title"))
	}
	tracks, err := r.db.SearchMusic(query)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	tracks, err = r.filterMusicForRequest(req, tracks)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	allAlbums, albumErr := r.db.GetAllAlbums()
	if albumErr != nil {
		return nil, internalSubsonicError(albumErr)
	}
	allAlbums, albumErr = r.filterAlbumsForRequest(req, allAlbums)
	if albumErr != nil {
		return nil, internalSubsonicError(albumErr)
	}
	allArtists, artistErr := r.db.GetAllArtistsForLibrary()
	if artistErr != nil {
		return nil, internalSubsonicError(artistErr)
	}
	lower := strings.ToLower(query)
	albums := make([]interface{}, 0)
	for _, album := range allAlbums {
		if strings.Contains(strings.ToLower(album.Name+" "+album.Artist), lower) {
			albums = append(albums, subsonicAlbumMap(album))
		}
	}
	artists := make([]interface{}, 0)
	allowedArtistNames := make(map[string]bool)
	for _, track := range tracks {
		allowedArtistNames[strings.ToLower(strings.TrimSpace(track.Artist))] = true
	}
	for _, artist := range allArtists {
		name := stringValue(artist["name"])
		if allowedArtistNames[strings.ToLower(strings.TrimSpace(name))] && strings.Contains(strings.ToLower(name), lower) {
			artists = append(artists, map[string]interface{}{
				"id": stringValue(artist["id"]), "name": stringValue(artist["name"]),
				"albumCount": intValue(artist["album_count"]),
			})
		}
	}
	resultKey := "searchResult3"
	switch strings.TrimSuffix(mux.Vars(req)["method"], ".view") {
	case "search":
		resultKey = "searchResult"
	case "search2":
		resultKey = "searchResult2"
	}
	return map[string]interface{}{resultKey: map[string]interface{}{
		"artist": limitInterfaces(artists, queryInt(req, "artistCount", 20, 0, 500)),
		"album":  limitInterfaces(albums, queryInt(req, "albumCount", 20, 0, 500)),
		"song":   limitInterfaces(subsonicSongs(tracks), queryInt(req, "songCount", 20, 0, 500)),
	}}, nil
}

func (r *Router) subsonicPlaylists(user *database.User) (map[string]interface{}, *subsonicError) {
	playlists, err := r.db.GetUserPlaylists(user.ID)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	items := make([]interface{}, 0, len(playlists))
	for _, playlist := range playlists {
		items = append(items, subsonicPlaylistMap(playlist, user.Username, nil))
	}
	return map[string]interface{}{"playlists": map[string]interface{}{
		"lastModified": playlistRevision(playlists),
		"playlist":     items,
	}}, nil
}

func (r *Router) subsonicPlaylist(user *database.User, id string) (map[string]interface{}, *subsonicError) {
	playlist, err := r.db.GetUserPlaylist(id, user.ID)
	if err != nil {
		return nil, notFoundSubsonicError("Playlist")
	}
	tracks, trackErr := r.db.GetPlaylistTracks(id, user.ID)
	if trackErr != nil {
		return nil, internalSubsonicError(trackErr)
	}
	tracks, trackErr = r.db.FilterMusicForUser(user.ID, tracks)
	if trackErr != nil {
		return nil, internalSubsonicError(trackErr)
	}
	item := subsonicPlaylistMap(*playlist, user.Username, tracks)
	return map[string]interface{}{"playlist": item}, nil
}

func (r *Router) subsonicCreatePlaylist(user *database.User, req *http.Request) (map[string]interface{}, *subsonicError) {
	if playlistID := req.FormValue("playlistId"); playlistID != "" {
		playlist, err := r.db.GetUserPlaylist(playlistID, user.ID)
		if err != nil {
			return nil, notFoundSubsonicError("Playlist")
		}
		if playlist.Type == database.PlaylistTypeSmart {
			return nil, &subsonicError{Code: 50, Message: "Smart playlists are read-only"}
		}
		if name := strings.TrimSpace(req.FormValue("name")); name != "" {
			playlist.Name = name
		}
		playlist.TrackIDs = append([]string(nil), req.Form["songId"]...)
		if err := r.db.UpdatePlaylist(playlist); err != nil {
			return nil, internalSubsonicError(err)
		}
		return r.subsonicPlaylist(user, playlist.ID)
	}
	name := strings.TrimSpace(req.FormValue("name"))
	if name == "" {
		return nil, missingSubsonicParameter("name")
	}
	playlist := &database.Playlist{UserID: user.ID, Name: name, TrackIDs: append([]string(nil), req.Form["songId"]...)}
	if err := r.db.AddPlaylist(playlist); err != nil {
		return nil, internalSubsonicError(err)
	}
	return r.subsonicPlaylist(user, playlist.ID)
}

func (r *Router) subsonicUpdatePlaylist(user *database.User, req *http.Request) *subsonicError {
	id := req.FormValue("playlistId")
	if id == "" {
		return missingSubsonicParameter("playlistId")
	}
	playlist, err := r.db.GetUserPlaylist(id, user.ID)
	if err != nil {
		return notFoundSubsonicError("Playlist")
	}
	if playlist.Type == database.PlaylistTypeSmart {
		return &subsonicError{Code: 50, Message: "Smart playlists are read-only"}
	}
	if name := strings.TrimSpace(req.FormValue("name")); name != "" {
		playlist.Name = name
	}
	if comment := req.FormValue("comment"); comment != "" {
		playlist.Description = comment
	}
	removeIndexes := make([]int, 0)
	for _, value := range req.Form["songIndexToRemove"] {
		if index, parseErr := strconv.Atoi(value); parseErr == nil {
			removeIndexes = append(removeIndexes, index)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(removeIndexes)))
	for _, index := range removeIndexes {
		if index >= 0 && index < len(playlist.TrackIDs) {
			playlist.TrackIDs = append(playlist.TrackIDs[:index], playlist.TrackIDs[index+1:]...)
		}
	}
	playlist.TrackIDs = append(playlist.TrackIDs, req.Form["songIdToAdd"]...)
	if err := r.db.UpdatePlaylist(playlist); err != nil {
		return internalSubsonicError(err)
	}
	return nil
}

func (r *Router) subsonicDeletePlaylist(user *database.User, id string) *subsonicError {
	if id == "" {
		return missingSubsonicParameter("id")
	}
	playlist, err := r.db.GetUserPlaylist(id, user.ID)
	if err != nil {
		return notFoundSubsonicError("Playlist")
	}
	if playlist.Type == database.PlaylistTypeSmart {
		return &subsonicError{Code: 50, Message: "Smart playlists are read-only"}
	}
	if err := r.db.DeletePlaylist(id, user.ID); err != nil {
		return notFoundSubsonicError("Playlist")
	}
	return nil
}

func (r *Router) subsonicStarred(user *database.User, method string) (map[string]interface{}, *subsonicError) {
	tracks, err := r.db.GetLikedTracks(user.ID)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	tracks, err = r.db.FilterMusicForUser(user.ID, tracks)
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	artistIDs, err := r.db.GetStarredMedia(user.ID, "artist")
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	albumIDs, err := r.db.GetStarredMedia(user.ID, "album")
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	artists := make([]interface{}, 0, len(artistIDs))
	for _, id := range artistIDs {
		if artist, err := r.db.GetArtistByID(id); err == nil {
			artists = append(artists, map[string]interface{}{"id": artist.ID, "name": artist.Name, "coverArt": artist.ID})
		}
	}
	albums := make([]interface{}, 0, len(albumIDs))
	for _, id := range albumIDs {
		if album, err := r.db.GetAlbumByID(id); err == nil {
			albums = append(albums, subsonicAlbumMap(*album))
		}
	}
	key := "starred2"
	if method == "getStarred" {
		key = "starred"
	}
	return map[string]interface{}{key: map[string]interface{}{"artist": artists, "album": albums, "song": subsonicSongs(tracks)}}, nil
}

func (r *Router) subsonicStar(user *database.User, req *http.Request) *subsonicError {
	for _, id := range req.Form["id"] {
		track, trackErr := r.db.GetMusic(id)
		if trackErr != nil {
			return notFoundSubsonicError("Song")
		}
		if allowed, _ := r.db.UserCanAccessMusic(user.ID, track); !allowed {
			return notFoundSubsonicError("Song")
		}
		if err := r.db.LikeTrack(user.ID, id); err != nil {
			return internalSubsonicError(err)
		}
	}
	for _, id := range req.Form["albumId"] {
		if _, err := r.db.GetAlbumByID(id); err != nil {
			return notFoundSubsonicError("Album")
		}
		if err := r.db.StarMedia(user.ID, id, "album"); err != nil {
			return internalSubsonicError(err)
		}
	}
	for _, id := range req.Form["artistId"] {
		if _, err := r.db.GetArtistByID(id); err != nil {
			return notFoundSubsonicError("Artist")
		}
		if err := r.db.StarMedia(user.ID, id, "artist"); err != nil {
			return internalSubsonicError(err)
		}
	}
	return nil
}

func (r *Router) subsonicUnstar(user *database.User, req *http.Request) *subsonicError {
	for _, id := range req.Form["id"] {
		if err := r.db.UnlikeTrack(user.ID, id); err != nil && !strings.Contains(err.Error(), "not in liked") {
			return internalSubsonicError(err)
		}
	}
	for _, id := range append(req.Form["albumId"], req.Form["artistId"]...) {
		if err := r.db.UnstarMedia(user.ID, id); err != nil {
			return internalSubsonicError(err)
		}
	}
	return nil
}

func (r *Router) subsonicScrobble(user *database.User, ids []string) *subsonicError {
	for _, id := range ids {
		track, trackErr := r.db.GetMusic(id)
		if trackErr != nil {
			return notFoundSubsonicError("Song")
		}
		if allowed, _ := r.db.UserCanAccessMusic(user.ID, track); !allowed {
			return notFoundSubsonicError("Song")
		}
		if err := r.db.AddToRecentlyPlayedFrom(user.ID, id, "subsonic", "Subsonic client"); err != nil {
			return internalSubsonicError(err)
		}
		_ = r.db.IncrementPlayCount(id)
		if track, err := r.db.GetMusic(id); err == nil {
			go r.submitScrobble(user.ID, *track, "listened", time.Now())
		}
	}
	return nil
}

func (r *Router) subsonicScanStatus() (map[string]interface{}, *subsonicError) {
	scan, err := r.scanStore.GetCurrentScan()
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	status := map[string]interface{}{"scanning": false, "count": 0}
	if scan != nil && (scan.Status == "running" || scan.Status == "stopping") {
		status["scanning"] = true
		status["count"] = scan.Processed
	}
	return map[string]interface{}{"scanStatus": status}, nil
}

func (r *Router) subsonicStartScan() (map[string]interface{}, *subsonicError) {
	if handlers.ScannerInstance == nil {
		return nil, &subsonicError{Code: 0, Message: "Scanner is unavailable"}
	}
	if _, err := handlers.ScannerInstance.StartScan(); err != nil && !strings.Contains(err.Error(), "already") {
		return nil, internalSubsonicError(err)
	}
	return r.subsonicScanStatus()
}

func (r *Router) subsonicStream(w http.ResponseWriter, req *http.Request, user *database.User, id string, download bool) {
	track, err := r.db.GetMusic(id)
	if err != nil || track.FilePath == "" {
		w.WriteHeader(http.StatusNotFound)
		writeSubsonicError(w, req.FormValue("f"), 70, "Song not found")
		return
	}
	allowed, accessErr := r.db.UserCanAccessMusic(user.ID, track)
	if accessErr != nil || !allowed {
		writeSubsonicError(w, req.FormValue("f"), 70, "Song not found")
		return
	}
	profile, _ := r.db.GetPlaybackProfile(user.ID)
	properties, _ := r.db.GetTrackAudioProperties(track.ID)
	database.ApplyTrackAudioProperties(track, properties)
	maxBitrate, _ := strconv.Atoi(req.FormValue("maxBitRate"))
	requestedFormat := strings.ToLower(req.FormValue("format"))
	offset, _ := strconv.ParseFloat(req.FormValue("timeOffset"), 64)
	transcode := !download && (profile.TranscodeEnabled || maxBitrate > 0 || requestedFormat != "" || offset > 0)
	if requestedFormat == "raw" {
		transcode = false
	}
	gain := streaming.ReplayGainDB(profile, properties)
	if !download && gain != 0 {
		transcode = true
	}
	if transcode {
		if requestedFormat == "" || requestedFormat == "raw" {
			requestedFormat = profile.TranscodeFormat
		}
		if maxBitrate == 0 {
			maxBitrate = profile.TranscodeBitrate
		}
		if err := streaming.Serve(w, req, *track, streaming.Options{
			Format: requestedFormat, Bitrate: maxBitrate, Offset: offset, GainDB: gain,
		}); err != nil && req.Context().Err() == nil {
			writeSubsonicError(w, req.FormValue("f"), 0, "Transcoding failed")
		}
		return
	}
	file, openErr := os.Open(track.FilePath)
	if openErr != nil {
		w.WriteHeader(http.StatusNotFound)
		writeSubsonicError(w, req.FormValue("f"), 70, "Song file is unavailable")
		return
	}
	defer file.Close()
	info, statErr := file.Stat()
	if statErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeSubsonicError(w, req.FormValue("f"), 0, "Song file cannot be read")
		return
	}
	contentType := subsonicAudioContentType(track.Format, track.FilePath)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	if download {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(track.FilePath)))
	}
	http.ServeContent(w, req, info.Name(), info.ModTime(), file)
}

func (r *Router) subsonicCoverArt(w http.ResponseWriter, req *http.Request, id string) {
	if id == "" {
		writeSubsonicError(w, req.FormValue("f"), 10, "Required parameter 'id' is missing")
		return
	}
	value := ""
	if track, err := r.db.GetMusic(id); err == nil {
		allowed, accessErr := r.requestCanAccessMusic(req, track)
		if accessErr != nil || !allowed {
			writeSubsonicError(w, req.FormValue("f"), 70, "Cover art not found")
			return
		}
		value = firstNonEmpty(track.CoverArtLargeURL, track.CoverArtURL, track.ImageURL)
	} else if album, albumErr := r.db.GetAlbumByID(id); albumErr == nil {
		tracks, trackErr := r.db.GetAlbumTracksByID(id)
		if trackErr == nil {
			tracks, trackErr = r.filterMusicForRequest(req, tracks)
		}
		if trackErr != nil || len(tracks) == 0 {
			writeSubsonicError(w, req.FormValue("f"), 70, "Cover art not found")
			return
		}
		value = firstNonEmpty(album.CoverArtLargeURL, album.CoverArtURL, album.CoverArtMediumURL, album.CoverArtSmallURL)
	} else if artist, artistErr := r.db.GetArtistByID(id); artistErr == nil {
		tracks, trackErr := r.db.GetArtistTracks(artist.Name)
		if trackErr == nil {
			tracks, trackErr = r.filterMusicForRequest(req, tracks)
		}
		if trackErr != nil || len(tracks) == 0 {
			writeSubsonicError(w, req.FormValue("f"), 70, "Cover art not found")
			return
		}
		value = firstNonEmpty(artist.ImageLargeURL, artist.ImageURL, artist.ImageMediumURL, artist.ImageSmallURL)
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		http.Redirect(w, req, value, http.StatusTemporaryRedirect)
		return
	}
	filename := filepath.Base(strings.TrimSpace(value))
	for _, directory := range utils.ArtworkSearchDirectories() {
		path := filepath.Join(directory, filename)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			http.ServeFile(w, req, path)
			return
		}
	}
	writeSubsonicError(w, req.FormValue("f"), 70, "Cover art not found")
}

func subsonicSongMap(track database.Music) map[string]interface{} {
	contentType := subsonicAudioContentType(track.Format, track.FilePath)
	result := map[string]interface{}{
		"id": track.ID, "title": track.Title,
		"album": track.Album, "artist": track.Artist, "artistId": track.ArtistID,
		"isDir": false, "isVideo": false, "type": "music", "coverArt": track.ID,
		"duration": track.Duration, "bitRate": 0, "track": track.TrackNumber,
		"year": track.Year, "genre": track.Genre, "size": track.FileSize,
		"suffix": strings.TrimPrefix(track.Format, "."), "contentType": contentType,
		"path": track.FileName, "playCount": track.PlayCount,
		"created": track.CreatedAt.Format(time.RFC3339),
	}
	if track.DiscNumber > 0 {
		result["discNumber"] = track.DiscNumber
	}
	result["replayGain"] = map[string]interface{}{
		"trackGain": track.ReplayGainTrackDB, "albumGain": track.ReplayGainAlbumDB,
		"trackPeak": track.ReplayGainTrackPeak, "albumPeak": track.ReplayGainAlbumPeak,
	}
	if strings.TrimSpace(track.Album) != "" {
		result["parent"] = albumIDForTrack(track)
	}
	return result
}

func subsonicAudioContentType(format, path string) string {
	return streaming.SourceContentType(format, path)
}

func subsonicAlbumMap(album database.Album) map[string]interface{} {
	return map[string]interface{}{
		"id": album.ID, "name": album.Name, "title": album.Name,
		"artist": album.Artist, "coverArt": album.ID, "songCount": album.TrackCount,
		"year": album.Year, "isDir": true,
	}
}

func subsonicPlaylistMap(playlist database.Playlist, owner string, tracks []database.Music) map[string]interface{} {
	duration := 0
	entries := make([]interface{}, 0, len(tracks))
	for _, track := range tracks {
		duration += track.Duration
		entries = append(entries, subsonicSongMap(track))
	}
	result := map[string]interface{}{
		"id": playlist.ID, "name": playlist.Name, "comment": playlist.Description,
		"owner": owner, "public": false, "songCount": len(playlist.TrackIDs),
		"duration": duration, "created": playlist.CreatedAt.Format(time.RFC3339Nano),
		"changed": playlist.UpdatedAt.Format(time.RFC3339Nano),
	}
	if playlist.Type == database.PlaylistTypeSmart {
		result["comment"] = strings.TrimSpace(playlist.Description + " [WaveNode smart playlist; read-only]")
	}
	if tracks != nil {
		result["entry"] = entries
	}
	return result
}

func playlistRevision(playlists []database.Playlist) int64 {
	var latest int64
	for _, playlist := range playlists {
		if revision := playlist.UpdatedAt.UnixMilli(); revision > latest {
			latest = revision
		}
	}
	return latest
}

func subsonicUser(user *database.User) map[string]interface{} {
	admin := user.Role == "admin"
	return map[string]interface{}{
		"username": user.Username, "email": user.Email, "scrobblingEnabled": true,
		"adminRole": admin, "settingsRole": true, "downloadRole": true,
		"uploadRole": admin, "playlistRole": true, "coverArtRole": true,
		"commentRole": false, "podcastRole": false, "streamRole": true,
		"jukeboxRole": false, "shareRole": false, "videoConversionRole": false,
	}
}

func subsonicSongs(tracks []database.Music) []interface{} {
	items := make([]interface{}, 0, len(tracks))
	for _, track := range tracks {
		items = append(items, subsonicSongMap(track))
	}
	return items
}

func subsonicResponseIncludesMedia(method string) bool {
	switch method {
	case "getArtists", "getIndexes", "getArtist", "getAlbum", "getSong", "getMusicDirectory",
		"getAlbumList", "getAlbumList2", "getRandomSongs", "getSongsByGenre", "getStarred",
		"getStarred2", "search", "search2", "search3", "getPlaylist", "getTopSongs",
		"getSimilarSongs", "getSimilarSongs2", "getBookmarks", "getPlayQueue":
		return true
	default:
		return false
	}
}

func applySubsonicRatings(value interface{}, ratings map[string]int, averageRatings map[string]float64) {
	switch typed := value.(type) {
	case map[string]interface{}:
		if id, ok := typed["id"].(string); ok {
			if rating := ratings[id]; rating > 0 {
				typed["userRating"] = rating
			}
			if rating := averageRatings[id]; rating > 0 {
				typed["averageRating"] = rating
			}
		}
		for _, child := range typed {
			applySubsonicRatings(child, ratings, averageRatings)
		}
	case []interface{}:
		for _, child := range typed {
			applySubsonicRatings(child, ratings, averageRatings)
		}
	}
}

func albumIDForTrack(track database.Music) string {
	hash := generateAlbumID(track.Album, track.Artist)
	return hash
}

func writeSubsonicResponse(w http.ResponseWriter, format string, data map[string]interface{}) {
	response := map[string]interface{}{
		"status": "ok", "version": subsonicAPIVersion, "type": "WaveNode",
		"serverVersion": WaveNodeVersion, "openSubsonic": true,
	}
	for key, value := range data {
		response[key] = value
	}
	writeSubsonicPayload(w, format, response)
}

func writeSubsonicError(w http.ResponseWriter, format string, code int, message string) {
	writeSubsonicPayload(w, format, map[string]interface{}{
		"status": "failed", "version": subsonicAPIVersion, "type": "WaveNode",
		"serverVersion": WaveNodeVersion, "openSubsonic": true,
		"error": map[string]interface{}{"code": code, "message": message},
	})
}

func writeSubsonicPayload(w http.ResponseWriter, format string, response map[string]interface{}) {
	if strings.EqualFold(format, "json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"subsonic-response": response})
		return
	}
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, _ = io.WriteString(w, xml.Header)
	encoder := xml.NewEncoder(w)
	root := xml.StartElement{Name: xml.Name{Local: "subsonic-response"}, Attr: []xml.Attr{
		{Name: xml.Name{Local: "xmlns"}, Value: "http://subsonic.org/restapi"},
	}}
	for _, key := range []string{"status", "version", "type", "serverVersion", "openSubsonic"} {
		if value, exists := response[key]; exists {
			root.Attr = append(root.Attr, xml.Attr{Name: xml.Name{Local: key}, Value: fmt.Sprint(value)})
			delete(response, key)
		}
	}
	_ = encoder.EncodeToken(root)
	keys := sortedMapKeys(response)
	for _, key := range keys {
		_ = encodeSubsonicXMLValue(encoder, key, response[key])
	}
	_ = encoder.EncodeToken(root.End())
	_ = encoder.Flush()
}

func encodeSubsonicXMLValue(encoder *xml.Encoder, name string, value interface{}) error {
	switch typed := value.(type) {
	case []interface{}:
		for _, item := range typed {
			if err := encodeSubsonicXMLValue(encoder, name, item); err != nil {
				return err
			}
		}
		return nil
	case map[string]interface{}:
		start := xml.StartElement{Name: xml.Name{Local: name}}
		childKeys := make([]string, 0)
		for _, key := range sortedMapKeys(typed) {
			switch typed[key].(type) {
			case map[string]interface{}, []interface{}:
				childKeys = append(childKeys, key)
			default:
				start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: key}, Value: fmt.Sprint(typed[key])})
			}
		}
		if err := encoder.EncodeToken(start); err != nil {
			return err
		}
		for _, key := range childKeys {
			if err := encodeSubsonicXMLValue(encoder, key, typed[key]); err != nil {
				return err
			}
		}
		return encoder.EncodeToken(start.End())
	default:
		return encoder.EncodeElement(fmt.Sprint(typed), xml.StartElement{Name: xml.Name{Local: name}})
	}
}

func sortedMapKeys(value map[string]interface{}) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func queryInt(req *http.Request, key string, fallback, minimum, maximum int) int {
	value, err := strconv.Atoi(req.FormValue(key))
	if err != nil {
		value = fallback
	}
	if value < minimum {
		value = minimum
	}
	if maximum > 0 && value > maximum {
		value = maximum
	}
	return value
}

func sliceAlbums(items []database.Album, offset, size int) []database.Album {
	if offset >= len(items) {
		return []database.Album{}
	}
	end := offset + size
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func sliceTracks(items []database.Music, offset, size int) []database.Music {
	if offset >= len(items) {
		return []database.Music{}
	}
	end := offset + size
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func limitInterfaces(items []interface{}, limit int) []interface{} {
	if limit < len(items) {
		return items[:limit]
	}
	return items
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func intValue(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		parsed, _ := strconv.Atoi(fmt.Sprint(value))
		return parsed
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func missingSubsonicParameter(name string) *subsonicError {
	return &subsonicError{Code: 10, Message: fmt.Sprintf("Required parameter '%s' is missing", name)}
}

func notFoundSubsonicError(name string) *subsonicError {
	return &subsonicError{Code: 70, Message: name + " not found"}
}

func internalSubsonicError(err error) *subsonicError {
	return &subsonicError{Code: 0, Message: err.Error()}
}
