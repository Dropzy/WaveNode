import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import styled from 'styled-components'
import { useNavigate } from 'react-router-dom'
import { Play, Pause, Disc, Plus, ListMusic, Edit, Trash2, User, Music2, MoreVertical, Download, PlusCircle, Heart, X, Search, ArrowUpDown, Check, CheckCircle2, List, Rows3, Sparkles, Upload, Podcast, ExternalLink, ArrowLeft, ChevronRight, Share2, ListPlus, BookOpen, Radio, type LucideIcon } from 'lucide-react'
import { albumAPI, musicAPI, playlistAPI, artistAPI, likedTracksAPI, pluginsAPI, podcastsAPI, type Music as APIMusic, type PluginTrackAction, type PodcastEpisode, type PodcastHomeResponse, type PodcastProgress, type PodcastSearchResult } from '../services/api'
import { useAuth } from '../contexts/AuthContext'
import { useAudio } from '../contexts/AudioContext'
import { getAlbumArtworkUrl, getArtworkGradient, getTrackArtworkUrl } from '../utils/mediaUrl'
import { playlistsChangedEvent } from '../utils/playlistEvents'
import { AddToPlaylistDialog } from '../components/TrackActionsMenu'
import { useTrackSelection } from '../hooks/useTrackSelection'
import { downloadPodcastEpisode, getPodcastDownloadUrl, hasPodcastDownload } from '../utils/podcastDownloads'
import { AudiobooksPanel } from '../components/AudiobooksPanel'
import RadioPanel from '../components/RadioPanel'

// Define types
interface Track {
  id: string
  title: string
  artist: string
  artist_id?: string
  album: string
  duration: number
  release_date: string
  genre: string
  file_path: string
  image_url?: string
  cover_art_url?: string
  cover_art_small_url?: string
  cover_art_medium_url?: string
  cover_art_large_url?: string
  upload_order?: number
  created_at: string
  updated_at: string
}

interface Playlist {
  id: string
  name: string
  description?: string
  type?: 'manual' | 'smart'
  track_ids: string[]
  created_at: string
  updated_at: string
}

interface Album {
  id: string
  name: string
  artist: string
  year: number
  track_count: number
  tracks?: Track[]
  cover_art_url?: string
  cover_art_small_url?: string
  cover_art_medium_url?: string
  cover_art_large_url?: string
  coverArt?: string  // Add camelCase version from API
}

type Music = Track

interface Artist {
  id?: string;
  name?: string;
  image_medium_url?: string;
  image_url?: string;
  image_small_url?: string;
  image_large_url?: string;
  track_count?: number;
  album_count?: number;
}

type TrackSort =
  | 'uploaded-desc'
  | 'title-asc'
  | 'artist-asc'
  | 'album-asc'
  | 'duration-asc'

type TrackView = 'compact' | 'list'
type LibraryTab = 'playlists' | 'albums' | 'artists' | 'tracks' | 'downloads' | 'podcasts' | 'audiobooks' | 'radio'

const libraryTabStorageKey = 'wavenode.library.activeTab'
const libraryDataStorageKey = 'wavenode.library.cache'
const libraryTabs: LibraryTab[] = ['playlists', 'albums', 'artists', 'tracks', 'downloads', 'podcasts', 'audiobooks', 'radio']

type CachedLibraryData = {
  music: Music[]
  playlists: Playlist[]
  albums: Album[]
  artists: Artist[]
  cachedAt: number
}

const getStoredLibraryTab = (): LibraryTab => {
  if (typeof window === 'undefined') {
    return 'playlists'
  }

  const storedTab = window.localStorage.getItem(libraryTabStorageKey)
  return libraryTabs.includes(storedTab as LibraryTab) ? storedTab as LibraryTab : 'playlists'
}

const readCachedLibraryData = (): CachedLibraryData | null => {
  if (typeof window === 'undefined') {
    return null
  }

  try {
    const rawCache = window.localStorage.getItem(libraryDataStorageKey)
    if (!rawCache) {
      return null
    }
    const parsed = JSON.parse(rawCache) as Partial<CachedLibraryData>
    if (!Array.isArray(parsed.music) || !Array.isArray(parsed.playlists) || !Array.isArray(parsed.albums) || !Array.isArray(parsed.artists)) {
      return null
    }
    return {
      music: parsed.music,
      playlists: parsed.playlists,
      albums: parsed.albums,
      artists: parsed.artists,
      cachedAt: typeof parsed.cachedAt === 'number' ? parsed.cachedAt : 0,
    }
  } catch {
    return null
  }
}

const writeCachedLibraryData = (cache: Omit<CachedLibraryData, 'cachedAt'>) => {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(libraryDataStorageKey, JSON.stringify({
      ...cache,
      cachedAt: Date.now(),
    }))
  } catch (error) {
    console.warn('Library cache could not be saved:', error)
  }
}

const fallbackDownloadFilename = (track: Track) => {
  const filePathName = track.file_path?.split(/[\\/]/).pop()
  return filePathName || `${track.artist} - ${track.title}`
}

const downloadLibraryTrack = async (track: Track, action: PluginTrackAction) => {
  if (action.action_type !== 'download') {
    return
  }

  await musicAPI.downloadMusic(track.id, fallbackDownloadFilename(track))
}

const LibraryContainer = styled.div`
  padding: 28px clamp(16px, 2vw, 32px) 40px;
  overflow-y: auto;
  
  @media (max-width: 768px) {
    padding: 16px;
    padding-top: 80px; // Account for mobile menu button
  }
`

const Header = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  
  @media (max-width:768px) {
    flex-direction: column;
    gap: 16px;
    align-items: stretch;
    margin-bottom: 20px;
  }
`

const Title = styled.h1`
  color: #fff;
  font-size: 32px;
  font-weight: 700;
  
  @media (max-width: 768px) {
    font-size: 24px;
  }
`

const SearchContainer = styled.div`
  position: relative;
  width: 300px;
  
  @media (max-width: 768px) {
    width: 100%;
  }
`

const SearchInput = styled.input`
  width: 100%;
  padding: 12px 16px 12px 44px;
  background-color: #242424;
  border: none;
  border-radius: 24px;
  color: #fff;
  font-size: 14px;

  &::placeholder {
    color: #b3b3b3;
  }

  &:focus {
    outline: none;
    background-color: #2a2a2a;
  }
  
  @media (max-width: 768px) {
    padding: 10px 14px 10px 40px;
    font-size: 16px;
  }
`

const SearchIcon = styled.div`
  position: absolute;
  left: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: #b3b3b3;
`

const MusicGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(175px, 1fr));
  gap: clamp(14px, 1.5vw, 24px);
  
  @media (max-width: 768px) {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 16px;
  }
  
  @media (max-width: 480px) {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
`

const PodcastGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;

  @media (max-width: 640px) {
    grid-template-columns: 1fr;
  }
`

const PodcastHomeSection = styled.section`
  margin-bottom: 32px;
`

const PodcastRow = styled.div`
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(260px, 340px);
  gap: 16px;
  overflow-x: auto;
  padding: 2px 2px 10px;
  scrollbar-width: thin;
  scrollbar-color: #4a4a4a transparent;

  &::-webkit-scrollbar {
    height: 6px;
  }

  &::-webkit-scrollbar-track {
    background: transparent;
  }

  &::-webkit-scrollbar-thumb {
    background: #4a4a4a;
    border-radius: 3px;
  }

  @media (max-width: 640px) {
    grid-auto-columns: minmax(250px, 86vw);
  }
`

const PodcastCard = styled.button`
  display: grid;
  grid-template-columns: 84px 1fr;
  gap: 14px;
  min-height: 128px;
  padding: 14px;
  border-radius: 8px;
  background: #181818;
  color: inherit;
  border: 0;
  width: 100%;
  text-align: left;
  font: inherit;
  cursor: pointer;
  transition: background 0.2s ease, transform 0.2s ease;

  &:hover,
  &:focus-visible {
    background: #282828;
    transform: translateY(-1px);
    outline: none;
  }
`

const PodcastArt = styled.div<{ $imageUrl?: string }>`
  width: 84px;
  height: 84px;
  border-radius: 8px;
  background: ${props => props.$imageUrl ? `url(${props.$imageUrl}) center/cover` : props.theme.colors.accentGradient};
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
`

const PodcastInfo = styled.div`
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
`

const PodcastTitle = styled.h3`
  color: #fff;
  font-size: 15px;
  font-weight: 700;
  line-height: 1.25;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
`

const PodcastPublisher = styled.p`
  color: #b3b3b3;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`

const PodcastDescription = styled.p`
  color: #d6d6d6;
  font-size: 12px;
  line-height: 1.45;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
`

const PodcastMeta = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  color: #b3b3b3;
  font-size: 12px;
  margin-top: auto;
`

const PodcastShowHeader = styled.div`
  display: grid;
  grid-template-columns: 156px minmax(0, 1fr);
  gap: 24px;
  margin-bottom: 28px;

  @media (max-width: 640px) {
    grid-template-columns: 96px minmax(0, 1fr);
    gap: 16px;
  }
`

const PodcastBackRow = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
`

const PodcastShowArt = styled(PodcastArt)`
  width: 156px;
  height: 156px;

  @media (max-width: 640px) {
    width: 96px;
    height: 96px;
  }
`

const PodcastShowInfo = styled.div`
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
`

const PodcastShowTitle = styled.h2`
  margin: 0;
  color: #fff;
  font-size: 24px;
  line-height: 1.2;
  letter-spacing: 0;

  @media (max-width: 640px) {
    font-size: 18px;
  }
`

const PodcastShowDescription = styled.p`
  margin: 0;
  color: #b3b3b3;
  font-size: 13px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
`

const PodcastHeaderActions = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: auto;
`

const PodcastIconButton = styled.button`
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: #2a2a2a;
  color: #fff;
  text-decoration: none;
  cursor: pointer;

  &:hover,
  &:focus-visible {
    background: #3a3a3a;
    outline: none;
  }

`

const PodcastEpisodeList = styled.div`
  display: flex;
  flex-direction: column;
`

const PodcastEpisodeRow = styled.div`
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr) auto;
  align-items: center;
  gap: 14px;
  min-height: 82px;
  padding: 12px 8px;
  border-bottom: 1px solid #282828;

  &:hover {
    background: #181818;
  }

  @media (max-width: 640px) {
    grid-template-columns: 40px minmax(0, 1fr);
  }
`

const PodcastEpisodeBody = styled.div`
  min-width: 0;
`

const PodcastEpisodeActions = styled.div`
	display: flex;
	align-items: center;
	gap: 4px;
`

const PodcastEpisodeTitle = styled.h3`
  margin: 0 0 5px;
  color: #fff;
  font-size: 14px;
  line-height: 1.35;
  letter-spacing: 0;
`

const PodcastEpisodeDescription = styled.p`
  margin: 0;
  color: #b3b3b3;
  font-size: 12px;
  line-height: 1.4;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
`

const PodcastEpisodeMeta = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  color: #8f8f8f;
  font-size: 11px;
`

const PodcastProgressTrack = styled.div`
  width: 100%;
  height: 3px;
  margin-top: 8px;
  overflow: hidden;
  background: #3a3a3a;
  border-radius: 2px;
`

const PodcastProgressFill = styled.div<{ $progress: number; $completed?: boolean }>`
  width: ${props => `${Math.max(0, Math.min(100, props.$progress))}%`};
  height: 100%;
  background: ${props => props.$completed ? '#45c46f' : props.theme.colors.accent};
`

const PodcastCompleted = styled.span`
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: #45c46f;
  font-weight: 600;
`

const PodcastEpisodeDuration = styled.span`
  color: #b3b3b3;
  font-size: 12px;
  white-space: nowrap;

  @media (max-width: 640px) {
    display: none;
  }
`

const MusicInfo = styled.div`
  margin-bottom: 8px;
  
  @media (max-width: 768px) {
    margin-bottom: 6px;
  }
`

const MusicTitle = styled.h3`
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 13px;
    margin-bottom: 3px;
  }
`

const MusicArtist = styled.p`
  color: #b3b3b3;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 11px;
  }
`

const MusicMeta = styled.div`
  color: #b3b3b3;
  font-size: 11px;
  display: flex;
  gap: 8px;
  
  @media (max-width: 768px) {
    font-size: 10px;
    gap: 6px;
    flex-wrap: wrap;
  }
`

const PlayButton = styled.button`
  position: absolute;
  bottom: 8px;
  right: 8px;
  width: 48px;
  height: 48px;
  background: ${({ theme }) => theme.colors.accentGradient};
  border: none;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.3);
  opacity: 0;
  z-index: 5;

  &:hover {
    background: ${({ theme }) => theme.colors.accentGradient};
    transform: scale(1.05);
  }
  
  @media (max-width: 768px) {
    width: 40px;
    height: 40px;
    bottom: 6px;
    right: 6px;
  }
`

const LoadingMessage = styled.div`
  color: #b3b3b3;
  text-align: center;
  padding: 40px;
  font-size: 16px;
`

const ErrorMessage = styled.div`
  color: #ff6b6b;
  text-align: center;
  padding: 40px;
  font-size: 16px;
`

const TabsContainer = styled.div`
  margin-bottom: 24px;
`

const TabsList = styled.div`
  display: flex;
  gap: 8px;
  border-bottom: 1px solid #282828;
  margin-bottom: 24px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  
  @media (max-width: 768px) {
    gap: 4px;
    margin-bottom: 16px;
  }
`

const TabButton = styled.button<{ $active: boolean }>`
  background: none;
  border: none;
  color: ${props => props.$active ? '#fff' : '#b3b3b3'};
  padding: 12px 24px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  border-bottom: 2px solid ${props => props.$active ? props.theme.colors.accent : 'transparent'};
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
  flex-shrink: 0;

  &:hover {
    color: #fff;
  }
  
  @media (max-width: 768px) {
    padding: 10px 16px;
    font-size: 13px;
    gap: 6px;
  }
`

const TabContent = styled.div`
  min-height: 400px;
  
  @media (max-width: 768px) {
    min-height: 300px;
  }
`

const SectionHeader = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  
  @media (max-width: 768px) {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
    margin-bottom: 16px;
  }
`

const SectionTitle = styled.h2`
  color: #fff;
  font-size: 24px;
  font-weight: 700;
  
  @media (max-width: 768px) {
    font-size: 20px;
  }
`

const TrackHeaderActions = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
  position: relative;

  @media (max-width: 768px) {
    width: 100%;
    justify-content: space-between;
  }
`

const SortButton = styled.button<{ $open?: boolean }>`
  height: 38px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border: 1px solid ${props => props.$open ? props.theme.colors.accent : props.theme.colors.borderStrong};
  border-radius: 20px;
  color: ${props => props.$open ? '#fff' : '#b3b3b3'};
  background: #242424;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;

  &:hover,
  &:focus-visible {
    color: #fff;
    border-color: #777;
    outline: none;
  }

  @media (max-width: 480px) {
    max-width: 190px;
  }
`

const SortMenu = styled.div`
  position: absolute;
  top: calc(100% + 8px);
  right: 82px;
  width: 230px;
  padding: 8px;
  border: 1px solid #3a3a3a;
  border-radius: 8px;
  background: #282828;
  box-shadow: 0 14px 38px rgba(0, 0, 0, 0.5);
  z-index: 100;

  @media (max-width: 768px) {
    right: auto;
    left: 0;
  }
`

const SortMenuLabel = styled.div`
  padding: 8px 10px 5px;
  color: #b3b3b3;
  font-size: 12px;
  font-weight: 700;
`

const SortMenuItem = styled.button<{ $active?: boolean }>`
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px;
  border: 0;
  border-radius: 5px;
  color: ${props => props.$active ? props.theme.colors.accent : props.theme.colors.text};
  background: transparent;
  font-size: 14px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;

  &:hover,
  &:focus-visible {
    background: #3a3a3a;
    outline: none;
  }
`

const SortMenuItemText = styled.span`
  display: flex;
  align-items: center;
  gap: 10px;
`

const SortMenuDivider = styled.div`
  height: 1px;
  margin: 7px 6px;
  background: #404040;
`

const SortButtonLabel = styled.span`
  max-width: 155px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`

const TrackList = styled.div<{ $compact?: boolean }>`
  display: flex;
  flex-direction: column;
  gap: ${props => props.$compact ? '2px' : '8px'};

  @media (max-width: 768px) {
    gap: ${props => props.$compact ? '2px' : '6px'};
  }
`

const TrackItem = styled.div<{ $selected?: boolean; $compact?: boolean }>`
  background-color: ${props => props.$selected ? '#3a3a3a' : props.$compact ? 'transparent' : '#181818'};
  border-radius: ${props => props.$compact ? '4px' : '8px'};
  padding: ${props => props.$compact ? '7px 12px' : '12px 16px'};
  min-height: ${props => props.$compact ? '44px' : '64px'};
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;

  &:hover {
    background-color: #282828;
  }

  @media (max-width: 768px) {
    padding: ${props => props.$compact ? '7px 10px' : '10px 12px'};
    gap: 10px;
  }
`

const PlayAllButton = styled.button`
  background: ${({ theme }) => theme.colors.accentGradient};
  border: none;
  border-radius: 20px;
  color: ${({ theme }) => theme.colors.accentText};
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s ease;

  &:hover {
    background: ${({ theme }) => theme.colors.accentGradient};
    transform: scale(1.05);
  }
  
  @media (max-width: 768px) {
    padding: 10px 16px;
    font-size: 13px;
    justify-content: center;
  }
`

const AddButton = styled.button`
  background: ${({ theme }) => theme.colors.accentGradient};
  border: none;
  border-radius: 20px;
  color: ${({ theme }) => theme.colors.accentText};
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s ease;

  &:hover {
    background: ${({ theme }) => theme.colors.accentGradient};
    transform: scale(1.05);
  }
  
  @media (max-width: 768px) {
    padding: 10px 16px;
    font-size: 13px;
    justify-content: center;
  }
`

const PlaylistCard = styled.div`
  background-color: #181818;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;

  &:hover {
    background-color: #282828;
    transform: translateY(-2px);
  }

  &:hover ${PlayButton} {
    opacity: 1;
  }
`

const PlaylistArt = styled.div`
  width: 100%;
  aspect-ratio: 1;
  background: linear-gradient(135deg, #ff6b6b, #ff8e8e);
  border-radius: 8px;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 48px;
  position: relative;
`

const AlbumCard = styled.div`
  background-color: #181818;
  border: 1px solid transparent;
  border-radius: 10px;
  padding: 12px;
  cursor: pointer;
  transition: background-color 0.2s ease, border-color 0.2s ease, transform 0.2s ease;
  position: relative;
  min-width: 0;

  &:hover {
    background-color: #242424;
    border-color: #333;
    transform: translateY(-2px);
  }

  &:hover ${PlayButton} {
    opacity: 1;
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.colors.accent};
    outline-offset: 2px;
  }

  @media (hover: none) {
    ${PlayButton} {
      opacity: 1;
    }
  }
`

const AlbumArt = styled.div<{ $imageUrl?: string; $fallback: string }>`
  width: 100%;
  aspect-ratio: 1;
  background: ${props => props.$imageUrl ? `url("${props.$imageUrl}")` : props.$fallback};
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  border-radius: 7px;
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.78);
  font-size: 48px;
  position: relative;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.24);

  &::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(to top, rgba(0, 0, 0, 0.22), transparent 45%);
    pointer-events: none;
  }
`

const ArtistCard = styled.div`
  background-color: #181818;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;

  &:hover {
    background-color: #282828;
    transform: translateY(-2px);
  }

  &:hover ${PlayButton} {
    opacity: 1;
  }
`

const ArtistArt = styled.div<{ $imageUrl?: string; $fallback: string }>`
  width: 100%;
  aspect-ratio: 1;
  background: ${props => props.$imageUrl ? `url("${props.$imageUrl}")` : props.$fallback};
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  border-radius: 8px;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 48px;
  position: relative;
`

const TrackNumberContainer = styled.div`
  width: 40px;
  display: flex;
  justify-content: center;
  align-items: center;
  position: relative;
  
  @media (max-width: 768px) {
    width: 35px;
  }
`

const TrackNumber = styled.span<{ $hidden?: boolean }>`
  color: #b3b3b3;
  font-size: 14px;
  text-align: center;
  opacity: ${props => props.$hidden ? 0 : 1};
  transition: opacity 0.2s ease;
  
  @media (max-width: 768px) {
    font-size: 12px;
  }
`

const PlayIcon = styled.button<{ $visible?: boolean }>`
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: ${({ theme }) => theme.colors.accent};
  opacity: ${props => props.$visible ? 1 : 0};
  transition: opacity 0.2s ease;
  cursor: pointer;
  padding: 4px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  
  &:hover {
    color: ${({ theme }) => theme.colors.accentHover};
    background: rgba(255, 255, 255, 0.08);
  }
`

const TrackCoverArt = styled.div<{ $imageUrl?: string }>`
  width: 40px;
  height: 40px;
  flex: 0 0 40px;
  background-image: ${props => props.$imageUrl ? `url("${props.$imageUrl}")` : 'linear-gradient(135deg, #4a90e2, #7bb3f0)'};
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 16px;

  @media (max-width: 768px) {
    width: 35px;
    height: 35px;
    flex-basis: 35px;
    font-size: 14px;
  }
`

const TrackAlbum = styled.div`
  color: #b3b3b3;
  font-size: 14px;
  flex: 1 1 180px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    display: none; /* Hide album column on mobile */
  }
`

const TrackDateAdded = styled.div`
  color: #b3b3b3;
  font-size: 14px;
  flex: 1 1 100px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    display: none; /* Hide date added column on mobile */
  }
`

const ContextMenuButton = styled.button<{ $visible?: boolean }>`
  background: none;
  border: none;
  color: #b3b3b3;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  opacity: ${props => props.$visible ? 1 : 0};
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);

  &:hover {
    background-color: #383838;
    color: #fff;
  }
  
  @media (max-width: 768px) {
    right: 12px;
    opacity: 1;
  }
`

const ContextMenu = styled.div<{ $visible: boolean; $x: number; $y: number }>`
  position: fixed;
  background-color: #282828;
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  padding: 8px 0;
  z-index: 1000;
  display: ${props => props.$visible ? 'block' : 'none'};
  left: ${props => props.$x}px;
  top: ${props => props.$y}px;
  min-width: 200px;
`

const ContextMenuItem = styled.button`
  background: none;
  border: none;
  color: #fff;
  padding: 12px 16px;
  cursor: pointer;
  font-size: 14px;
  text-align: left;
  width: 100%;
  display: flex;
  align-items: center;
  gap: 12px;
  transition: background-color 0.2s ease;

  &:hover {
    background-color: #383838;
  }

  &:disabled {
    color: #5e5e5e;
    cursor: not-allowed;
  }
`

const TrackInfo = styled.div`
  flex: 2;
  min-width: 200px; /* Allow flex item to shrink but maintain minimum width */
  max-width: 400px; /* Prevent it from taking too much space */
  
  @media (max-width: 768px) {
    flex: 3; /* Take more space on mobile since album/date are hidden */
    min-width: 150px;
  }
`

const TrackName = styled.div`
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 2px;
  
  @media (max-width: 768px) {
    font-size: 13px;
  }
`

const TrackArtist = styled.div`
  color: #b3b3b3;
  font-size: 12px;
  
  @media (max-width: 768px) {
    font-size: 11px;
  }
`

const TrackDuration = styled.span`
  color: #b3b3b3;
  font-size: 14px;
  flex: 0 0 60px;
  text-align: right;
  margin-right: 48px; /* Make space for context menu */
  
  @media (max-width: 768px) {
    font-size: 12px;
    flex: 0 0 50px;
    margin-right: 40px;
  }
`

const EmptyState = styled.div`
  text-align: center;
  padding: 60px 20px;
  color: #b3b3b3;
  
  @media (max-width: 768px) {
    padding: 40px 16px;
  }
`

const EmptyStateIcon = styled.div`
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.5;
  
  @media (max-width: 768px) {
    font-size: 48px;
    margin-bottom: 12px;
  }
`

const EmptyStateText = styled.div`
  font-size: 18px;
  margin-bottom: 8px;
  
  @media (max-width: 768px) {
    font-size: 16px;
  }
`

const EmptyStateSubtext = styled.div`
  font-size: 14px;
  opacity: 0.8;
  
  @media (max-width: 768px) {
    font-size: 13px;
  }
`

// Modal styles
const ModalOverlay = styled.div`
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
`

const ModalContent = styled.div`
  background-color: #282828;
  border-radius: 8px;
  padding: 32px;
  width: 90%;
  max-width: 500px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  
  @media (max-width: 768px) {
    padding: 24px;
    width: 95%;
    margin: 20px;
  }
`

const ModalHeader = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
`

const ModalTitle = styled.h2`
  color: #fff;
  font-size: 24px;
  font-weight: 700;
  margin: 0;
`

const CloseButton = styled.button`
  background: none;
  border: none;
  color: #b3b3b3;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    background-color: #383838;
    color: #fff;
  }
`

const FormGroup = styled.div`
  margin-bottom: 20px;
`

const Label = styled.label`
  display: block;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
`

const Input = styled.input`
  width: 100%;
  padding: 12px 16px;
  background-color: #3e3e3e;
  border: 1px solid #5e5e5e;
  border-radius: 4px;
  color: #fff;
  font-size: 14px;
  transition: all 0.2s ease;

  &:focus {
    outline: none;
    border-color: ${({ theme }) => theme.colors.accent};
    background-color: #4e4e4e;
  }

  &::placeholder {
    color: #b3b3b3;
  }
`

const Textarea = styled.textarea`
  width: 100%;
  padding: 12px 16px;
  background-color: #3e3e3e;
  border: 1px solid #5e5e5e;
  border-radius: 4px;
  color: #fff;
  font-size: 14px;
  font-family: inherit;
  resize: vertical;
  min-height: 100px;
  transition: all 0.2s ease;

  &:focus {
    outline: none;
    border-color: ${({ theme }) => theme.colors.accent};
    background-color: #4e4e4e;
  }

  &::placeholder {
    color: #b3b3b3;
  }
`

const ModalActions = styled.div`
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 24px;
`

const Button = styled.button<{ $variant?: 'primary' | 'secondary' | 'danger' }>`
  padding: 12px 24px;
  border: none;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  min-width: 100px;

  ${props => props.$variant === 'primary' ? `
    background: ${props.theme.colors.accentGradient};
    color: ${props.theme.colors.accentText};

    &:hover {
      background: ${props.theme.colors.accentGradient};
      transform: scale(1.05);
    }

    &:disabled {
      background-color: #5e5e5e;
      cursor: not-allowed;
      transform: none;
    }
  ` : props.$variant === 'danger' ? `
    background-color: #ff4444;
    color: #fff;

    &:hover {
      background-color: #ff6666;
      transform: scale(1.05);
    }

    &:disabled {
      background-color: #5e5e5e;
      cursor: not-allowed;
      transform: none;
    }
  ` : `
    background-color: transparent;
    color: #b3b3b3;
    border: 1px solid #5e5e5e;

    &:hover {
      background-color: #383838;
      color: #fff;
      border-color: #b3b3b3;
    }
  `}
`

const PlaylistActions = styled.div`
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s ease;
  z-index: 5;

  ${PlaylistCard}:hover & {
    opacity: 1;
  }
`

const ActionButton = styled.button<{ $variant?: 'edit' | 'delete' }>`
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  background-color: rgba(0, 0, 0, 0.7);
  color: #fff;

  ${props => props.$variant === 'edit' ? `
    &:hover {
      background: ${props.theme.colors.accentGradient};
      transform: scale(1.05);
    }
  ` : `
    &:hover {
      background-color: #ff4444;
      transform: scale(1.05);
    }
  `}
`

const DeleteModalContent = styled.div`
  background-color: #282828;
  border-radius: 8px;
  padding: 32px;
  width: 90%;
  max-width: 400px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  
  @media (max-width: 768px) {
    padding: 24px;
    width: 95%;
    margin: 20px;
  }
`

const DeleteModalText = styled.p`
  color: #fff;
  font-size: 16px;
  line-height: 1.5;
  margin-bottom: 24px;
`

const DeleteModalPlaylistName = styled.span`
  color: ${({ theme }) => theme.colors.accent};
  font-weight: 600;
`

const Select = styled.select`
  width: 100%;
  padding: 12px 16px;
  background-color: #3e3e3e;
  border: 1px solid #5e5e5e;
  border-radius: 4px;
  color: #fff;
  font-size: 14px;
  transition: all 0.2s ease;

  &:focus {
    outline: none;
    border-color: ${({ theme }) => theme.colors.accent};
    background-color: #4e4e4e;
  }

  &::placeholder {
    color: #b3b3b3;
  }
`

const formatDuration = (seconds: number): string => {
  const totalSeconds = Math.max(0, Math.floor(Number.isFinite(seconds) ? seconds : 0))
  const minutes = Math.floor(totalSeconds / 60)
  const remainingSeconds = totalSeconds % 60
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
}

const formatDateAdded = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

export const Library: React.FC = () => {
  const { isAuthenticated, token } = useAuth()
  const { currentTrack, isPlaying, currentTime, duration, playFromQueue, playFromQueueAt, playPlaylist, addToQueue, togglePlayPause } = useAudio()
  const navigate = useNavigate()
  const cachedLibraryData = useMemo(() => readCachedLibraryData(), [])
  const [music, setMusic] = useState<Music[]>(() => cachedLibraryData?.music || [])
  const [playlists, setPlaylists] = useState<Playlist[]>(() => cachedLibraryData?.playlists || [])
  const [albums, setAlbums] = useState<Album[]>(() => cachedLibraryData?.albums || [])

  useEffect(() => {
    const refreshPlaylists = () => {
      void playlistAPI.getAllPlaylists()
        .then(setPlaylists)
        .catch(error => console.error('Failed to refresh playlists:', error))
    }
    window.addEventListener(playlistsChangedEvent, refreshPlaylists)
    return () => window.removeEventListener(playlistsChangedEvent, refreshPlaylists)
  }, [])
  
  const [artists, setArtists] = useState<Artist[]>(() => cachedLibraryData?.artists || [])
  const [loading, setLoading] = useState(!cachedLibraryData)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [activeTab, setActiveTab] = useState<LibraryTab>(getStoredLibraryTab)
  const [trackSort, setTrackSort] = useState<TrackSort>('uploaded-desc')
  const [trackView, setTrackView] = useState<TrackView>('list')
  const [sortMenuOpen, setSortMenuOpen] = useState(false)
  const [pluginTrackActions, setPluginTrackActions] = useState<PluginTrackAction[]>([])
  const [podcasts, setPodcasts] = useState<PodcastSearchResult[]>([])
  const [podcastsLoading, setPodcastsLoading] = useState(false)
  const [podcastsError, setPodcastsError] = useState<string | null>(null)
  const [podcastHome, setPodcastHome] = useState<PodcastHomeResponse>({ continue_listening: [], top_podcasts: [], subscriptions: [] })
  const [podcastHomeLoading, setPodcastHomeLoading] = useState(false)
  const [podcastHomeError, setPodcastHomeError] = useState<string | null>(null)
  const [selectedPodcast, setSelectedPodcast] = useState<PodcastSearchResult | null>(null)
  const [podcastEpisodes, setPodcastEpisodes] = useState<PodcastEpisode[]>([])
  const [podcastEpisodesLoading, setPodcastEpisodesLoading] = useState(false)
  const [podcastEpisodesError, setPodcastEpisodesError] = useState<string | null>(null)
	const [downloadedPodcastEpisodes, setDownloadedPodcastEpisodes] = useState<Set<string>>(new Set())
	const [autoDownloadPodcast, setAutoDownloadPodcast] = useState(false)
  const podcastRequestIdRef = useRef(0)
  const sortMenuRef = useRef<HTMLDivElement | null>(null)
  
  // Context menu state
  const [contextMenu, setContextMenu] = useState<{ visible: boolean; x: number; y: number; track: Track | null }>({
    visible: false,
    x: 0,
    y: 0,
    track: null
  })
  const [hoveredTrackIndex, setHoveredTrackIndex] = useState<number | null>(null)
  
  // Modal state
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [showAddToPlaylistModal, setShowAddToPlaylistModal] = useState(false)
  const [playlistName, setPlaylistName] = useState('')
  const [playlistDescription, setPlaylistDescription] = useState('')
  const [editingPlaylist, setEditingPlaylist] = useState<Playlist | null>(null)
  const [deletingPlaylist, setDeletingPlaylist] = useState<Playlist | null>(null)
  const [selectedTrackForPlaylist, setSelectedTrackForPlaylist] = useState<Track | null>(null)
  const [bulkPlaylistTracks, setBulkPlaylistTracks] = useState<Track[]>([])
  const [selectedPlaylistId, setSelectedPlaylistId] = useState('')
  const [isCreating, setIsCreating] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [isAddingToPlaylist, setIsAddingToPlaylist] = useState(false)
  const [isImportingPlaylist, setIsImportingPlaylist] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  const [editError, setEditError] = useState<string | null>(null)
  const [addToPlaylistError, setAddToPlaylistError] = useState<string | null>(null)
  const importPlaylistInput = useRef<HTMLInputElement | null>(null)

  const loadLibraryData = useCallback(async () => {
    const results = await Promise.allSettled([
      musicAPI.getAllMusic(),
      playlistAPI.getAllPlaylists(),
      albumAPI.getAllAlbums(),
      artistAPI.getAllArtists(),
    ])

    const sectionNames = ['tracks', 'playlists', 'albums', 'artists']
    const failedSections: string[] = []

    const nextMusic = results[0].status === 'fulfilled' ? results[0].value : null
    const nextPlaylists = results[1].status === 'fulfilled' ? results[1].value : null
    const nextAlbums = results[2].status === 'fulfilled' ? results[2].value : null
    const nextArtists = results[3].status === 'fulfilled' ? results[3].value : null

    if (nextMusic) {
      setMusic(nextMusic)
    } else {
      failedSections.push(sectionNames[0])
      console.error('Error loading tracks:', results[0].status === 'rejected' ? results[0].reason : 'No data returned')
    }

    if (nextPlaylists) {
      setPlaylists(nextPlaylists)
    } else {
      failedSections.push(sectionNames[1])
      console.error('Error loading playlists:', results[1].status === 'rejected' ? results[1].reason : 'No data returned')
    }

    if (nextAlbums) {
      setAlbums(nextAlbums)
    } else {
      failedSections.push(sectionNames[2])
      console.error('Error loading albums:', results[2].status === 'rejected' ? results[2].reason : 'No data returned')
    }

    if (nextArtists) {
      setArtists(nextArtists)
    } else {
      failedSections.push(sectionNames[3])
      console.error('Error loading artists:', results[3].status === 'rejected' ? results[3].reason : 'No data returned')
    }

    if (failedSections.length === results.length) {
      throw new Error('All library requests failed')
    }

    if (failedSections.length > 0) {
      console.warn(`Some library sections could not be loaded: ${failedSections.join(', ')}`)
    }

    if (nextMusic && nextPlaylists && nextAlbums && nextArtists) {
      writeCachedLibraryData({
        music: nextMusic,
        playlists: nextPlaylists,
        albums: nextAlbums,
        artists: nextArtists,
      })
    }
    setError(null)
  }, [])

  // Check for scan completion and refresh data
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search)
    if (urlParams.get('scan') === 'completed') {
      // Clean URL and refresh data
      window.history.replaceState({}, '', window.location.pathname)
      const fetchData = async () => {
        try {
          await loadLibraryData()
        } catch (err) {
          setError('Failed to load library data')
          console.error('Error fetching data:', err)
        }
      }
      fetchData()
    }
  }, [loadLibraryData])

  useEffect(() => {
    const fetchData = async () => {
      // Only fetch data if user is authenticated
      if (!isAuthenticated || !token) {
        setLoading(false)
        return
      }

      try {
        if (!cachedLibraryData) {
          setLoading(true)
        }
        await loadLibraryData()
      } catch (err) {
        setError('Failed to load library data')
        console.error('Error fetching data:', err)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [cachedLibraryData, isAuthenticated, loadLibraryData, token])

  useEffect(() => {
    if (!isAuthenticated || !token) {
      setPluginTrackActions([])
      return
    }

    let isCurrent = true
    pluginsAPI.getTrackActions()
      .then(actions => {
        if (isCurrent) {
          setPluginTrackActions(actions)
        }
      })
      .catch(error => console.error('Failed to load plugin track actions:', error))

    return () => {
      isCurrent = false
    }
  }, [isAuthenticated, token])

  useEffect(() => {
    if (!isAuthenticated || !token || activeTab !== 'podcasts') {
      return
    }

    const query = searchQuery.trim()
    if (!query) {
      setPodcasts([])
      setPodcastsError(null)
      setPodcastsLoading(false)
      return
    }

    let isCurrent = true
    const timeoutId = window.setTimeout(() => {
      setPodcastsLoading(true)
      setPodcastsError(null)
      podcastsAPI.search(query)
        .then(response => {
          if (isCurrent) {
            setPodcasts(response.results)
          }
        })
        .catch(error => {
          console.error('Failed to search podcasts:', error)
          if (isCurrent) {
            setPodcasts([])
            setPodcastsError('Podcast search could not be loaded.')
          }
        })
        .finally(() => {
          if (isCurrent) {
            setPodcastsLoading(false)
          }
        })
    }, 300)

    return () => {
      isCurrent = false
      window.clearTimeout(timeoutId)
    }
  }, [activeTab, isAuthenticated, searchQuery, token])

  const loadPodcastHome = useCallback(async () => {
    setPodcastHomeLoading(true)
    setPodcastHomeError(null)
    try {
      setPodcastHome(await podcastsAPI.getHome())
    } catch (requestError) {
      console.error('Failed to load podcast home:', requestError)
      setPodcastHomeError('Podcast recommendations could not be loaded.')
    } finally {
      setPodcastHomeLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!isAuthenticated || !token || activeTab !== 'podcasts') return
    void loadPodcastHome()
  }, [activeTab, isAuthenticated, loadPodcastHome, token])

  useEffect(() => {
    const handleProgress = (event: Event) => {
      const saved = (event as CustomEvent<PodcastProgress>).detail
      if (!saved) return
      setPodcastEpisodes(previous => previous.map(episode => episode.id === saved.episode_id
        ? { ...episode, progress_seconds: saved.position_seconds, completed: saved.completed }
        : episode))
      setPodcastHome(previous => {
        const remaining = previous.continue_listening.filter(item =>
          item.podcast_id !== saved.podcast_id || item.episode_id !== saved.episode_id)
        return {
          ...previous,
          continue_listening: saved.position_seconds > 0 && !saved.completed
            ? [saved, ...remaining].slice(0, 12)
            : remaining,
        }
      })
    }
    window.addEventListener('wavenode:podcast-progress', handleProgress)
    return () => window.removeEventListener('wavenode:podcast-progress', handleProgress)
  }, [])

  useEffect(() => {
    const handleClickOutside = () => {
      setContextMenu({ visible: false, x: 0, y: 0, track: null })
    }

    if (contextMenu.visible) {
      document.addEventListener('click', handleClickOutside)
      return () => {
        document.removeEventListener('click', handleClickOutside)
      }
    }
  }, [contextMenu.visible])

  useEffect(() => {
    if (!sortMenuOpen) return
    const closeSortMenu = (event: MouseEvent) => {
      if (!sortMenuRef.current?.contains(event.target as Node)) {
        setSortMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', closeSortMenu)
    return () => document.removeEventListener('mousedown', closeSortMenu)
  }, [sortMenuOpen])

  // Tab-aware search filtering
  const getSearchPlaceholder = () => {
    switch (activeTab) {
      case 'playlists':
        return 'Search playlists...'
      case 'albums':
        return 'Search albums...'
      case 'artists':
        return 'Search artists...'
      case 'tracks':
        return 'Search tracks...'
      case 'downloads':
        return 'Search downloads...'
      case 'podcasts':
        return 'Search podcasts...'
      case 'audiobooks':
        return 'Search public-domain audiobooks...'
      case 'radio':
        return 'Search radio stations...'
      default:
        return 'Search in your library...'
    }
  }

  const filteredPlaylists = playlists.filter(playlist =>
    playlist.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (playlist.description && playlist.description.toLowerCase().includes(searchQuery.toLowerCase()))
  )

  const filteredAlbums = albums.filter(album =>
    album.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    album.artist.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const filteredArtists = artists.filter(artist => {
    const artistName = typeof artist === 'string' ? artist : artist.name
    return artistName && artistName.toLowerCase().includes(searchQuery.toLowerCase())
  })

  const filteredMusic = useMemo(() => music.filter(track =>
    track.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    track.artist.toLowerCase().includes(searchQuery.toLowerCase()) ||
    track.album.toLowerCase().includes(searchQuery.toLowerCase())
  ), [music, searchQuery])
  const sortedMusic = useMemo(() => {
    const tracks = [...filteredMusic]
    const compareText = (left: string, right: string) =>
      left.localeCompare(right, undefined, { sensitivity: 'base', numeric: true })
    const uploadedAt = (track: Track) => {
      const timestamp = Date.parse(track.created_at)
      return Number.isNaN(timestamp) ? 0 : timestamp
    }

    tracks.sort((left, right) => {
      switch (trackSort) {
        case 'title-asc':
          return compareText(left.title, right.title)
        case 'artist-asc':
          return compareText(left.artist, right.artist) || compareText(left.title, right.title)
        case 'album-asc':
          return compareText(left.album, right.album) || compareText(left.title, right.title)
        case 'duration-asc':
          return left.duration - right.duration
        case 'uploaded-desc':
        default:
          return uploadedAt(right) - uploadedAt(left)
            || (right.upload_order || 0) - (left.upload_order || 0)
      }
    })
    return tracks
  }, [filteredMusic, trackSort])

  const getUniqueArtists = () => {
    const artistMap = new Map<string, { name: string; count: number; albums: string[] }>()
    music.forEach(track => {
      if (!artistMap.has(track.artist)) {
        artistMap.set(track.artist, {
          name: track.artist,
          count: 0,
          albums: []
        })
      }
      const artist = artistMap.get(track.artist)!
      artist.count++
      if (!artist.albums.includes(track.album)) {
        artist.albums.push(track.album)
      }
    })
    return Array.from(artistMap.values())
  }

  const getDownloadedTracks = () => {
    // This would typically check for downloaded/offline tracks
    // For now, we'll return a subset as "downloaded"
    return music.slice(0, Math.min(3, music.length))
  }
  const filteredDownloads = getDownloadedTracks().filter(track =>
    track.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    track.artist.toLowerCase().includes(searchQuery.toLowerCase()) ||
    track.album.toLowerCase().includes(searchQuery.toLowerCase())
  )
  const selectableTracks = activeTab === 'downloads' ? filteredDownloads : sortedMusic
  const trackSelection = useTrackSelection(selectableTracks)

  // Context menu handlers
  const handleContextMenu = (e: React.MouseEvent, track: Track, index: number) => {
    e.preventDefault()
    e.stopPropagation()
    trackSelection.ensureSelected(index)
    
    // Calculate position ensuring menu stays within viewport
    const menuWidth = 200 // Approximate width of context menu
    const menuHeight = 250 // Approximate height of context menu
    const padding = 8 // Small padding from edges
    
    let x = e.clientX
    let y = e.clientY
    
    // Check if menu would go off right edge
    if (x + menuWidth > window.innerWidth - padding) {
      x = window.innerWidth - menuWidth - padding
    }
    
    // Check if menu would go off bottom edge
    if (y + menuHeight > window.innerHeight - padding) {
      y = window.innerHeight - menuHeight - padding
    }
    
    // Ensure menu doesn't go off left or top edges
    x = Math.max(padding, x)
    y = Math.max(padding, y)
    
    setContextMenu({
      visible: true,
      x,
      y,
      track
    })
  }

  const handleAddToPlaylist = (track: Track) => {
    const tracks = trackSelection.selectedIds.has(track.id) ? trackSelection.selectedTracks : [track]
    if (tracks.length > 1) {
      setBulkPlaylistTracks(tracks)
      setContextMenu({ visible: false, x: 0, y: 0, track: null })
      return
    }
    setSelectedTrackForPlaylist(track)
    setSelectedPlaylistId('')
    setAddToPlaylistError(null)
    setShowAddToPlaylistModal(true)
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleLikeTrack = async (track: Track) => {
    try {
      await likedTracksAPI.likeTrack(track.id)
      // Optionally show success feedback
      console.log('Track liked successfully:', track.title)
    } catch (error) {
      console.error('Failed to like track:', error)
      // Optionally show error feedback
    }
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleAddToQueue = (track: Track) => {
    const tracks = trackSelection.selectedIds.has(track.id) ? trackSelection.selectedTracks : [track]
    tracks.forEach(addToQueue)
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handlePluginTrackAction = (action: PluginTrackAction, track: Track) => {
    const tracks = trackSelection.selectedIds.has(track.id) ? trackSelection.selectedTracks : [track]
    setContextMenu({ visible: false, x: 0, y: 0, track: null })

    if (action.action_type === 'download') {
      void Promise.all(tracks.map(selectedTrack => downloadLibraryTrack(selectedTrack, action))).catch(error => {
        console.error('Failed to download selected tracks:', error)
      })
    }
  }

  const handleGoToArtist = (track: Track) => {
    // For now, we need to find artist hash from artists data
    // This is a temporary solution until we have artist hashes in track data
    const artist = artists.find(a => {
      if (typeof a === 'string') {
        return a === track.artist
      } else {
        return a.name === track.artist
      }
    })
    
    if (artist && typeof artist !== 'string' && artist.id) {
      let artistId = artist.id
      // Remove "artist_" prefix if it exists (for backward compatibility)
      if (artistId.startsWith('artist_')) {
        artistId = artistId.substring(7)
      }
      navigate(`/artist/${artistId}`)
    } else {
      // Fallback to encoded name if no hash found
      navigate(`/artist/${encodeURIComponent(track.artist)}`)
    }
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleGoToAlbum = (track: Track) => {
    navigate(`/album/${encodeURIComponent(track.album)}`)
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  // Add to playlist handlers
  const handleAddTrackToPlaylist = async () => {
    if (!selectedTrackForPlaylist || !selectedPlaylistId) {
      setAddToPlaylistError('Please select a playlist')
      return
    }

    setIsAddingToPlaylist(true)
    setAddToPlaylistError(null)

    try {
      const updatedPlaylist = await playlistAPI.addTrackToPlaylist(selectedPlaylistId, selectedTrackForPlaylist.id)

      if (updatedPlaylist) {
        setPlaylists(playlists.map(p => p.id === selectedPlaylistId ? updatedPlaylist : p))
        setShowAddToPlaylistModal(false)
        setSelectedTrackForPlaylist(null)
        setSelectedPlaylistId('')
        setAddToPlaylistError(null)
        
        // Refresh sidebar playlists
        const windowWithRefresh = window as { refreshSidebarPlaylists?: () => Promise<void> }
        if (windowWithRefresh.refreshSidebarPlaylists) {
          await windowWithRefresh.refreshSidebarPlaylists()
        }
      } else {
        setAddToPlaylistError('Failed to add track to playlist')
      }
    } catch (err) {
      setAddToPlaylistError('Failed to add track to playlist. Please try again.')
      console.error('Error adding track to playlist:', err)
    } finally {
      setIsAddingToPlaylist(false)
    }
  }

  // Create playlist handlers
  const handleImportPlaylist = async (file?: File) => {
    if (!file) return
    setIsImportingPlaylist(true)
    try {
      const importedPlaylist = await playlistAPI.importM3U(file)
      if (!importedPlaylist) {
        window.alert('No tracks in that M3U file matched this library.')
        return
      }
      await loadLibraryData()
      const windowWithRefresh = window as { refreshSidebarPlaylists?: () => Promise<void> }
      if (windowWithRefresh.refreshSidebarPlaylists) {
        await windowWithRefresh.refreshSidebarPlaylists()
      }
      navigate(`/playlist/${importedPlaylist.id}`)
    } catch (err) {
      console.error('Error importing playlist:', err)
      window.alert('No tracks in that M3U file matched this library.')
    } finally {
      setIsImportingPlaylist(false)
      if (importPlaylistInput.current) {
        importPlaylistInput.current.value = ''
      }
    }
  }

  const handleCreatePlaylist = async () => {
    if (!playlistName.trim()) {
      setCreateError('Playlist name is required')
      return
    }

    setIsCreating(true)
    setCreateError(null)

    try {
      const newPlaylist = await playlistAPI.createPlaylist({
        name: playlistName.trim(),
        description: playlistDescription.trim(),
        track_ids: []
      })

      if (newPlaylist) {
        setPlaylists([...playlists, newPlaylist])
        setShowCreateModal(false)
        setPlaylistName('')
        setPlaylistDescription('')
        setCreateError(null)
        
        // Refresh sidebar playlists
        const windowWithRefresh = window as { refreshSidebarPlaylists?: () => Promise<void> }
        if (windowWithRefresh.refreshSidebarPlaylists) {
          await windowWithRefresh.refreshSidebarPlaylists()
        }
      } else {
        setCreateError('Failed to create playlist')
      }
    } catch (err) {
      setCreateError('Failed to create playlist. Please try again.')
      console.error('Error creating playlist:', err)
    } finally {
      setIsCreating(false)
    }
  }

  // Edit playlist handlers
  const handleEditPlaylist = async () => {
    if (!playlistName.trim() || !editingPlaylist) {
      setEditError('Playlist name is required')
      return
    }

    setIsEditing(true)
    setEditError(null)

    try {
      const updatedPlaylist = await playlistAPI.updatePlaylist(editingPlaylist.id, {
        name: playlistName.trim(),
        description: playlistDescription.trim()
      })

      if (updatedPlaylist) {
        setPlaylists(playlists.map(p => p.id === editingPlaylist.id ? updatedPlaylist : p))
        setShowEditModal(false)
        setPlaylistName('')
        setPlaylistDescription('')
        setEditingPlaylist(null)
        setEditError(null)
        
        // Refresh sidebar playlists
        const windowWithRefresh = window as { refreshSidebarPlaylists?: () => Promise<void> }
        if (windowWithRefresh.refreshSidebarPlaylists) {
          await windowWithRefresh.refreshSidebarPlaylists()
        }
      } else {
        setEditError('Failed to update playlist')
      }
    } catch (err) {
      setEditError('Failed to update playlist. Please try again.')
      console.error('Error updating playlist:', err)
    } finally {
      setIsEditing(false)
    }
  }

  // Delete playlist handlers
  const handleDeletePlaylist = async () => {
    if (!deletingPlaylist) return

    setIsDeleting(true)

    try {
      const success = await playlistAPI.deletePlaylist(deletingPlaylist.id)
      
      if (success) {
        setPlaylists(playlists.filter(p => p.id !== deletingPlaylist.id))
        setShowDeleteModal(false)
        setDeletingPlaylist(null)
        
        // Refresh sidebar playlists
        const windowWithRefresh = window as { refreshSidebarPlaylists?: () => Promise<void> }
        if (windowWithRefresh.refreshSidebarPlaylists) {
          await windowWithRefresh.refreshSidebarPlaylists()
        }
      } else {
        console.error('Failed to delete playlist')
      }
    } catch (err) {
      console.error('Error deleting playlist:', err)
    } finally {
      setIsDeleting(false)
    }
  }

  const handleOpenCreateModal = () => {
    setPlaylistName('')
    setPlaylistDescription('')
    setCreateError(null)
    setShowCreateModal(true)
  }

  const handleOpenEditModal = (playlist: Playlist, e: React.MouseEvent) => {
    e.stopPropagation()
    setEditingPlaylist(playlist)
    setPlaylistName(playlist.name)
    setPlaylistDescription(playlist.description || '')
    setEditError(null)
    setShowEditModal(true)
  }

  const handleOpenDeleteModal = (playlist: Playlist, e: React.MouseEvent) => {
    e.stopPropagation()
    setDeletingPlaylist(playlist)
    setShowDeleteModal(true)
  }

  const handleCloseCreateModal = () => {
    if (!isCreating) {
      setShowCreateModal(false)
      setPlaylistName('')
      setPlaylistDescription('')
      setCreateError(null)
    }
  }

  const handleCloseEditModal = () => {
    if (!isEditing) {
      setShowEditModal(false)
      setPlaylistName('')
      setPlaylistDescription('')
      setEditingPlaylist(null)
      setEditError(null)
    }
  }

  const handleCloseDeleteModal = () => {
    if (!isDeleting) {
      setShowDeleteModal(false)
      setDeletingPlaylist(null)
    }
  }

  const handleCloseAddToPlaylistModal = () => {
    if (!isAddingToPlaylist) {
      setShowAddToPlaylistModal(false)
      setSelectedTrackForPlaylist(null)
      setSelectedPlaylistId('')
      setAddToPlaylistError(null)
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent, isEdit: boolean = false) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      if (isEdit) {
        handleEditPlaylist()
      } else {
        handleCreatePlaylist()
      }
    }
  }

  const handleAlbumClick = (albumId: string) => {
    navigate(`/album/${albumId}`)
  }

  const handlePlaylistClick = (playlistId: string) => {
    navigate(`/playlist/${encodeURIComponent(playlistId)}`)
  }

  // FIXED VERSION: Always prioritize backend ID over generated hash
  const handleArtistClick = useCallback((artist: Artist | string) => {
    // Handle both string and object types for artists
    let artistId: string
    let artistName: string
    
    if (typeof artist === 'string') {
      artistName = artist
      artistId = artistName
    } else {
      // Handle enriched Artist object
      const enrichedArtist = artist as {
        id?: string;
        name?: string;
      }
      artistName = enrichedArtist.name || enrichedArtist.id || 'Unknown Artist'
      
      // ALWAYS use the id from backend if available
      if (enrichedArtist.id) {
        artistId = enrichedArtist.id
      } else {
        artistId = artistName
      }
    }
    
    // Remove "artist_" prefix if it exists (for backward compatibility)
    if (artistId.startsWith('artist_')) {
      artistId = artistId.substring(7)
    }
    
    navigate(`/artist/${artistId}`)
  }, [navigate])

  const handlePlayTrack = (track: Track, e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation()
    }
    const tracks = activeTab === 'downloads' ? filteredDownloads : sortedMusic
    const index = tracks.findIndex(item => item.id === track.id)
    playFromQueue(tracks, index === -1 ? 0 : index)
  }

  const handlePlayAlbum = async (albumId: string, albumName: string, artistName: string, e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation()
    }
    
    try {
      console.log('Playing album:', { albumId, albumName, artistName })
      
      // Try to get album tracks from API using album ID
      const response = await albumAPI.getAlbumTracks(albumId)
      
      if (response && 'tracks' in response && response.tracks && response.tracks.length > 0) {
        console.log(`Found ${response.tracks.length} tracks for album "${albumName}" via API`)
        playPlaylist(response.tracks)
      } else if (response && 'success' in response && !response.success && response.similarAlbums) {
        // Handle fallback response with similar albums
        console.warn(`Album "${albumName}" not found, but found ${response.similarAlbums.length} similar albums`)
        // Fall back to local filtering as last resort
        const albumTracks = music.filter(track => track.album === albumName && track.artist === artistName)
        if (albumTracks.length > 0) {
          console.log(`Falling back to local filtering: found ${albumTracks.length} tracks`)
          playPlaylist(albumTracks)
        } else {
          console.warn(`No tracks found for album "${albumName}" locally either`)
        }
      } else {
        // Fall back to local filtering if API fails
        console.warn('API call failed or returned no tracks, falling back to local filtering')
        const albumTracks = music.filter(track => track.album === albumName && track.artist === artistName)
        if (albumTracks.length > 0) {
          console.log(`Found ${albumTracks.length} tracks for album "${albumName}" via local filtering`)
          playPlaylist(albumTracks)
        } else {
          console.warn(`No tracks found for album "${albumName}" locally either`)
        }
      }
    } catch (error) {
      console.error('Error playing album:', error)
      // Fall back to local filtering as last resort
      const albumTracks = music.filter(track => track.album === albumName && track.artist === artistName)
      if (albumTracks.length > 0) {
        console.log(`Found ${albumTracks.length} tracks for album "${albumName}" via local filtering (error fallback)`)
        playPlaylist(albumTracks)
      }
    }
  }

  const handlePlayArtist = (artistName: string, e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation()
    }
    const artistTracks = music.filter(track => track.artist === artistName)
    if (artistTracks.length > 0) {
      playPlaylist(artistTracks)
    }
  }

  const handlePlayPlaylistTracks = (playlist: Playlist, e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation()
    }
    const playlistTracks = music.filter(track => playlist.track_ids?.includes(track.id))
    if (playlistTracks.length > 0) {
      playPlaylist(playlistTracks)
    }
  }

  const handlePlayAllTracks = () => {
    if (sortedMusic.length > 0) {
      playPlaylist(sortedMusic)
    }
  }

  const trackSortLabels: Record<TrackSort, string> = {
    'uploaded-desc': 'Recently uploaded',
    'title-asc': 'Title',
    'artist-asc': 'Artist',
    'album-asc': 'Album',
    'duration-asc': 'Duration',
  }

  const renderPlaylists = () => (
    <TabContent>
      <SectionHeader>
        <SectionTitle>Playlists</SectionTitle>
        <div style={{ display: 'flex', gap: '10px' }}>
          <AddButton onClick={() => importPlaylistInput.current?.click()} disabled={isImportingPlaylist}>
            <Upload size={16} />
            {isImportingPlaylist ? 'Importing...' : 'Import M3U'}
          </AddButton>
          <AddButton onClick={() => navigate('/smart-playlist/new')}>
            <Sparkles size={16} />
            Smart Playlist
          </AddButton>
          <AddButton onClick={handleOpenCreateModal}>
            <Plus size={16} />
            Create Playlist
          </AddButton>
          <input
            ref={importPlaylistInput}
            type="file"
            accept=".m3u,.m3u8,audio/x-mpegurl"
            hidden
            onChange={event => void handleImportPlaylist(event.target.files?.[0])}
          />
        </div>
      </SectionHeader>
      {filteredPlaylists.length === 0 ? (
        <EmptyState>
          <EmptyStateIcon>
            <ListMusic size={64} />
          </EmptyStateIcon>
          <EmptyStateText>No playlists found</EmptyStateText>
          <EmptyStateSubtext>Try adjusting your search terms</EmptyStateSubtext>
        </EmptyState>
      ) : (
        <MusicGrid>
          {filteredPlaylists.map((playlist) => (
            <PlaylistCard 
              key={playlist.id}
              onClick={() => handlePlaylistClick(playlist.id)}
            >
              <PlaylistActions>
                <ActionButton 
                  $variant="edit"
                  onClick={(e) => {
                    if (playlist.type === 'smart') {
                      e.stopPropagation()
                      navigate(`/smart-playlist/${playlist.id}/edit`)
                    } else {
                      handleOpenEditModal(playlist, e)
                    }
                  }}
                  title="Edit playlist"
                >
                  {playlist.type === 'smart' ? <Sparkles size={16} /> : <Edit size={16} />}
                </ActionButton>
                <ActionButton 
                  $variant="delete"
                  onClick={(e) => handleOpenDeleteModal(playlist, e)}
                  title="Delete playlist"
                >
                  <Trash2 size={16} />
                </ActionButton>
              </PlaylistActions>
              <PlaylistArt>
                {playlist.type === 'smart' ? <Sparkles size={48} /> : <ListMusic size={48} />}
                <PlayButton onClick={(e) => handlePlayPlaylistTracks(playlist, e)}>
                  <Play size={20} color="#000" />
                </PlayButton>
              </PlaylistArt>
              <MusicInfo>
                <MusicTitle>{playlist.name}</MusicTitle>
                {playlist.type === 'smart' && <MusicArtist>Smart playlist · updates automatically</MusicArtist>}
                <MusicArtist>{playlist.description || 'No description'}</MusicArtist>
                <MusicMeta>
                  <span>{playlist.track_ids?.length || 0} tracks</span>
                </MusicMeta>
              </MusicInfo>
            </PlaylistCard>
          ))}
        </MusicGrid>
      )}
    </TabContent>
  )

  const renderAlbums = () => {
    return (
      <TabContent>
        <SectionHeader>
          <SectionTitle>Albums</SectionTitle>
        </SectionHeader>
        {filteredAlbums.length === 0 ? (
          <EmptyState>
            <EmptyStateIcon>
              <Disc size={64} />
            </EmptyStateIcon>
            <EmptyStateText>No albums found</EmptyStateText>
            <EmptyStateSubtext>Try adjusting your search terms</EmptyStateSubtext>
          </EmptyState>
        ) : (
          <MusicGrid>
            {filteredAlbums.map((album) => {
              const artworkUrl = getAlbumArtworkUrl(
                album,
                music.filter((track) => track.album === album.name),
              )

              return (
              <AlbumCard
                key={`${album.name}-${album.artist}`}
                onClick={() => handleAlbumClick(album.id)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault()
                    handleAlbumClick(album.id)
                  }
                }}
                role="button"
                tabIndex={0}
                title={`${album.name} by ${album.artist}`}
              >
                <AlbumArt
                  $imageUrl={artworkUrl}
                  $fallback={getArtworkGradient(`${album.name}|${album.artist}`)}
                >
                  {!artworkUrl && <Disc size={52} />}
                  <PlayButton onClick={(e) => handlePlayAlbum(album.id, album.name, album.artist, e)}>
                    <Play size={20} color="#000" />
                  </PlayButton>
                </AlbumArt>
                <MusicInfo>
                  <MusicTitle>{album.name}</MusicTitle>
                  <MusicArtist>{album.artist}</MusicArtist>
                  <MusicMeta>
                    <span>{album.track_count || album.tracks?.length || 0} tracks</span>
                    <span>•</span>
                    <span>{album.year}</span>
                  </MusicMeta>
                </MusicInfo>
              </AlbumCard>
              )
            })}
          </MusicGrid>
        )}
      </TabContent>
    )
  }

  const renderArtists = () => {
    // Handle both string and object types for artists
    const displayArtists = artists.length > 0 ? filteredArtists : getUniqueArtists().filter(artist =>
      artist.name.toLowerCase().includes(searchQuery.toLowerCase())
    )
    
    return (
      <TabContent>
        <SectionHeader>
          <SectionTitle>Artists</SectionTitle>
        </SectionHeader>
        {displayArtists.length === 0 ? (
          <EmptyState>
            <EmptyStateIcon>
              <User size={64} />
            </EmptyStateIcon>
            <EmptyStateText>No artists found</EmptyStateText>
            <EmptyStateSubtext>Try adjusting your search terms</EmptyStateSubtext>
          </EmptyState>
        ) : (
          <MusicGrid>
            {displayArtists.map((artist) => {
              // Handle both string and object types for artists
              let artistName: string
              let imageUrl: string | undefined
              let trackCount: number
              let albumCount: number
              
              if (typeof artist === 'string') {
                // Backend is returning simple strings
                artistName = artist
                // Get track and album counts from music data
                const artistTracks = music.filter(track => track.artist === artistName)
                imageUrl = artistTracks.map(getTrackArtworkUrl).find(Boolean)
                trackCount = artistTracks.length
                const uniqueAlbums = new Set(artistTracks.map(track => track.album))
                albumCount = uniqueAlbums.size
              } else {
                // Handle enriched Artist object
                const enrichedArtist = artist as {
                  name?: string;
                  id?: string;
                  image_medium_url?: string;
                  image_url?: string;
                  image_small_url?: string;
                  image_large_url?: string;
                  track_count?: number;
                  album_count?: number;
                }
                artistName = enrichedArtist.name || enrichedArtist.id || 'Unknown Artist'
                imageUrl = enrichedArtist.image_medium_url
                  || enrichedArtist.image_large_url
                  || enrichedArtist.image_url
                  || enrichedArtist.image_small_url

                // Prefer backend counts because they include split and featured artists.
                const artistTracks = music.filter(track => track.artist === artistName)
                imageUrl = imageUrl || artistTracks.map(getTrackArtworkUrl).find(Boolean)
                trackCount = enrichedArtist.track_count ?? artistTracks.length
                const uniqueAlbums = new Set(artistTracks.map(track => track.album))
                albumCount = enrichedArtist.album_count ?? uniqueAlbums.size
              }
              
              return (
                <ArtistCard 
                  key={artistName}
                  onClick={() => handleArtistClick(artist)}
                >
              <ArtistArt $imageUrl={imageUrl} $fallback={getArtworkGradient(artistName)}>
                {imageUrl ? null : <User size={48} />}
                <PlayButton onClick={(e) => handlePlayArtist(artistName, e)}>
                  <Play size={20} color="#000" />
                </PlayButton>
              </ArtistArt>
                  <MusicInfo>
                    <MusicTitle>{artistName}</MusicTitle>
                    <MusicMeta>
                      <span>{trackCount} tracks</span>
                      <span>•</span>
                      <span>{albumCount} albums</span>
                    </MusicMeta>
                  </MusicInfo>
                </ArtistCard>
              )
            })}
          </MusicGrid>
        )}
      </TabContent>
    )
  }

  const renderTracks = () => (
    <TabContent>
      <SectionHeader>
        <SectionTitle>All Tracks ({music.length})</SectionTitle>
        {filteredMusic.length > 0 && (
          <TrackHeaderActions ref={sortMenuRef}>
            <SortButton
              type="button"
              $open={sortMenuOpen}
              aria-haspopup="menu"
              aria-expanded={sortMenuOpen}
              onClick={() => setSortMenuOpen(open => !open)}
            >
              <ArrowUpDown size={16} aria-hidden="true" />
              <SortButtonLabel>{trackSortLabels[trackSort]}</SortButtonLabel>
              {trackView === 'compact' ? <Rows3 size={16} /> : <List size={16} />}
            </SortButton>
            {sortMenuOpen && (
              <SortMenu role="menu" aria-label="Track sorting and view options">
                <SortMenuLabel>Sort by</SortMenuLabel>
                {([
                  ['title-asc', 'Title'],
                  ['artist-asc', 'Artist'],
                  ['album-asc', 'Album'],
                  ['uploaded-desc', 'Recently uploaded'],
                  ['duration-asc', 'Duration'],
                ] as Array<[TrackSort, string]>).map(([value, label]) => (
                  <SortMenuItem
                    key={value}
                    type="button"
                    role="menuitemradio"
                    aria-checked={trackSort === value}
                    $active={trackSort === value}
                    onClick={() => {
                      setTrackSort(value)
                      setSortMenuOpen(false)
                    }}
                  >
                    <span>{label}</span>
                    {trackSort === value && <Check size={18} />}
                  </SortMenuItem>
                ))}
                <SortMenuDivider />
                <SortMenuLabel>View as</SortMenuLabel>
                <SortMenuItem
                  type="button"
                  role="menuitemradio"
                  aria-checked={trackView === 'compact'}
                  $active={trackView === 'compact'}
                  onClick={() => {
                    setTrackView('compact')
                    setSortMenuOpen(false)
                  }}
                >
                  <SortMenuItemText><Rows3 size={18} />Compact</SortMenuItemText>
                  {trackView === 'compact' && <Check size={18} />}
                </SortMenuItem>
                <SortMenuItem
                  type="button"
                  role="menuitemradio"
                  aria-checked={trackView === 'list'}
                  $active={trackView === 'list'}
                  onClick={() => {
                    setTrackView('list')
                    setSortMenuOpen(false)
                  }}
                >
                  <SortMenuItemText><List size={18} />List</SortMenuItemText>
                  {trackView === 'list' && <Check size={18} />}
                </SortMenuItem>
              </SortMenu>
            )}
            <PlayAllButton onClick={handlePlayAllTracks}>
              <Play size={16} />
              Play All
            </PlayAllButton>
          </TrackHeaderActions>
        )}
      </SectionHeader>
      {filteredMusic.length === 0 ? (
        <EmptyState>
          <EmptyStateIcon>
            <Music2 size={64} />
          </EmptyStateIcon>
          <EmptyStateText>No tracks found</EmptyStateText>
          <EmptyStateSubtext>Try adjusting your search terms</EmptyStateSubtext>
          </EmptyState>
        ) : (
          <TrackList $compact={trackView === 'compact'}>
            {sortedMusic.map((track, index) => {
              const artworkUrl = getTrackArtworkUrl(track)
              return (
              <TrackItem 
                key={track.id}
                ref={element => { trackSelection.rowRefs.current[index] = element }}
                role="option"
                tabIndex={0}
                aria-selected={trackSelection.selectedIds.has(track.id)}
                $selected={trackSelection.selectedIds.has(track.id)}
                $compact={trackView === 'compact'}
                onMouseEnter={() => setHoveredTrackIndex(index)}
                onMouseLeave={() => setHoveredTrackIndex(null)}
                onClick={event => trackSelection.selectIndex(index, event)}
                onDoubleClick={() => handlePlayTrack(track)}
                onKeyDown={event => trackSelection.handleKeyDown(index, event, () => handlePlayTrack(track))}
                onContextMenu={event => handleContextMenu(event, track, index)}
              >
                <TrackNumberContainer>
                  <TrackNumber $hidden={hoveredTrackIndex === index}>
                    {index + 1}
                  </TrackNumber>
                  <PlayIcon 
                    $visible={hoveredTrackIndex === index}
                    aria-label={`Play ${track.title}`}
                    onClick={(e) => {
                      e.stopPropagation()
                      handlePlayTrack(track, e)
                    }}
                  >
                    <Play size={16} />
                  </PlayIcon>
                </TrackNumberContainer>
              {trackView === 'list' ? (
                <TrackCoverArt $imageUrl={artworkUrl}>
                  {!artworkUrl && <Music2 size={20} />}
                </TrackCoverArt>
              ) : null}
              <TrackInfo>
                <TrackName>{track.title}</TrackName>
                <TrackArtist>{track.artist}</TrackArtist>
              </TrackInfo>
              <TrackAlbum>{track.album}</TrackAlbum>
              <TrackDateAdded>{formatDateAdded(track.created_at)}</TrackDateAdded>
              <TrackDuration>{formatDuration(track.duration)}</TrackDuration>
              <ContextMenuButton 
                $visible={hoveredTrackIndex === index}
                onClick={(e) => handleContextMenu(e, track, index)}
                title="More options"
              >
                <MoreVertical size={16} />
              </ContextMenuButton>
            </TrackItem>
              )
            })}
        </TrackList>
      )}
    </TabContent>
  )

  const renderDownloads = () => {
    return (
      <TabContent>
        <SectionHeader>
          <SectionTitle>Downloads</SectionTitle>
        </SectionHeader>
        {filteredDownloads.length === 0 ? (
          <EmptyState>
            <EmptyStateIcon>
              <Download size={64} />
            </EmptyStateIcon>
            <EmptyStateText>No downloaded songs found</EmptyStateText>
            <EmptyStateSubtext>Try adjusting your search terms</EmptyStateSubtext>
          </EmptyState>
        ) : (
          <TrackList>
            {filteredDownloads.map((track, index) => {
              const artworkUrl = getTrackArtworkUrl(track)
              return (
              <TrackItem 
                key={track.id}
                ref={element => { trackSelection.rowRefs.current[index] = element }}
                role="option"
                tabIndex={0}
                aria-selected={trackSelection.selectedIds.has(track.id)}
                $selected={trackSelection.selectedIds.has(track.id)}
                onMouseEnter={() => setHoveredTrackIndex(index)}
                onMouseLeave={() => setHoveredTrackIndex(null)}
                onClick={event => trackSelection.selectIndex(index, event)}
                onDoubleClick={() => handlePlayTrack(track)}
                onKeyDown={event => trackSelection.handleKeyDown(index, event, () => handlePlayTrack(track))}
                onContextMenu={event => handleContextMenu(event, track, index)}
              >
                <TrackNumberContainer>
                  <TrackNumber $hidden={hoveredTrackIndex === index}>
                    {index + 1}
                  </TrackNumber>
                  <PlayIcon 
                    $visible={hoveredTrackIndex === index}
                    aria-label={`Play ${track.title}`}
                    onClick={(e) => {
                      e.stopPropagation()
                      handlePlayTrack(track, e)
                    }}
                  >
                    <Play size={16} />
                  </PlayIcon>
                </TrackNumberContainer>
              <TrackCoverArt $imageUrl={artworkUrl}>
                {!artworkUrl && <Music2 size={20} />}
              </TrackCoverArt>
              <TrackInfo>
                <TrackName>{track.title}</TrackName>
                <TrackArtist>{track.artist}</TrackArtist>
              </TrackInfo>
              <TrackAlbum>{track.album}</TrackAlbum>
              <TrackDateAdded>{formatDateAdded(track.created_at)}</TrackDateAdded>
              <TrackDuration>{formatDuration(track.duration)}</TrackDuration>
              <ContextMenuButton 
                $visible={hoveredTrackIndex === index}
                onClick={(e) => handleContextMenu(e, track, index)}
                title="More options"
              >
                <MoreVertical size={16} />
              </ContextMenuButton>
            </TrackItem>
              )
            })}
        </TrackList>
      )}
    </TabContent>
    )
  }

  const openPodcast = async (podcast: PodcastSearchResult) => {
    const requestId = ++podcastRequestIdRef.current
    setSelectedPodcast(podcast)
    setPodcastEpisodes([])
    setPodcastEpisodesError(null)
    setPodcastEpisodesLoading(true)
    try {
      const response = await podcastsAPI.getEpisodes(podcast.id)
      if (podcastRequestIdRef.current === requestId) {
        setSelectedPodcast(response.podcast)
        setPodcastEpisodes(response.episodes)
		const downloaded = await Promise.all(response.episodes.map(async episode => ({
			id: `podcast:${response.podcast.id}:${episode.id}`,
			exists: await hasPodcastDownload(`podcast:${response.podcast.id}:${episode.id}`),
		})))
		setDownloadedPodcastEpisodes(new Set(downloaded.filter(item => item.exists).map(item => item.id)))
      }
    } catch (requestError) {
      console.error('Failed to load podcast episodes:', requestError)
      if (podcastRequestIdRef.current === requestId) {
        setPodcastEpisodesError('Episodes could not be loaded for this podcast.')
      }
    } finally {
      if (podcastRequestIdRef.current === requestId) {
        setPodcastEpisodesLoading(false)
      }
    }
  }

  const closePodcast = () => {
    podcastRequestIdRef.current += 1
    setSelectedPodcast(null)
    setPodcastEpisodes([])
    setPodcastEpisodesError(null)
    setPodcastEpisodesLoading(false)
  }

  const podcastTrackID = (episode: PodcastEpisode) => `podcast:${selectedPodcast?.id || 'unknown'}:${episode.id}`

  const podcastEpisodeTracks = useMemo<APIMusic[]>(() => {
    if (!selectedPodcast) return []
    return podcastEpisodes.map(episode => ({
      id: `podcast:${selectedPodcast.id}:${episode.id}`,
      title: episode.title || 'Untitled episode',
      artist: selectedPodcast.title || selectedPodcast.publisher || 'Podcast',
      album: selectedPodcast.title || 'Podcast',
      genre: 'Podcast',
      duration: episode.duration || 0,
      release_date: episode.published_at || '',
      file_path: '',
      image_url: episode.image_url || selectedPodcast.image_url || selectedPodcast.thumbnail_url,
      created_at: episode.published_at || '',
      updated_at: episode.published_at || '',
      stream_url: episode.audio_url,
      is_external: true,
      external_kind: 'podcast',
      podcast_id: selectedPodcast.id,
      podcast_title: selectedPodcast.title,
      podcast_publisher: selectedPodcast.publisher,
      podcast_episode_id: episode.id,
      podcast_description: episode.description,
      podcast_website_url: episode.website_url || selectedPodcast.website_url,
	  podcast_audio_url: episode.audio_url,
	  podcast_chapters_url: episode.chapters_url,
	  podcast_chapters_type: episode.chapters_type,
    }))
  }, [podcastEpisodes, selectedPodcast])

  const formatPodcastDate = (value?: string) => {
    if (!value) return ''
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return ''
    return date.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })
  }

  const playPodcastEpisode = async (episode: PodcastEpisode) => {
    const index = podcastEpisodes.findIndex(item => item.id === episode.id)
    let track = podcastEpisodeTracks[index]
    if (!track) return
    if (currentTrack?.id === track.id) {
      void togglePlayPause()
      return
    }
	const offlineUrl = await getPodcastDownloadUrl(track.id)
	if (offlineUrl) track = { ...track, stream_url: offlineUrl }
	const queue = podcastEpisodeTracks.map((item, itemIndex) => itemIndex === index ? track : item)
    playFromQueueAt(queue, index, episode.completed ? 0 : episode.progress_seconds || 0)
  }

  const podcastProgressPercent = (position: number, total: number) => total > 0 ? (position / total) * 100 : 0

  const continueTrack = (progress: PodcastProgress): APIMusic => ({
    id: `podcast:${progress.podcast_id}:${progress.episode_id}`,
    title: progress.episode_title,
    artist: progress.podcast_title || progress.publisher || 'Podcast',
    album: progress.podcast_title || 'Podcast',
    genre: 'Podcast',
    duration: progress.duration_seconds,
    release_date: progress.published_at || '',
    file_path: '',
    image_url: progress.image_url,
    created_at: progress.published_at || '',
    updated_at: progress.updated_at,
    stream_url: progress.audio_url,
    is_external: true,
    external_kind: 'podcast',
    podcast_id: progress.podcast_id,
    podcast_title: progress.podcast_title,
    podcast_publisher: progress.publisher,
    podcast_episode_id: progress.episode_id,
    podcast_description: progress.description,
    podcast_website_url: progress.website_url,
	podcast_audio_url: progress.audio_url,
	podcast_progress_seconds: progress.position_seconds,
	podcast_completed: progress.completed,
  })

  const resumePodcast = async (progress: PodcastProgress) => {
    let track = continueTrack(progress)
    if (currentTrack?.id === track.id) {
      void togglePlayPause()
      return
    }
	try {
		const response = await podcastsAPI.getEpisodes(progress.podcast_id)
		const tracks = response.episodes.map(episode => ({
			...continueTrack({ ...progress, episode_id: episode.id, episode_title: episode.title, description: episode.description,
				audio_url: episode.audio_url, website_url: episode.website_url || response.podcast.website_url || '', image_url: episode.image_url || response.podcast.image_url || '',
				published_at: episode.published_at, duration_seconds: episode.duration, position_seconds: episode.progress_seconds, completed: episode.completed }),
			podcast_chapters_url: episode.chapters_url,
		}));
		const index = Math.max(0, tracks.findIndex(item => item.podcast_episode_id === progress.episode_id));
		const offlineUrl = await getPodcastDownloadUrl(track.id)
		if (offlineUrl) tracks[index] = { ...tracks[index], stream_url: offlineUrl }
		playFromQueueAt(tracks, index, progress.position_seconds)
	} catch {
		const offlineUrl = await getPodcastDownloadUrl(track.id)
		if (offlineUrl) track = { ...track, stream_url: offlineUrl }
		playFromQueueAt([track], 0, progress.position_seconds)
	}
  }

	const togglePodcastSubscription = async () => {
		if (!selectedPodcast) return
		const existing = podcastHome.subscriptions.find(item => item.podcast_id === selectedPodcast.id)
		if (existing) {
			await podcastsAPI.unsubscribe(selectedPodcast.id)
			setPodcastHome(previous => ({ ...previous, subscriptions: previous.subscriptions.filter(item => item.podcast_id !== selectedPodcast.id) }))
			return
		}
		const saved = await podcastsAPI.subscribe({
			podcast_id: selectedPodcast.id, title: selectedPodcast.title, publisher: selectedPodcast.publisher,
			description: selectedPodcast.description || '', image_url: selectedPodcast.image_url || '', thumbnail_url: selectedPodcast.thumbnail_url || '',
			website_url: selectedPodcast.website_url || '', feed_url: selectedPodcast.feed_url || '', auto_download: autoDownloadPodcast, playback_speed: 1,
		})
		setPodcastHome(previous => ({ ...previous, subscriptions: [saved, ...previous.subscriptions.filter(item => item.podcast_id !== saved.podcast_id)] }))
		if (autoDownloadPodcast && podcastEpisodes[0]) await handlePodcastDownload(podcastEpisodes[0])
	}

	const handlePodcastDownload = async (episode: PodcastEpisode) => {
		if (!selectedPodcast) return
		const id = `podcast:${selectedPodcast.id}:${episode.id}`
		await downloadPodcastEpisode(id, episode.audio_url)
		setDownloadedPodcastEpisodes(previous => new Set(previous).add(id))
	}

	const sharePodcastEpisode = async (episode: PodcastEpisode) => {
		const url = episode.website_url || selectedPodcast?.website_url || window.location.href
		if (navigator.share) await navigator.share({ title: episode.title, text: episode.description, url })
		else await navigator.clipboard.writeText(url)
	}

  const renderPodcastShow = () => {
    if (!selectedPodcast) return null
    const artwork = selectedPodcast.image_url || selectedPodcast.thumbnail_url
    return (
      <TabContent>
        <SectionHeader>
          <PodcastBackRow>
            <PodcastIconButton onClick={closePodcast} title="Back to podcast search" aria-label="Back to podcast search">
              <ArrowLeft size={18} />
            </PodcastIconButton>
            <SectionTitle>Episodes</SectionTitle>
          </PodcastBackRow>
        </SectionHeader>
        <PodcastShowHeader>
          <PodcastShowArt $imageUrl={artwork}>
            {!artwork && <Podcast size={42} />}
          </PodcastShowArt>
          <PodcastShowInfo>
            <PodcastShowTitle>{selectedPodcast.title || 'Untitled podcast'}</PodcastShowTitle>
            <PodcastPublisher>{selectedPodcast.publisher || 'Unknown publisher'}</PodcastPublisher>
            {selectedPodcast.description && <PodcastShowDescription>{selectedPodcast.description}</PodcastShowDescription>}
            <PodcastHeaderActions>
              {podcastEpisodes.length > 0 && (
                <Button $variant="primary" onClick={() => playPodcastEpisode(podcastEpisodes[0])}>
                  <Play size={16} /> Play latest
                </Button>
              )}
			  <Button $variant="secondary" onClick={() => void togglePodcastSubscription()}>
				{podcastHome.subscriptions.some(item => item.podcast_id === selectedPodcast.id) ? <><Check size={16} /> Following</> : <><Plus size={16} /> Follow</>}
			  </Button>
			  <label><input type="checkbox" checked={autoDownloadPodcast} onChange={event => setAutoDownloadPodcast(event.target.checked)} /> Auto-download</label>
              {selectedPodcast.website_url && (
                <PodcastIconButton as="a" href={selectedPodcast.website_url} target="_blank" rel="noreferrer" title="Open podcast website" aria-label="Open podcast website">
                  <ExternalLink size={17} />
                </PodcastIconButton>
              )}
            </PodcastHeaderActions>
          </PodcastShowInfo>
        </PodcastShowHeader>
        {podcastEpisodesLoading ? (
          <LoadingMessage>Loading episodes...</LoadingMessage>
        ) : podcastEpisodesError ? (
          <ErrorMessage>{podcastEpisodesError}</ErrorMessage>
        ) : podcastEpisodes.length === 0 ? (
          <EmptyState>
            <EmptyStateIcon><Podcast size={64} /></EmptyStateIcon>
            <EmptyStateText>No playable episodes found</EmptyStateText>
          </EmptyState>
        ) : (
          <PodcastEpisodeList>
            {podcastEpisodes.map(episode => {
              const active = currentTrack?.id === podcastTrackID(episode)
              const published = formatPodcastDate(episode.published_at)
              const shownPosition = active ? currentTime : episode.progress_seconds
              const shownDuration = active && duration > 0 ? duration : episode.duration
              const completed = active
                ? shownDuration > 0 && (shownDuration - shownPosition <= 30 || shownPosition / shownDuration >= 0.95)
                : episode.completed
              return (
                <PodcastEpisodeRow key={episode.id}>
                  <PodcastIconButton
                    onClick={() => playPodcastEpisode(episode)}
                    title={active && isPlaying ? 'Pause episode' : 'Play episode'}
                    aria-label={active && isPlaying ? 'Pause episode' : `Play ${episode.title}`}
                  >
                    {active && isPlaying ? <Pause size={17} /> : <Play size={17} />}
                  </PodcastIconButton>
                  <PodcastEpisodeBody>
                    <PodcastEpisodeTitle>{episode.title || 'Untitled episode'}</PodcastEpisodeTitle>
                    {episode.description && <PodcastEpisodeDescription>{episode.description}</PodcastEpisodeDescription>}
                    <PodcastEpisodeMeta>
                      {published && <span>{published}</span>}
                      {episode.explicit && <span>Explicit</span>}
                      {completed && <PodcastCompleted><CheckCircle2 size={13} /> Played</PodcastCompleted>}
                    </PodcastEpisodeMeta>
                    {(shownPosition > 0 || completed) && (
                      <PodcastProgressTrack aria-label={`${Math.round(podcastProgressPercent(shownPosition, shownDuration))}% listened`}>
                        <PodcastProgressFill $progress={completed ? 100 : podcastProgressPercent(shownPosition, shownDuration)} $completed={completed} />
                      </PodcastProgressTrack>
                    )}
                  </PodcastEpisodeBody>
				  <PodcastEpisodeActions>
					<PodcastIconButton onClick={() => addToQueue(podcastEpisodeTracks[podcastEpisodes.indexOf(episode)])} title="Add to queue" aria-label={`Add ${episode.title} to queue`}><ListPlus size={16} /></PodcastIconButton>
					<PodcastIconButton onClick={() => void handlePodcastDownload(episode)} title={downloadedPodcastEpisodes.has(podcastTrackID(episode)) ? 'Downloaded' : 'Download episode'} aria-label={`Download ${episode.title}`}><Download size={16} /></PodcastIconButton>
					<PodcastIconButton onClick={() => void sharePodcastEpisode(episode)} title="Share episode" aria-label={`Share ${episode.title}`}><Share2 size={16} /></PodcastIconButton>
					<PodcastEpisodeDuration>{episode.duration ? formatDuration(episode.duration) : ''}</PodcastEpisodeDuration>
				  </PodcastEpisodeActions>
                </PodcastEpisodeRow>
              )
            })}
          </PodcastEpisodeList>
        )}
      </TabContent>
    )
  }

  const renderPodcasts = () => {
    if (selectedPodcast) return renderPodcastShow()
    return (
      <TabContent>
        <SectionHeader>
          <SectionTitle>Podcasts</SectionTitle>
        </SectionHeader>
        {!searchQuery.trim() ? (
          podcastHomeLoading ? <LoadingMessage>Loading podcasts...</LoadingMessage> : (
            <>
              {podcastHome.continue_listening.length > 0 && (
                <PodcastHomeSection>
                  <SectionHeader><SectionTitle>Continue listening</SectionTitle></SectionHeader>
                  <PodcastRow>
                    {podcastHome.continue_listening.map(progress => {
                      const active = currentTrack?.id === `podcast:${progress.podcast_id}:${progress.episode_id}`
                      const position = active ? currentTime : progress.position_seconds
                      const total = active && duration > 0 ? duration : progress.duration_seconds
                      return (
                        <PodcastCard key={`${progress.podcast_id}:${progress.episode_id}`} onClick={() => resumePodcast(progress)}>
                          <PodcastArt $imageUrl={progress.image_url}>{!progress.image_url && <Podcast size={34} />}</PodcastArt>
                          <PodcastInfo>
                            <PodcastTitle>{progress.podcast_title}</PodcastTitle>
                            <PodcastPublisher>{progress.episode_title}</PodcastPublisher>
                            <PodcastMeta>{formatDuration(Math.max(0, total - position))} left</PodcastMeta>
                            <PodcastProgressTrack>
                              <PodcastProgressFill $progress={podcastProgressPercent(position, total)} />
                            </PodcastProgressTrack>
                          </PodcastInfo>
                        </PodcastCard>
                      )
                    })}
                  </PodcastRow>
                </PodcastHomeSection>
              )}
			  {podcastHome.subscriptions.length > 0 && (
				<PodcastHomeSection>
				  <SectionHeader><SectionTitle>Following</SectionTitle></SectionHeader>
				  <PodcastRow>{podcastHome.subscriptions.map(subscription => (
					<PodcastCard key={subscription.podcast_id} onClick={() => void openPodcast({ id: subscription.podcast_id, title: subscription.title, publisher: subscription.publisher, description: subscription.description, image_url: subscription.image_url, thumbnail_url: subscription.thumbnail_url, website_url: subscription.website_url, feed_url: subscription.feed_url, explicit: false })}>
					  <PodcastArt $imageUrl={subscription.image_url || subscription.thumbnail_url}>{!subscription.image_url && <Podcast size={34} />}</PodcastArt>
					  <PodcastInfo><PodcastTitle>{subscription.title}</PodcastTitle><PodcastPublisher>{subscription.publisher}</PodcastPublisher><PodcastMeta>Following <ChevronRight size={14} /></PodcastMeta></PodcastInfo>
					</PodcastCard>
				  ))}</PodcastRow>
				</PodcastHomeSection>
			  )}
              <PodcastHomeSection>
                <SectionHeader><SectionTitle>Top podcasts</SectionTitle></SectionHeader>
                {podcastHome.top_podcasts.length > 0 ? (
                  <PodcastRow>
                    {podcastHome.top_podcasts.map(podcast => (
                      <PodcastCard key={podcast.id} onClick={() => void openPodcast(podcast)}>
                        <PodcastArt $imageUrl={podcast.image_url || podcast.thumbnail_url}>
                          {!podcast.image_url && !podcast.thumbnail_url && <Podcast size={34} />}
                        </PodcastArt>
                        <PodcastInfo>
                          <PodcastTitle>{podcast.title}</PodcastTitle>
                          <PodcastPublisher>{podcast.publisher}</PodcastPublisher>
                          <PodcastMeta>Podcast <ChevronRight size={14} /></PodcastMeta>
                        </PodcastInfo>
                      </PodcastCard>
                    ))}
                  </PodcastRow>
                ) : (
                  <EmptyStateSubtext>{podcastHomeError || 'Top podcasts are temporarily unavailable.'}</EmptyStateSubtext>
                )}
              </PodcastHomeSection>
            </>
          )
        ) : podcastsLoading ? (
          <LoadingMessage>Searching podcasts...</LoadingMessage>
        ) : podcastsError ? (
          <ErrorMessage>{podcastsError}</ErrorMessage>
        ) : podcasts.length === 0 ? (
          <EmptyState>
            <EmptyStateIcon><Podcast size={64} /></EmptyStateIcon>
            <EmptyStateText>No podcasts found</EmptyStateText>
            <EmptyStateSubtext>Try a different search term</EmptyStateSubtext>
          </EmptyState>
        ) : (
          <PodcastGrid>
            {podcasts.map(podcast => (
              <PodcastCard key={podcast.id} onClick={() => void openPodcast(podcast)}>
                <PodcastArt $imageUrl={podcast.thumbnail_url || podcast.image_url}>
                  {!podcast.thumbnail_url && !podcast.image_url && <Podcast size={34} />}
                </PodcastArt>
                <PodcastInfo>
                  <PodcastTitle>{podcast.title || 'Untitled podcast'}</PodcastTitle>
                  <PodcastPublisher>{podcast.publisher || 'Unknown publisher'}</PodcastPublisher>
                  {podcast.description && <PodcastDescription>{podcast.description}</PodcastDescription>}
                  <PodcastMeta>
                    {podcast.total_episodes ? <span>{podcast.total_episodes} episodes</span> : null}
                    {podcast.explicit ? <span>Explicit</span> : null}
                    <ChevronRight size={14} />
                  </PodcastMeta>
                </PodcastInfo>
              </PodcastCard>
            ))}
          </PodcastGrid>
        )}
      </TabContent>
    )
  }

  const tabs: Array<{ id: LibraryTab; label: string; icon: LucideIcon }> = [
    { id: 'playlists', label: 'Playlists', icon: ListMusic },
    { id: 'albums', label: 'Albums', icon: Disc },
    { id: 'artists', label: 'Artists', icon: User },
    { id: 'tracks', label: 'Tracks', icon: Music2 },
    { id: 'downloads', label: 'Download', icon: Download },
    { id: 'podcasts', label: 'Podcasts', icon: Podcast },
    { id: 'audiobooks', label: 'Audiobooks', icon: BookOpen },
    { id: 'radio', label: 'Radio', icon: Radio },
  ]

  if (loading) {
    return <LoadingMessage>Loading your music library...</LoadingMessage>
  }

  if (error) {
    return <ErrorMessage>{error}</ErrorMessage>
  }

  return (
    <LibraryContainer>
      <Header>
        <Title>Your Library</Title>
        <SearchContainer>
          <SearchIcon>
            <Search size={20} />
          </SearchIcon>
          <SearchInput
            type="text"
            placeholder={getSearchPlaceholder()}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </SearchContainer>
      </Header>

      <TabsContainer>
        <TabsList>
          {tabs.map((tab) => {
            const Icon = tab.icon
            return (
              <TabButton
                key={tab.id}
                $active={activeTab === tab.id}
                onClick={() => {
                  setActiveTab(tab.id)
                  window.localStorage.setItem(libraryTabStorageKey, tab.id)
                }}
              >
                <Icon size={18} />
                {tab.label}
              </TabButton>
            )
          })}
        </TabsList>

        {activeTab === 'playlists' && renderPlaylists()}
        {activeTab === 'albums' && renderAlbums()}
        {activeTab === 'artists' && renderArtists()}
        {activeTab === 'tracks' && renderTracks()}
        {activeTab === 'downloads' && renderDownloads()}
        {activeTab === 'podcasts' && renderPodcasts()}
        {activeTab === 'audiobooks' && <AudiobooksPanel query={searchQuery} />}
        {activeTab === 'radio' && <RadioPanel query={searchQuery} />}
      </TabsContainer>

      {/* Context Menu */}
      <ContextMenu 
        $visible={contextMenu.visible} 
        $x={contextMenu.x} 
        $y={contextMenu.y}
      >
        {contextMenu.track && (
          <>
            <ContextMenuItem onClick={() => handlePlayTrack(contextMenu.track!)}>
              <Play size={16} />
              Play
            </ContextMenuItem>
            <ContextMenuItem onClick={() => handleAddToQueue(contextMenu.track!)}>
              <PlusCircle size={16} />
              {trackSelection.selectedTracks.length > 1
                ? `Add ${trackSelection.selectedTracks.length} to Queue`
                : 'Add to Queue'}
            </ContextMenuItem>
            <ContextMenuItem onClick={() => handleLikeTrack(contextMenu.track!)}>
              <Heart size={16} />
              Like
            </ContextMenuItem>
            <ContextMenuItem onClick={() => handleAddToPlaylist(contextMenu.track!)}>
              <PlusCircle size={16} />
              {trackSelection.selectedTracks.length > 1
                ? `Add ${trackSelection.selectedTracks.length} to Playlist`
                : 'Add to Playlist'}
            </ContextMenuItem>
            {pluginTrackActions.map(action => {
              const selectedCount = trackSelection.selectedIds.has(contextMenu.track!.id)
                ? trackSelection.selectedTracks.length
                : 1
              return (
                <ContextMenuItem
                  key={`${action.plugin_id}:${action.id}`}
                  onClick={() => handlePluginTrackAction(action, contextMenu.track!)}
                >
                  <Download size={16} />
                  {selectedCount > 1 ? `${action.label} (${selectedCount})` : action.label}
                </ContextMenuItem>
              )
            })}
            <ContextMenuItem onClick={() => handleGoToArtist(contextMenu.track!)}>
              <User size={16} />
              Go to Artist
            </ContextMenuItem>
            <ContextMenuItem onClick={() => handleGoToAlbum(contextMenu.track!)}>
              <Disc size={16} />
              Go to Album
            </ContextMenuItem>
          </>
        )}
      </ContextMenu>
      <AddToPlaylistDialog
        tracks={bulkPlaylistTracks}
        open={bulkPlaylistTracks.length > 0}
        onClose={() => setBulkPlaylistTracks([])}
      />

      {/* Create Playlist Modal */}
      {showCreateModal && (
        <ModalOverlay onClick={handleCloseCreateModal}>
          <ModalContent onClick={(e) => e.stopPropagation()}>
            <ModalHeader>
              <ModalTitle>Create Playlist</ModalTitle>
              <CloseButton onClick={handleCloseCreateModal}>
                <X size={20} />
              </CloseButton>
            </ModalHeader>

            <FormGroup>
              <Label htmlFor="playlist-name">Playlist Name *</Label>
              <Input
                id="playlist-name"
                type="text"
                value={playlistName}
                onChange={(e) => setPlaylistName(e.target.value)}
                placeholder="Enter playlist name"
                onKeyDown={(e) => handleKeyPress(e, false)}
                disabled={isCreating}
                autoFocus
              />
            </FormGroup>

            <FormGroup>
              <Label htmlFor="playlist-description">Description</Label>
              <Textarea
                id="playlist-description"
                value={playlistDescription}
                onChange={(e) => setPlaylistDescription(e.target.value)}
                placeholder="Add an optional description"
                disabled={isCreating}
                rows={4}
              />
            </FormGroup>

            {createError && (
              <div style={{ color: '#ff6b6b', fontSize: '14px', marginBottom: '16px' }}>
                {createError}
              </div>
            )}

            <ModalActions>
              <Button
                $variant="secondary"
                onClick={handleCloseCreateModal}
                disabled={isCreating}
              >
                Cancel
              </Button>
              <Button
                $variant="primary"
                onClick={handleCreatePlaylist}
                disabled={isCreating || !playlistName.trim()}
              >
                {isCreating ? 'Creating...' : 'Create'}
              </Button>
            </ModalActions>
          </ModalContent>
        </ModalOverlay>
      )}

      {/* Edit Playlist Modal */}
      {showEditModal && (
        <ModalOverlay onClick={handleCloseEditModal}>
          <ModalContent onClick={(e) => e.stopPropagation()}>
            <ModalHeader>
              <ModalTitle>Edit Playlist</ModalTitle>
              <CloseButton onClick={handleCloseEditModal}>
                <X size={20} />
              </CloseButton>
            </ModalHeader>

            <FormGroup>
              <Label htmlFor="edit-playlist-name">Playlist Name *</Label>
              <Input
                id="edit-playlist-name"
                type="text"
                value={playlistName}
                onChange={(e) => setPlaylistName(e.target.value)}
                placeholder="Enter playlist name"
                onKeyDown={(e) => handleKeyPress(e, true)}
                disabled={isEditing}
                autoFocus
              />
            </FormGroup>

            <FormGroup>
              <Label htmlFor="edit-playlist-description">Description</Label>
              <Textarea
                id="edit-playlist-description"
                value={playlistDescription}
                onChange={(e) => setPlaylistDescription(e.target.value)}
                placeholder="Add an optional description"
                disabled={isEditing}
                rows={4}
              />
            </FormGroup>

            {editError && (
              <div style={{ color: '#ff6b6b', fontSize: '14px', marginBottom: '16px' }}>
                {editError}
              </div>
            )}

            <ModalActions>
              <Button
                $variant="secondary"
                onClick={handleCloseEditModal}
                disabled={isEditing}
              >
                Cancel
              </Button>
              <Button
                $variant="primary"
                onClick={handleEditPlaylist}
                disabled={isEditing || !playlistName.trim()}
              >
                {isEditing ? 'Updating...' : 'Update'}
              </Button>
            </ModalActions>
          </ModalContent>
        </ModalOverlay>
      )}

      {/* Delete Confirmation Modal */}
      {showDeleteModal && deletingPlaylist && (
        <ModalOverlay onClick={handleCloseDeleteModal}>
          <DeleteModalContent onClick={(e) => e.stopPropagation()}>
            <ModalHeader>
              <ModalTitle>Delete Playlist</ModalTitle>
              <CloseButton onClick={handleCloseDeleteModal}>
                <X size={20} />
              </CloseButton>
            </ModalHeader>

            <DeleteModalText>
              Are you sure you want to delete playlist{' '}
              <DeleteModalPlaylistName>{deletingPlaylist.name}</DeleteModalPlaylistName>?
              This action cannot be undone.
            </DeleteModalText>

            <ModalActions>
              <Button
                $variant="secondary"
                onClick={handleCloseDeleteModal}
                disabled={isDeleting}
              >
                Cancel
              </Button>
              <Button
                $variant="danger"
                onClick={handleDeletePlaylist}
                disabled={isDeleting}
              >
                {isDeleting ? 'Deleting...' : 'Delete'}
              </Button>
            </ModalActions>
          </DeleteModalContent>
        </ModalOverlay>
      )}

      {/* Add to Playlist Modal */}
      {showAddToPlaylistModal && selectedTrackForPlaylist && (
        <ModalOverlay onClick={handleCloseAddToPlaylistModal}>
          <ModalContent onClick={(e) => e.stopPropagation()}>
            <ModalHeader>
              <ModalTitle>Add to Playlist</ModalTitle>
              <CloseButton onClick={handleCloseAddToPlaylistModal}>
                <X size={20} />
              </CloseButton>
            </ModalHeader>

            <FormGroup>
              <Label htmlFor="playlist-select">Select Playlist</Label>
              <Select
                id="playlist-select"
                value={selectedPlaylistId}
                onChange={(e) => setSelectedPlaylistId(e.target.value)}
                disabled={isAddingToPlaylist}
                autoFocus
              >
                <option value="">Choose a playlist...</option>
                {playlists.map((playlist) => (
                  <option key={playlist.id} value={playlist.id}>
                    {playlist.name}
                  </option>
                ))}
              </Select>
            </FormGroup>

            {addToPlaylistError && (
              <div style={{ color: '#ff6b6b', fontSize: '14px', marginBottom: '16px' }}>
                {addToPlaylistError}
              </div>
            )}

            <ModalActions>
              <Button
                $variant="secondary"
                onClick={handleCloseAddToPlaylistModal}
                disabled={isAddingToPlaylist}
              >
                Cancel
              </Button>
              <Button
                $variant="primary"
                onClick={handleAddTrackToPlaylist}
                disabled={isAddingToPlaylist || !selectedPlaylistId}
              >
                {isAddingToPlaylist ? 'Adding...' : 'Add to Playlist'}
              </Button>
            </ModalActions>
          </ModalContent>
        </ModalOverlay>
      )}
    </LibraryContainer>
  )
}
