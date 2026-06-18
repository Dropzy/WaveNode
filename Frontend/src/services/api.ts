import axios from 'axios'
import { notifyPlaylistsChanged } from '../utils/playlistEvents'

const configuredApiBaseUrl = (globalThis as { MUSIC_SERVER_API_BASE_URL?: string }).MUSIC_SERVER_API_BASE_URL

export const API_BASE_URL = configuredApiBaseUrl || import.meta.env.VITE_API_BASE_URL || '/api'

export const API_ORIGIN = new URL(API_BASE_URL, window.location.origin).origin

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

export interface Music {
  id: string
  title: string
  artist: string
  artist_id?: string
  artist_image_url?: string
  album: string
  genre: string
  duration: number
  track_number?: number
  disc_number?: number
  disc_total?: number
  replaygain_track_db?: number
  replaygain_album_db?: number
  release_date: string
  file_path: string
  image_url?: string
  cover_art_url?: string
  cover_art_small_url?: string
  cover_art_medium_url?: string
  cover_art_large_url?: string
  upload_order?: number
  created_at: string
  updated_at: string
  stream_url?: string
  is_external?: boolean
}

export interface PluginRowItem {
  id: string
  title: string
  subtitle?: string
  description?: string
  image_url?: string
  stream_url: string
  homepage_url?: string
}

export interface PluginHomeRow {
  plugin_id: string
  id: string
  title: string
  subtitle?: string
  type: 'radio'
  items: PluginRowItem[]
}

export interface PluginRecord {
  id: string
  name: string
  version: string
  enabled: boolean
  manifest: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface Playlist {
  id: string
  name: string
  description: string
  type?: 'manual' | 'smart'
  smart_rules?: SmartPlaylistRules
  track_ids: string[]
  created_at: string
  updated_at: string
}

export interface PluginRadioMetadata {
  station_title: string
  stream_title: string
  error?: string
}

export interface PluginTrackAction {
  plugin_id: string
  id: string
  label: string
  icon?: string
  action_type: 'download'
  url: string
}

export interface DiscoverySettings {
  listenbrainz_user: string
}

export interface DiscoveryRecommendation {
  title: string
  artist: string
  album: string
  source_playlist: string
  matched_track_id?: string
}

export interface DiscoveryPreview {
  source: string
  listenbrainz_url: string
  total: number
  matched: Music[]
  missing: DiscoveryRecommendation[]
  recommendations: DiscoveryRecommendation[]
}

export interface DiscoveryImportResult {
  playlist: Playlist
  preview: DiscoveryPreview
}

export interface SmartPlaylistCondition {
  field: 'title' | 'artist' | 'album' | 'genre' | 'year' | 'duration' | 'play_count' | 'rating' | 'date_added' | 'liked' | 'has_artwork'
  operator: string
  value: string
}

export interface SmartPlaylistRules {
  match: 'all' | 'any'
  conditions: SmartPlaylistCondition[]
  groups?: SmartPlaylistGroup[]
  sort_by: 'title' | 'artist' | 'album' | 'genre' | 'year' | 'duration' | 'date_added' | 'play_count' | 'rating' | 'random'
  sort_direction: 'asc' | 'desc'
  limit: number
}

export interface SmartPlaylistGroup {
  match: 'all' | 'any'
  conditions: SmartPlaylistCondition[]
  groups?: SmartPlaylistGroup[]
}

export interface PlaybackProfile {
  replaygain_mode: 'off' | 'track' | 'album'
  replaygain_preamp_db: number
  transcode_enabled: boolean
  transcode_format: 'mp3' | 'opus' | 'aac'
  transcode_bitrate: number
}

export interface ScrobbleSettings {
  listenbrainz_enabled: boolean
  has_listenbrainz_token: boolean
  listenbrainz_token?: string
  lastfm_enabled: boolean
  lastfm_server_configured: boolean
  lastfm_username?: string
  has_lastfm_session_key: boolean
  has_lastfm_pending_token: boolean
}

export interface LastFMIntegrationSettings {
  api_key: string
  shared_secret?: string
  has_shared_secret: boolean
  configured: boolean
}

export interface UserSession {
  id: string
  device_name: string
  user_agent: string
  ip_address: string
  last_seen_at: string
  created_at: string
  expires_at: string
  revoked_at?: string
}

export interface ListeningHistoryEntry {
  id: string
  played_at: string
  source: string
  device: string
  track: Music
}

export interface User {
  id: string
  username: string
  email: string
  role: 'admin' | 'user'
  created_at: string
  updated_at: string
}

export interface Album {
  id: string
  name: string
  artist: string
  year: number
  track_count: number
  tracks?: Music[]
  cover_art_url?: string
  cover_art_small_url?: string
  cover_art_medium_url?: string
  cover_art_large_url?: string
}

export interface AlbumInfo {
  name: string
  artist: string
  year: number
  cover_art_url?: string
  cover_art_small_url?: string
  cover_art_medium_url?: string
  cover_art_large_url?: string
}

export interface AlbumTracksResponse {
  album: AlbumInfo
  tracks: Music[]
}

export interface SimilarAlbum {
  name: string
  artist: string
  year: number
  track_count: number
}

export interface AlbumTracksFallbackResponse {
  success: boolean
  message: string
  similarAlbums?: SimilarAlbum[]
  album?: AlbumInfo
  tracks?: Music[]
}

export interface ArtistInfo {
  name: string
  track_count: number
  album_count: number
  id: string
  image_url?: string
  image_small_url?: string
  image_medium_url?: string
  image_large_url?: string
}

export interface Artist {
  id: string
  name: string
  track_count?: number
  album_count?: number
  spotify_id: string
  spotify_url: string
  image_url: string
  image_small_url: string
  image_medium_url: string
  image_large_url: string
  followers: number
  popularity: number
  genres: string[]
  biography: string
  country: string
  external_urls: Record<string, string>
  uri: string
  href: string
  type: string
  api_data: string
  last_enriched_at: string
  created_at: string
  updated_at: string
}

export interface ArtistImage {
  id: number
  artist_id: string
  source: string
  image_url: string
  thumbnail_url?: string
  source_page_url?: string
  license_name?: string
  license_url?: string
  author_name?: string
  attribution_text?: string
  width: number
  height: number
  mime_type?: string
  confidence_score: number
  is_primary: boolean
  created_at: string
  updated_at: string
}

export interface ArtistLookupResult {
  artist: {
    mbid: string
    name: string
    country?: string
    wikidata_id?: string
    confidence_score: number
  }
  image?: ArtistImage
  candidates: ArtistImage[]
  refreshed: boolean
}

export interface ArtistTracksResponse {
  artist: ArtistInfo
  tracks: Music[]
  albums: Album[]
}

export interface AuthResponse {
  token: string
  user: User
}

export interface SetupStatus {
  required: boolean
  token_required: boolean
  default_artwork_path: string
  registration_enabled: boolean
}

export interface DirectoryBrowserData {
  current_path: string
  parent_path: string
  directories: Array<{ name: string; path: string }>
  roots: string[]
}

export interface ScanStatus {
  id: string
  status: 'running' | 'stopping' | 'stopped' | 'completed' | 'failed'
  progress: number
  total_files: number
  processed: number
  current_file: string
  errors: string[]
  songs_added: number
  songs_updated: number
  tracks_skipped: number
}

export interface SetupResult extends AuthResponse {
  scan: ScanStatus | null
  scan_warning: string
}

export interface APIResponse<T> {
  success: boolean
  message: string
  data?: T
  error?: string
}

export interface SearchResult {
  songs: Music[]
  albums: Album[]
  artists: Array<{
    id?: string
    name: string
    track_count: number
    album_count: number
    image_url?: string
    image_small_url?: string
    image_medium_url?: string
    image_large_url?: string
  }>
  playlists: Playlist[]
}

export const pluginsAPI = {
  getHomeRows: async (): Promise<PluginHomeRow[]> => {
    const response = await api.get<APIResponse<PluginHomeRow[]>>('/plugins/home-rows')
    return response.data.data || []
  },

  getRadioMetadata: async (streamUrl: string): Promise<PluginRadioMetadata | null> => {
    const response = await api.get<APIResponse<PluginRadioMetadata>>('/plugins/radio-metadata', {
      params: { stream_url: streamUrl },
    })
    return response.data.data || null
  },

  getTrackActions: async (): Promise<PluginTrackAction[]> => {
    const response = await api.get<APIResponse<PluginTrackAction[]>>('/plugins/track-actions')
    return response.data.data || []
  },
}

export const discoveryAPI = {
  getSettings: async (): Promise<DiscoverySettings> => {
    const response = await api.get<APIResponse<DiscoverySettings>>('/discovery/settings')
    return response.data.data || { listenbrainz_user: '' }
  },

  saveSettings: async (settings: DiscoverySettings): Promise<DiscoverySettings> => {
    const response = await api.put<APIResponse<DiscoverySettings>>('/discovery/settings', settings)
    return response.data.data || settings
  },

  preview: async (source = 'weekly-exploration'): Promise<DiscoveryPreview> => {
    const response = await api.get<APIResponse<DiscoveryPreview>>('/discovery/preview', { params: { source } })
    if (!response.data.data) {
      throw new Error(response.data.error || 'Discovery preview was not returned')
    }
    return response.data.data
  },

  importPlaylist: async (source = 'weekly-exploration', playlistName?: string): Promise<DiscoveryImportResult> => {
    const response = await api.post<APIResponse<DiscoveryImportResult>>('/discovery/import', {
      source,
      playlist_name: playlistName,
    })
    if (!response.data.data) {
      throw new Error(response.data.error || 'Discovery playlist was not created')
    }
    notifyPlaylistsChanged()
    return response.data.data
  },
}

export const musicAPI = {
  getAllMusic: async (): Promise<Music[]> => {
    const response = await api.get<APIResponse<Music[]>>('/music')
    return response.data.data || []
  },

  downloadMusic: async (id: string, fallbackFilename = 'track'): Promise<void> => {
    const response = await api.get<Blob>(`/music/${id}/download`, {
      responseType: 'blob',
    })
    const contentDispositionHeader = response.headers['content-disposition']
    const contentDisposition = typeof contentDispositionHeader === 'string' ? contentDispositionHeader : ''
    const filenameMatch = contentDisposition.match(/filename\*=UTF-8''([^;]+)|filename="?([^"]+)"?/i)
    const filename = decodeURIComponent(filenameMatch?.[1] || filenameMatch?.[2] || fallbackFilename)
    const contentTypeHeader = response.headers['content-type']
    const blob = new Blob([response.data], {
      type: typeof contentTypeHeader === 'string' ? contentTypeHeader : 'application/octet-stream',
    })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
  },

  getMusic: async (id: string): Promise<Music | null> => {
    const response = await api.get<APIResponse<Music>>(`/music/${id}`)
    return response.data.data || null
  },

  searchMusic: async (query: string): Promise<Music[]> => {
    const response = await api.get<APIResponse<Music[]>>(`/music/search?q=${encodeURIComponent(query)}`)
    return response.data.data || []
  },

  comprehensiveSearch: async (query: string): Promise<SearchResult> => {
    const response = await api.get<APIResponse<SearchResult>>(`/search?q=${encodeURIComponent(query)}`)
    return response.data.data || { songs: [], albums: [], artists: [], playlists: [] }
  },

  addMusic: async (music: Omit<Music, 'id' | 'created_at' | 'updated_at'>): Promise<Music | null> => {
    const response = await api.post<APIResponse<Music>>('/music', music)
    return response.data.data || null
  },

  updateMusic: async (id: string, music: Partial<Music>): Promise<Music | null> => {
    const response = await api.put<APIResponse<Music>>(`/music/${id}`, music)
    return response.data.data || null
  },

  deleteMusic: async (id: string): Promise<boolean> => {
    const response = await api.delete<APIResponse<null>>(`/music/${id}`)
    return response.data.success
  },

  getCurrentScan: async (): Promise<APIResponse<ScanStatus> | null> => {
    try {
      const response = await api.get<APIResponse<ScanStatus>>('/scan/current')
      return response.data
    } catch (error) {
      console.error('Failed to fetch current scan:', error)
      return null
    }
  },
}

export const albumAPI = {
  getAllAlbums: async (): Promise<Album[]> => {
    const response = await api.get<APIResponse<Album[]>>('/albums')
    return response.data.data || []
  },

  getAlbumTracks: async (albumId: string): Promise<AlbumTracksResponse | AlbumTracksFallbackResponse | null> => {
    const cacheKey = `album_${albumId}`
    const now = Date.now()
    
    // Check if we recently failed to fetch this album
    const cachedFailure = failedRequestsCache.get(cacheKey)
    if (cachedFailure) {
      const timeSinceFailure = now - cachedFailure.timestamp
      
      // If we haven't exceeded cache duration and retry count, skip request
      if (timeSinceFailure < CACHE_DURATION && cachedFailure.retryCount >= MAX_RETRY_COUNT) {
        console.warn(`Skipping request for album "${albumId}" - recently failed (${cachedFailure.retryCount} attempts)`)
        return null
      }
      
      // If cache duration has passed, remove entry and try again
      if (timeSinceFailure >= CACHE_DURATION) {
        failedRequestsCache.delete(cacheKey)
      }
    }

    try {
      // albumId is now a hash, so we don't need to encode it
      const response = await api.get<APIResponse<AlbumTracksResponse>>(`/albums/${albumId}/tracks`)
      
      if (response.data.data) {
        // Clear any previous failure cache on success
        failedRequestsCache.delete(cacheKey)
        return response.data.data
      } else {
        // Try to get fallback response with similar albums
        try {
          const fallbackResponse = await api.get<AlbumTracksFallbackResponse>(`/albums/${albumId}/tracks`)
          if (fallbackResponse.data && fallbackResponse.data.success && fallbackResponse.data.similarAlbums) {
            failedRequestsCache.delete(cacheKey)
            return fallbackResponse.data
          }
        } catch (fallbackError) {
          console.warn('Fallback request failed:', fallbackError)
        }
        
        // Cache failure with retry count
        const currentRetryCount = (cachedFailure?.retryCount || 0) + 1
        failedRequestsCache.set(cacheKey, {
          timestamp: now,
          retryCount: currentRetryCount
        })
        
        console.warn(`Album "${albumId}" request failed (attempt ${currentRetryCount}/${MAX_RETRY_COUNT})`)
        
        // Only set timeout to clear cache if we haven't exceeded max retries
        if (currentRetryCount < MAX_RETRY_COUNT) {
          setTimeout(() => failedRequestsCache.delete(cacheKey), CACHE_DURATION)
        }
        
        return null
      }
    } catch (error) {
      // Try to get fallback response with similar albums
      try {
        const fallbackResponse = await api.get<AlbumTracksFallbackResponse>(`/albums/${albumId}/tracks`)
        if (fallbackResponse.data && fallbackResponse.data.success && fallbackResponse.data.similarAlbums) {
          failedRequestsCache.delete(cacheKey)
          return fallbackResponse.data
        }
      } catch (fallbackError) {
        console.warn('Fallback request failed:', fallbackError)
      }
      
      // Cache failure with retry count
      const currentRetryCount = (cachedFailure?.retryCount || 0) + 1
      failedRequestsCache.set(cacheKey, {
        timestamp: now,
        retryCount: currentRetryCount
      })
      
      console.warn(`Album "${albumId}" request failed (attempt ${currentRetryCount}/${MAX_RETRY_COUNT}):`, error)
      
      // Only set timeout to clear cache if we haven't exceeded max retries
      if (currentRetryCount < MAX_RETRY_COUNT) {
        setTimeout(() => failedRequestsCache.delete(cacheKey), CACHE_DURATION)
      }
      
      throw error
    }
  },
}

export const artistAPI = {
  getAllArtists: async (): Promise<Artist[]> => {
    const response = await api.get<APIResponse<Artist[]>>('/artists');
    return response.data.data || [];
  },

  getArtistTracks: async (artistName: string): Promise<ArtistTracksResponse | null> => {
    const response = await api.get<APIResponse<ArtistTracksResponse>>(`/artists/${encodeURIComponent(artistName)}/tracks`)
    return response.data.data || null
  },

  getArtistTracksById: async (artistId: string): Promise<ArtistTracksResponse | null> => {
    const response = await api.get<APIResponse<ArtistTracksResponse>>(`/artists/${encodeURIComponent(artistId)}/tracks`)
    if (response.data.success && response.data.data) {
      return response.data.data
    }
    return null
  },

  lookup: async (name: string): Promise<ArtistLookupResult | null> => {
    const response = await api.get<APIResponse<ArtistLookupResult>>('/artists/lookup', { params: { name } })
    return response.data.data || null
  },

  getImage: async (artistId: string): Promise<ArtistImage | null> => {
    const response = await api.get<APIResponse<ArtistImage | { image: null }>>(`/artists/${encodeURIComponent(artistId)}/image`)
    const data = response.data.data
    return data && 'image_url' in data ? data : null
  },
}

export const adminArtistImagesAPI = {
  refreshMetadata: async (artistId: string): Promise<ArtistLookupResult | null> => {
    const response = await api.post<APIResponse<ArtistLookupResult>>(`/admin/artists/${encodeURIComponent(artistId)}/refresh-metadata`)
    return response.data.data || null
  },

  listCandidates: async (artistId: string): Promise<ArtistImage[]> => {
    const response = await api.get<APIResponse<ArtistImage[]>>(`/admin/artists/${encodeURIComponent(artistId)}/image-candidates`)
    return response.data.data || []
  },

  setPrimary: async (artistId: string, imageId: number): Promise<ArtistImage | null> => {
    const response = await api.put<APIResponse<ArtistImage>>(`/admin/artists/${encodeURIComponent(artistId)}/image-primary`, { image_id: imageId })
    return response.data.data || null
  },
}

export const playlistAPI = {
  getAllPlaylists: async (): Promise<Playlist[]> => {
    const response = await api.get<APIResponse<Playlist[]>>('/playlists')
    return response.data.data || []
  },

  getPlaylist: async (id: string): Promise<Playlist | null> => {
    const response = await api.get<APIResponse<Playlist>>(`/playlists/${id}`)
    return response.data.data || null
  },

  createPlaylist: async (playlist: Omit<Playlist, 'id' | 'created_at' | 'updated_at'>): Promise<Playlist | null> => {
    const response = await api.post<APIResponse<Playlist>>('/playlists', playlist)
    if (response.data.data) notifyPlaylistsChanged()
    return response.data.data || null
  },

  updatePlaylist: async (id: string, playlist: Partial<Playlist>): Promise<Playlist | null> => {
    const response = await api.put<APIResponse<Playlist>>(`/playlists/${id}`, playlist)
    if (response.data.data) notifyPlaylistsChanged()
    return response.data.data || null
  },

  deletePlaylist: async (id: string): Promise<boolean> => {
    const response = await api.delete<APIResponse<null>>(`/playlists/${id}`)
    if (response.data.success) notifyPlaylistsChanged()
    return response.data.success
  },

  addTrackToPlaylist: async (playlistId: string, trackId: string): Promise<Playlist | null> => {
    const response = await api.post<APIResponse<Playlist>>(`/playlists/${playlistId}/tracks`, { track_id: trackId })
    if (response.data.data) notifyPlaylistsChanged()
    return response.data.data || null
  },

  addTracksToPlaylist: async (playlistId: string, trackIds: string[]): Promise<Playlist | null> => {
    const response = await api.post<APIResponse<Playlist>>(`/playlists/${playlistId}/tracks/bulk`, { track_ids: trackIds })
    if (response.data.data) notifyPlaylistsChanged()
    return response.data.data || null
  },

  removeTrackFromPlaylist: async (playlistId: string, trackId: string): Promise<Playlist | null> => {
    const response = await api.delete<APIResponse<Playlist>>(`/playlists/${playlistId}/tracks/${trackId}`)
    if (response.data.data) notifyPlaylistsChanged()
    return response.data.data || null
  },

  createSmartPlaylist: async (playlist: { name: string; description: string; smart_rules: SmartPlaylistRules }): Promise<Playlist | null> => {
    const response = await api.post<APIResponse<Playlist>>('/smart-playlists', playlist)
    if (response.data.data) notifyPlaylistsChanged()
    return response.data.data || null
  },

  updateSmartPlaylist: async (id: string, playlist: { name: string; description: string; smart_rules: SmartPlaylistRules }): Promise<Playlist | null> => {
    const response = await api.put<APIResponse<Playlist>>(`/smart-playlists/${id}`, playlist)
    if (response.data.data) notifyPlaylistsChanged()
    return response.data.data || null
  },

  previewSmartPlaylist: async (rules: SmartPlaylistRules): Promise<Music[]> => {
    const response = await api.post<APIResponse<Music[]>>('/smart-playlists/preview', rules)
    return response.data.data || []
  },

  importM3U: async (file: File, name?: string): Promise<Playlist | null> => {
    const form = new FormData()
    form.append('playlist', file)
    if (name) form.append('name', name)
    const response = await api.post<APIResponse<Playlist>>('/playlists/import', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    if (response.data.data) notifyPlaylistsChanged()
    return response.data.data || null
  },

  exportM3U: async (id: string, name: string): Promise<void> => {
    const response = await api.get(`/playlists/${id}/export.m3u`, { responseType: 'blob' })
    const href = URL.createObjectURL(response.data)
    const link = document.createElement('a')
    link.href = href
    link.download = `${name}.m3u8`
    link.click()
    URL.revokeObjectURL(href)
  },
}

export const accountAPI = {
  getPlaybackProfile: async (): Promise<PlaybackProfile> => {
    const response = await api.get<APIResponse<PlaybackProfile>>('/playback-profile')
    return response.data.data!
  },
  savePlaybackProfile: async (profile: PlaybackProfile): Promise<PlaybackProfile> => {
    const response = await api.put<APIResponse<PlaybackProfile>>('/playback-profile', profile)
    return response.data.data!
  },
  getSessions: async (): Promise<{ sessions: UserSession[]; current_session_id: string }> => {
    const response = await api.get<APIResponse<{ sessions: UserSession[]; current_session_id: string }>>('/auth/sessions')
    return response.data.data || { sessions: [], current_session_id: '' }
  },
  revokeSession: async (id: string) => api.delete(`/auth/sessions/${id}`),
  revokeOtherSessions: async () => api.delete('/auth/sessions/others'),
  getScrobbleSettings: async (): Promise<ScrobbleSettings> => {
    const response = await api.get<APIResponse<ScrobbleSettings>>('/scrobble/settings')
    return response.data.data || {
      listenbrainz_enabled: false,
      has_listenbrainz_token: false,
      lastfm_enabled: false,
      lastfm_server_configured: false,
      has_lastfm_session_key: false,
      has_lastfm_pending_token: false,
    }
  },
  saveScrobbleSettings: async (settings: ScrobbleSettings): Promise<ScrobbleSettings> => {
    const response = await api.put<APIResponse<ScrobbleSettings>>('/scrobble/settings', settings)
    return response.data.data || settings
  },
  startLastFMAuth: async (): Promise<{ auth_url: string }> => {
    const response = await api.post<APIResponse<{ auth_url: string }>>('/scrobble/lastfm/start')
    if (!response.data.data) {
      throw new Error(response.data.error || 'Last.fm connection could not be started')
    }
    return response.data.data
  },
  completeLastFMAuth: async (token?: string): Promise<ScrobbleSettings> => {
    const response = await api.post<APIResponse<ScrobbleSettings>>('/scrobble/lastfm/complete', token ? { token } : {})
    if (!response.data.data) {
      throw new Error(response.data.error || 'Last.fm connection could not be completed')
    }
    return response.data.data
  },
  disconnectLastFM: async (): Promise<ScrobbleSettings> => {
    const response = await api.delete<APIResponse<ScrobbleSettings>>('/scrobble/lastfm')
    if (!response.data.data) {
      throw new Error(response.data.error || 'Last.fm connection could not be disconnected')
    }
    return response.data.data
  },
}

export const adminIntegrationsAPI = {
  getLastFM: async (): Promise<LastFMIntegrationSettings> => {
    const response = await api.get<APIResponse<LastFMIntegrationSettings>>('/admin/integrations/lastfm')
    return response.data.data || { api_key: '', has_shared_secret: false, configured: false }
  },
  saveLastFM: async (settings: LastFMIntegrationSettings): Promise<LastFMIntegrationSettings> => {
    const response = await api.put<APIResponse<LastFMIntegrationSettings>>('/admin/integrations/lastfm', settings)
    return response.data.data || settings
  },
}

export const scrobbleAPI = {
  nowPlaying: async (trackId: string): Promise<void> => {
    await api.post(`/scrobble/now-playing/${encodeURIComponent(trackId)}`)
  },
  listened: async (trackId: string, listenedAt?: number): Promise<void> => {
    await api.post(`/scrobble/listened/${encodeURIComponent(trackId)}`, listenedAt ? { listened_at: listenedAt } : {})
  },
}

export const historyAPI = {
  get: async (search = ''): Promise<ListeningHistoryEntry[]> => {
    const response = await api.get<APIResponse<ListeningHistoryEntry[]>>('/history', { params: { search, limit: 500 } })
    return response.data.data || []
  },
  clear: async () => api.delete('/history'),
  exportCSV: async (): Promise<void> => {
    const response = await api.get('/history/export', { responseType: 'blob' })
    const href = URL.createObjectURL(response.data)
    const link = document.createElement('a')
    link.href = href
    link.download = 'wavenode-listening-history.csv'
    link.click()
    URL.revokeObjectURL(href)
  },
}

export const authAPI = {
  login: async (username: string, password: string) => {
    const response = await api.post<APIResponse<AuthResponse>>('/auth/login', { username, password })
    return response.data
  },

  register: async (username: string, email: string, password: string) => {
    const response = await api.post<APIResponse<AuthResponse>>('/auth/register', { username, email, password })
    return response.data
  },
}

export const ratingsAPI = {
  getRating: async (id: string): Promise<number> => {
    const response = await api.get<APIResponse<{ rating: number }>>(`/ratings/${id}`)
    return response.data.data?.rating || 0
  },

  setRating: async (id: string, rating: number): Promise<number> => {
    const response = await api.put<APIResponse<{ rating: number }>>(`/ratings/${id}`, { rating })
    return response.data.data?.rating || 0
  },
}

export const setupAPI = {
  getStatus: async (): Promise<SetupStatus> => {
    const response = await api.get<APIResponse<SetupStatus>>('/setup/status')
    if (!response.data.data) {
      throw new Error('Setup status was not returned')
    }
    return response.data.data
  },

  browseDirectories: async (path?: string, setupToken?: string): Promise<DirectoryBrowserData> => {
    const response = await api.get<APIResponse<DirectoryBrowserData>>('/setup/directories', {
      params: path ? { path } : undefined,
      headers: setupToken ? { 'X-WaveNode-Setup-Token': setupToken } : undefined,
    })
    if (!response.data.data) {
      throw new Error('Directory listing was not returned')
    }
    return response.data.data
  },

  complete: async (details: {
    username: string
    email: string
    password: string
    music_paths: string[]
    artwork_path: string
  }, setupToken?: string): Promise<SetupResult> => {
    const response = await api.post<APIResponse<SetupResult>>('/setup/complete', details, {
      headers: setupToken ? { 'X-WaveNode-Setup-Token': setupToken } : undefined,
    })
    if (!response.data.data) {
      throw new Error('Setup result was not returned')
    }
    return response.data.data
  },
}

export const healthAPI = {
  checkHealth: async () => {
    const response = await api.get<APIResponse<{ status: string; timestamp: string; version: string }>>('/health')
    return response.data
  },
}

export const userAPI = {
  getCurrentUser: async (): Promise<User | null> => {
    const response = await api.get<APIResponse<User>>('/auth/me')
    return response.data.data || null
  },
}

export const likedTracksAPI = {
  getLikedTracks: async (): Promise<Music[]> => {
    const response = await api.get<APIResponse<Music[]>>('/liked-tracks')
    return response.data.data || []
  },

  likeTrack: async (trackId: string): Promise<boolean> => {
    const response = await api.post<APIResponse<null>>(`/liked-tracks/${trackId}`)
    return response.data.success
  },

  unlikeTrack: async (trackId: string): Promise<boolean> => {
    const response = await api.delete<APIResponse<null>>(`/liked-tracks/${trackId}`)
    return response.data.success
  },

  isTrackLiked: async (trackId: string): Promise<boolean> => {
    const response = await api.get<APIResponse<{ is_liked: boolean }>>(`/liked-tracks/${trackId}/check`)
    return response.data.data?.is_liked || false
  },
}

export const recentlyPlayedAPI = {
  getRecentlyPlayed: async (): Promise<Music[]> => {
    const response = await api.get<APIResponse<Music[]>>('/recently-played')
    return response.data.data || []
  },

  addRecentlyPlayed: async (trackId: string): Promise<boolean> => {
    const response = await api.post<APIResponse<null>>(`/recently-played/${trackId}`)
    return response.data.success
  },
}

// Token validation and refresh utilities
export const tokenUtils = {
  isTokenExpired: (token: string): boolean => {
    try {
      const payload = JSON.parse(atob(token.split('.')[1]))
      const currentTime = Math.floor(Date.now() / 1000)
      return payload.exp < currentTime
    } catch (error) {
      console.error('Error parsing token:', error)
      return true // Assume expired if we can't parse it
    }
  },

  getToken: (): string | null => {
    return localStorage.getItem('token')
  },

  getValidToken: async (): Promise<string | null> => {
    const token = localStorage.getItem('token')
    if (!token) {
      return null
    }

    // Check if token is expired
    if (tokenUtils.isTokenExpired(token)) {
      console.warn('Token is expired, removing it...')
      localStorage.removeItem('token')
      delete api.defaults.headers.common['Authorization']
      return null
    }

    // Set token in headers for this session
    api.defaults.headers.common['Authorization'] = `Bearer ${token}`
    return token
  },

  setToken: (token: string) => {
    localStorage.setItem('token', token)
    api.defaults.headers.common['Authorization'] = `Bearer ${token}`
  },

  clearToken: () => {
    localStorage.removeItem('token')
    delete api.defaults.headers.common['Authorization']
  }
}

// Simple cache to prevent repeated failed requests with retry logic
const failedRequestsCache = new Map<string, { timestamp: number; retryCount: number }>()
const CACHE_DURATION = 2 * 60 * 1000 // 2 minutes (reduced from 5 minutes)
const MAX_RETRY_COUNT = 3 // Maximum number of retries before giving up
