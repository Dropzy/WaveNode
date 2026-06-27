import React, { useState, useEffect } from 'react'
import styled from 'styled-components'
import { Heart, Play, Search, Music, MoreVertical, PlusCircle, User, Disc } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { useAudio } from '../contexts/AudioContext'
import { likedTracksAPI } from '../services/api'
import { generateAlbumHash } from '../utils/albumUtils'
import { AddToPlaylistDialog } from '../components/TrackActionsMenu'
import { useTrackSelection } from '../hooks/useTrackSelection'

// Define types - using Music type from api.ts
interface Track {
  id: string
  title: string
  artist: string
  album: string
  duration: number
  release_date: string
  file_path: string
  genre: string
  created_at: string
  updated_at: string
}

const LikedSongsContainer = styled.div`
  padding: 24px;
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
  
  @media (max-width: 768px) {
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
  display: flex;
  align-items: center;
  gap: 12px;
  
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

const TrackList = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
  
  @media (max-width: 768px) {
    gap: 6px;
  }
`

const TrackItem = styled.div<{ $selected?: boolean }>`
  background-color: ${props => props.$selected ? '#3a3a3a' : '#181818'};
  border-radius: 8px;
  padding: 12px 16px;
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
    padding: 10px 12px;
    gap: 10px;
  }
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

const PlayIcon = styled.div<{ $visible?: boolean }>`
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: ${({ theme }) => theme.colors.accent};
  opacity: ${props => props.$visible ? 1 : 0};
  transition: opacity 0.2s ease;
  cursor: pointer;
  
  &:hover {
    color: ${({ theme }) => theme.colors.accentHover};
  }
`

const TrackCoverArt = styled.div`
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #4a90e2, #7bb3f0);
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 16px;
  flex-shrink: 0;
  
  @media (max-width: 768px) {
    width: 35px;
    height: 35px;
    font-size: 14px;
  }
`

const TrackInfo = styled.div`
  flex: 2;
  min-width: 200px;
  max-width: 400px;
  
  @media (max-width: 768px) {
    flex: 3;
    min-width: 150px;
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
    display: none;
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
    display: none;
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

const formatDuration = (seconds: number): string => {
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
}

const formatDateAdded = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', { 
    month: 'short', 
    day: 'numeric', 
    year: 'numeric' 
  })
}

export const LikedSongs: React.FC = () => {
  const { isAuthenticated } = useAuth()
  const { playFromQueue, addToQueue } = useAudio()
  const navigate = useNavigate()
  const [searchQuery, setSearchQuery] = useState('')
  const [likedTracks, setLikedTracks] = useState<Track[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  
  // Context menu state
  const [contextMenu, setContextMenu] = useState<{ visible: boolean; x: number; y: number; track: Track | null }>({
    visible: false,
    x: 0,
    y: 0,
    track: null
  })
  const [hoveredTrackIndex, setHoveredTrackIndex] = useState<number | null>(null)

  // Fetch liked tracks from API
  useEffect(() => {
    const fetchLikedTracks = async () => {
      if (!isAuthenticated) {
        setLoading(false)
        return
      }

      try {
        setLoading(true)
        setError(null)
        const tracks = await likedTracksAPI.getLikedTracks()
        setLikedTracks(tracks)
      } catch (err) {
        console.error('Failed to fetch liked tracks:', err)
        setError('Failed to load liked tracks')
      } finally {
        setLoading(false)
      }
    }

    fetchLikedTracks()
  }, [isAuthenticated])

  const filteredLikedTracks = likedTracks.filter(track =>
    track.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    track.artist.toLowerCase().includes(searchQuery.toLowerCase()) ||
    track.album.toLowerCase().includes(searchQuery.toLowerCase())
  )
  const trackSelection = useTrackSelection(filteredLikedTracks)
  const [playlistTracks, setPlaylistTracks] = useState<Track[]>([])

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

  const handlePlayTrack = async (track: Track, e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation()
    }
    try {
      const index = filteredLikedTracks.findIndex(item => item.id === track.id)
      playFromQueue(filteredLikedTracks, index === -1 ? 0 : index)
    } catch (err) {
      console.error('Failed to play track:', err)
    }
  }

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
    
    // Check if menu would go off the right edge
    if (x + menuWidth > window.innerWidth - padding) {
      x = window.innerWidth - menuWidth - padding
    }
    
    // Check if menu would go off the bottom edge
    if (y + menuHeight > window.innerHeight - padding) {
      y = window.innerHeight - menuHeight - padding
    }
    
    // Ensure menu doesn't go off the left or top edges
    x = Math.max(padding, x)
    y = Math.max(padding, y)
    
    setContextMenu({
      visible: true,
      x,
      y,
      track
    })
  }

  const handleAddToQueue = (track: Track) => {
    const tracks = trackSelection.selectedIds.has(track.id) ? trackSelection.selectedTracks : [track]
    tracks.forEach(addToQueue)
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleAddToPlaylist = (track: Track) => {
    setPlaylistTracks(trackSelection.selectedIds.has(track.id) ? trackSelection.selectedTracks : [track])
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleUnlikeTrack = async (track: Track) => {
    try {
      await likedTracksAPI.unlikeTrack(track.id)
      // Remove from local state
      setLikedTracks(likedTracks.filter(t => t.id !== track.id))
      console.log('Track unliked successfully:', track.title)
    } catch (error) {
      console.error('Failed to unlike track:', error)
    }
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleGoToArtist = (track: Track) => {
    navigate(`/artist/${encodeURIComponent(track.artist)}`)
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleGoToAlbum = (track: Track) => {
    const albumHash = generateAlbumHash(track.album, track.artist)
    navigate(`/album/${albumHash}`)
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  if (!isAuthenticated) {
    return (
      <LikedSongsContainer>
        <EmptyState>
          <EmptyStateIcon>
            <Heart size={64} />
          </EmptyStateIcon>
          <EmptyStateText>Please log in to view your liked songs</EmptyStateText>
          <EmptyStateSubtext>Sign in to see and manage your favorite tracks</EmptyStateSubtext>
        </EmptyState>
      </LikedSongsContainer>
    )
  }

  return (
    <LikedSongsContainer>
      <Header>
        <Title>
          <Heart size={32} />
          Liked Songs
        </Title>
        <SearchContainer>
          <SearchIcon>
            <Search size={20} />
          </SearchIcon>
          <SearchInput
            type="text"
            placeholder="Search in liked songs..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </SearchContainer>
      </Header>

      {loading ? (
        <EmptyState>
          <EmptyStateText>Loading liked songs...</EmptyStateText>
        </EmptyState>
      ) : error ? (
        <EmptyState>
          <EmptyStateText>Error loading liked songs</EmptyStateText>
          <EmptyStateSubtext>{error}</EmptyStateSubtext>
        </EmptyState>
      ) : filteredLikedTracks.length === 0 ? (
        <EmptyState>
          <EmptyStateIcon>
            <Heart size={64} />
          </EmptyStateIcon>
          <EmptyStateText>No liked songs yet</EmptyStateText>
          <EmptyStateSubtext>Tap heart icon on any song to add it to your liked songs</EmptyStateSubtext>
        </EmptyState>
      ) : (
        <>
          <TrackList>
            {filteredLikedTracks.map((track, index) => (
              <TrackItem 
                key={track.id}
                ref={element => { trackSelection.rowRefs.current[index] = element }}
                role="option"
                tabIndex={0}
                aria-selected={trackSelection.selectedIds.has(track.id)}
                $selected={trackSelection.selectedIds.has(track.id)}
                onClick={event => trackSelection.selectIndex(index, event)}
                onDoubleClick={() => void handlePlayTrack(track)}
                onKeyDown={event => trackSelection.handleKeyDown(index, event, () => void handlePlayTrack(track))}
                onMouseEnter={() => setHoveredTrackIndex(index)}
                onMouseLeave={() => setHoveredTrackIndex(null)}
                onContextMenu={(e) => handleContextMenu(e, track, index)}
              >
                <TrackNumberContainer>
                  <TrackNumber $hidden={hoveredTrackIndex === index}>
                    {index + 1}
                  </TrackNumber>
                  <PlayIcon 
                    $visible={hoveredTrackIndex === index}
                    onClick={(e) => {
                      e.stopPropagation()
                      handlePlayTrack(track, e)
                    }}
                  >
                    <Play size={16} />
                  </PlayIcon>
                </TrackNumberContainer>
                <TrackCoverArt>
                  <Music size={20} />
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
            ))}
          </TrackList>

          {/* Context Menu */}
          {contextMenu.track && (
            <ContextMenu 
              $visible={contextMenu.visible} 
              $x={contextMenu.x} 
              $y={contextMenu.y}
            >
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
              <ContextMenuItem onClick={() => handleAddToPlaylist(contextMenu.track!)}>
                <PlusCircle size={16} />
                {trackSelection.selectedTracks.length > 1
                  ? `Add ${trackSelection.selectedTracks.length} to Playlist`
                  : 'Add to Playlist'}
              </ContextMenuItem>
              <ContextMenuItem onClick={() => handleUnlikeTrack(contextMenu.track!)}>
                <Heart size={16} />
                Remove from Liked Songs
              </ContextMenuItem>
              <ContextMenuItem onClick={() => handleGoToArtist(contextMenu.track!)}>
                <User size={16} />
                Go to Artist
              </ContextMenuItem>
              <ContextMenuItem onClick={() => handleGoToAlbum(contextMenu.track!)}>
                <Disc size={16} />
                Go to Album
              </ContextMenuItem>
            </ContextMenu>
          )}
        </>
      )}
      <AddToPlaylistDialog
        tracks={playlistTracks}
        open={playlistTracks.length > 0}
        onClose={() => setPlaylistTracks([])}
      />
    </LikedSongsContainer>
  )
}
