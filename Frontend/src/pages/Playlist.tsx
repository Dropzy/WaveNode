import React, { useState, useEffect, useRef } from 'react'
import { useParams } from 'react-router-dom'
import styled from 'styled-components'
import { Play, Heart, MoreHorizontal, Music as MusicIcon, MoreVertical, PlusCircle, User, Disc, ArrowLeft, Edit, Trash2, Shuffle, Sparkles, Download } from 'lucide-react'
import { likedTracksAPI, playlistAPI, musicAPI, type Playlist, type Music } from '../services/api'
import { useAudio } from '../contexts/AudioContext'
import { useNavigate } from 'react-router-dom'
import { generateAlbumHash } from '../utils/albumUtils'
import { AddToPlaylistDialog } from '../components/TrackActionsMenu'

const PlaylistContainer = styled.div`
  padding: 24px;
  overflow-y: auto;
  
  @media (max-width: 768px) {
    padding: 16px;
    padding-top: 80px; // Account for mobile menu button
  }
`

const PlaylistHeader = styled.div`
  display: flex;
  align-items: flex-start;
  gap: 24px;
  margin-bottom: 32px;
  
  @media (max-width: 768px) {
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 16px;
    margin-bottom: 24px;
  }
`

const PlaylistArt = styled.div`
  width: 232px;
  height: 232px;
  background: linear-gradient(135deg, #450af5, #c4efd9);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 72px;
  font-weight: bold;
  box-shadow: 0 4px 60px rgba(0, 0, 0, 0.3);
  
  @media (max-width: 768px) {
    width: 200px;
    height: 200px;
    font-size: 60px;
  }
  
  @media (max-width: 480px) {
    width: 160px;
    height: 160px;
    font-size: 48px;
  }
`

const PlaylistInfo = styled.div`
  flex: 1;
  padding-top: 20px;
  
  @media (max-width: 768px) {
    padding-top: 0;
    width: 100%;
  }
`

const BackButton = styled.button`
  background: none;
  border: none;
  color: #b3b3b3;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 24px;
  padding: 8px 0;
  transition: all 0.2s ease;

  &:hover {
    color: #fff;
  }
  
  @media (max-width: 768px) {
    margin-bottom: 16px;
  }
`

const PlaylistType = styled.p`
  color: #b3b3b3;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 1.5px;
  margin-bottom: 8px;
  
  @media (max-width: 768px) {
    font-size: 11px;
  }
`

const PlaylistTitle = styled.h1`
  color: #fff;
  font-size: 72px;
  font-weight: 700;
  margin-bottom: 16px;
  line-height: 1;
  
  @media (max-width: 768px) {
    font-size: 48px;
    margin-bottom: 12px;
  }
  
  @media (max-width: 480px) {
    font-size: 32px;
    margin-bottom: 8px;
  }
`

const PlaylistDescription = styled.p`
  color: #b3b3b3;
  font-size: 14px;
  line-height: 1.6;
  margin-bottom: 24px;
  max-width: 600px;
  
  @media (max-width: 768px) {
    font-size: 13px;
    margin-bottom: 16px;
    max-width: none;
  }
`

const PlaylistMeta = styled.div`
  display: flex;
  align-items: center;
  gap: 32px;
  color: #b3b3b3;
  font-size: 14px;
  
  @media (max-width: 768px) {
    flex-direction: column;
    gap: 8px;
    align-items: center;
    font-size: 13px;
  }
`

const PlaylistActions = styled.div`
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 32px;
  
  @media (max-width: 768px) {
    justify-content: center;
    margin-bottom: 24px;
  }
`

const PlayButton = styled.button`
  width: 56px;
  height: 56px;
  background-color: #1db954;
  border: none;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.3);

  &:hover {
    background-color: #1ed760;
    transform: scale(1.05);
  }
  
  @media (max-width: 768px) {
    width: 48px;
    height: 48px;
  }
`

const ShuffleButton = styled.button`
  width: 56px;
  height: 56px;
  background-color: transparent;
  border: 1px solid #b3b3b3;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  color: #b3b3b3;

  &:hover {
    border-color: #1db954;
    color: #1db954;
    transform: scale(1.05);
  }
  
  @media (max-width: 768px) {
    width: 48px;
    height: 48px;
  }
`

const LikeButton = styled.button`
  width: 48px;
  height: 48px;
  background-color: transparent;
  border: 1px solid #b3b3b3;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    border-color: #fff;
    transform: scale(1.05);
  }

  &.liked {
    background-color: #1db954;
    border-color: #1db954;
  }
  
  @media (max-width: 768px) {
    width: 40px;
    height: 40px;
  }
`

const MoreButton = styled.button`
  width: 48px;
  height: 48px;
  background-color: transparent;
  border: none;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  color: #b3b3b3;

  &:hover {
    background-color: rgba(255, 255, 255, 0.1);
    color: #fff;
  }
  
  @media (max-width: 768px) {
    width: 40px;
    height: 40px;
  }
`

// New track list components from Library
const TrackList = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
  
  @media (max-width: 768px) {
    gap: 6px;
  }
`

const TrackItem = styled.div<{ $selected?: boolean }>`
  background-color: ${props => props.$selected ? '#4a4a4a' : '#181818'};
  border: 1px solid ${props => props.$selected ? '#696969' : 'transparent'};
  border-radius: 8px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;

  &:hover {
    background-color: ${props => props.$selected ? '#555' : '#282828'};
  }

  &:focus-visible {
    outline: 2px solid #1ed760;
    outline-offset: 1px;
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
  color: #1db954;
  opacity: ${props => props.$visible ? 1 : 0};
  transition: opacity 0.2s ease;
  cursor: pointer;
  
  &:hover {
    color: #1ed760;
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

const TrackDuration = styled.span`
  color: #b3b3b3;
  font-size: 14px;
  flex: 0 0 60px;
  text-align: right;
  margin-right: 48px;
  
  @media (max-width: 768px) {
    font-size: 12px;
    flex: 0 0 50px;
    margin-right: 40px;
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
`

const SelectionStatus = styled.div`
  margin: 0 0 10px;
  color: #b3b3b3;
  font-size: 13px;
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

// Modal components
const ModalOverlay = styled.div<{ $visible: boolean }>`
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: ${props => props.$visible ? 'flex' : 'none'};
  align-items: center;
  justify-content: center;
  z-index: 1000;
`

const ModalContent = styled.div`
  background-color: #282828;
  border-radius: 8px;
  padding: 24px;
  min-width: 400px;
  max-width: 90vw;
  max-height: 90vh;
  overflow-y: auto;
  
  @media (max-width: 768px) {
    min-width: 300px;
    padding: 20px;
  }
`

const ModalHeader = styled.h2`
  color: #fff;
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 20px;
`

const ModalForm = styled.form`
  display: flex;
  flex-direction: column;
  gap: 16px;
`

const FormGroup = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
`

const FormLabel = styled.label`
  color: #b3b3b3;
  font-size: 14px;
  font-weight: 600;
`

const FormInput = styled.input`
  background-color: #3e3e3e;
  border: 1px solid #5e5e5e;
  border-radius: 4px;
  padding: 12px 16px;
  color: #fff;
  font-size: 14px;
  
  &:focus {
    outline: none;
    border-color: #1db954;
  }
  
  &::placeholder {
    color: #b3b3b3;
  }
`

const FormTextarea = styled.textarea`
  background-color: #3e3e3e;
  border: 1px solid #5e5e5e;
  border-radius: 4px;
  padding: 12px 16px;
  color: #fff;
  font-size: 14px;
  resize: vertical;
  min-height: 100px;
  font-family: inherit;
  
  &:focus {
    outline: none;
    border-color: #1db954;
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
  
  ${props => {
    switch (props.$variant) {
      case 'primary':
        return `
          background-color: #1db954;
          color: #000;
          
          &:hover {
            background-color: #1ed760;
          }
        `
      case 'danger':
        return `
          background-color: #ff4444;
          color: #fff;
          
          &:hover {
            background-color: #ff6666;
          }
        `
      default:
        return `
          background-color: transparent;
          color: #fff;
          border: 1px solid #b3b3b3;
          
          &:hover {
            border-color: #fff;
            background-color: rgba(255, 255, 255, 0.1);
          }
        `
    }
  }}
`

const DeleteModalContent = styled.div`
  color: #fff;
  
  p {
    margin-bottom: 24px;
    line-height: 1.5;
  }
  
  strong {
    color: #ff6b6b;
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

export const PlaylistPage: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const { playFromQueue, playPlaylist, playPlaylistShuffled, addToQueue } = useAudio()
  const navigate = useNavigate()
  const [playlist, setPlaylist] = useState<Playlist | null>(null)
  const [tracks, setTracks] = useState<Music[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  
  // Context menu state
  const [contextMenu, setContextMenu] = useState<{ visible: boolean; x: number; y: number; track: Music | null }>({
    visible: false,
    x: 0,
    y: 0,
    track: null
  })
  const [hoveredTrackIndex, setHoveredTrackIndex] = useState<number | null>(null)
  const [tracksToAddToPlaylist, setTracksToAddToPlaylist] = useState<Music[]>([])
  const [selectedTrackIds, setSelectedTrackIds] = useState<Set<string>>(new Set())
  const [selectionAnchor, setSelectionAnchor] = useState<number | null>(null)
  const trackRowRefs = useRef<Array<HTMLDivElement | null>>([])
  
  // Playlist context menu state
  const [playlistContextMenu, setPlaylistContextMenu] = useState<{ visible: boolean; x: number; y: number }>({
    visible: false,
    x: 0,
    y: 0
  })
  
  // Edit modal state
  const [editModalOpen, setEditModalOpen] = useState(false)
  const [editingPlaylist, setEditingPlaylist] = useState<Playlist | null>(null)
  
  // Delete modal state
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const [playlistToDelete, setPlaylistToDelete] = useState<Playlist | null>(null)

  useEffect(() => {
    const fetchPlaylistData = async () => {
      if (!id) return

      try {
        setLoading(true)
        setError(null)

        const [playlistData, allMusic] = await Promise.all([
          playlistAPI.getPlaylist(id),
          musicAPI.getAllMusic()
        ])

        if (playlistData) {
          setPlaylist(playlistData)
          
          // Get tracks for this playlist
          const musicByID = new Map(allMusic.map(track => [track.id, track]))
          const playlistTracks = (playlistData.track_ids || [])
            .map(trackID => musicByID.get(trackID))
            .filter((track): track is Music => Boolean(track))
          setTracks(playlistTracks)
        }
      } catch (err) {
        setError('Failed to load playlist')
        console.error('Error fetching playlist:', err)
      } finally {
        setLoading(false)
      }
    }

    fetchPlaylistData()
  }, [id])

  useEffect(() => {
    const handleClickOutside = () => {
      setContextMenu({ visible: false, x: 0, y: 0, track: null })
      setPlaylistContextMenu({ visible: false, x: 0, y: 0 })
    }

    if (contextMenu.visible || playlistContextMenu.visible) {
      document.addEventListener('click', handleClickOutside)
      return () => {
        document.removeEventListener('click', handleClickOutside)
      }
    }
  }, [contextMenu.visible, playlistContextMenu.visible])

  useEffect(() => {
    const clearSelection = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !contextMenu.visible) {
        setSelectedTrackIds(new Set())
        setSelectionAnchor(null)
      }
    }
    document.addEventListener('keydown', clearSelection)
    return () => document.removeEventListener('keydown', clearSelection)
  }, [contextMenu.visible])

  const selectRange = (fromIndex: number, toIndex: number) => {
    const start = Math.min(fromIndex, toIndex)
    const end = Math.max(fromIndex, toIndex)
    setSelectedTrackIds(new Set(tracks.slice(start, end + 1).map(track => track.id)))
  }

  const handleTrackSelection = (index: number, event: React.MouseEvent) => {
    const track = tracks[index]
    if (event.shiftKey && selectionAnchor !== null) {
      selectRange(selectionAnchor, index)
      return
    }
    if (event.ctrlKey || event.metaKey) {
      setSelectedTrackIds(current => {
        const next = new Set(current)
        if (next.has(track.id)) next.delete(track.id)
        else next.add(track.id)
        return next
      })
      setSelectionAnchor(index)
      return
    }
    setSelectedTrackIds(new Set([track.id]))
    setSelectionAnchor(index)
  }

  const handleTrackKeyDown = (index: number, event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Enter') {
      event.preventDefault()
      handlePlayTrack(tracks[index])
      return
    }
    if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return

    event.preventDefault()
    const nextIndex = Math.max(0, Math.min(tracks.length - 1, index + (event.key === 'ArrowDown' ? 1 : -1)))
    if (event.shiftKey) {
      const anchor = selectionAnchor ?? index
      if (selectionAnchor === null) setSelectionAnchor(anchor)
      selectRange(anchor, nextIndex)
    } else {
      setSelectedTrackIds(new Set([tracks[nextIndex].id]))
      setSelectionAnchor(nextIndex)
    }
    trackRowRefs.current[nextIndex]?.focus()
  }

  // Context menu handlers
  const handleContextMenu = (e: React.MouseEvent, track: Music, index: number) => {
    e.preventDefault()
    e.stopPropagation()

    if (!selectedTrackIds.has(track.id)) {
      setSelectedTrackIds(new Set([track.id]))
      setSelectionAnchor(index)
    }
    
    // Calculate position ensuring menu stays within viewport
    const menuWidth = 200
    const menuHeight = 250
    const padding = 8
    
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

  const handlePlayTrack = (track: Music, e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation()
    }
    const index = tracks.findIndex(item => item.id === track.id)
    playFromQueue(tracks, index === -1 ? 0 : index)
  }

  const handlePlayAll = () => {
    if (tracks.length > 0) {
      playPlaylist(tracks)
    }
  }

  const handleShuffleAll = () => {
    if (tracks.length > 0) {
      playPlaylistShuffled(tracks)
    }
  }

  const handleAddToQueue = (track: Music) => {
    addToQueue(track)
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleLikeTrack = async (track: Music) => {
    setContextMenu({ visible: false, x: 0, y: 0, track: null })

    try {
      const liked = await likedTracksAPI.likeTrack(track.id)
      if (!liked) {
        throw new Error('The server did not confirm the track was liked')
      }
    } catch (error) {
      console.error('Failed to like track:', error)
      window.alert('Failed to add this track to Liked Songs. Please try again.')
    }
  }

  const selectedTracks = tracks.filter(track => selectedTrackIds.has(track.id))

  const handleAddSelectionToQueue = () => {
    selectedTracks.forEach(track => addToQueue(track))
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleGoToArtist = (track: Music) => {
    navigate(`/artist/${encodeURIComponent(track.artist)}`)
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleGoToAlbum = (track: Music) => {
    const albumHash = generateAlbumHash(track.album, track.artist)
    navigate(`/album/${albumHash}`)
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleEditTrack = (track: Music) => {
    // For now, we'll use a simple prompt. In a real app, you'd want a proper modal/dialog
    const newTitle = prompt('Edit track title:', track.title)
    if (newTitle && newTitle !== track.title) {
      musicAPI.updateMusic(track.id, { title: newTitle })
        .then(updatedTrack => {
          if (updatedTrack) {
            // Update the track in the local state
            setTracks(prevTracks => 
              prevTracks.map(t => t.id === track.id ? updatedTrack : t)
            )
          }
        })
        .catch(error => {
          console.error('Error updating track:', error)
          alert('Failed to update track')
        })
    }
    setContextMenu({ visible: false, x: 0, y: 0, track: null })
  }

  const handleRemoveTrack = async (track: Music) => {
    if (!id) {
      return
    }

    setContextMenu({ visible: false, x: 0, y: 0, track: null })

    try {
      const updatedPlaylist = await playlistAPI.removeTrackFromPlaylist(id, track.id)
      if (!updatedPlaylist) {
        throw new Error('Updated playlist was not returned')
      }

      setPlaylist(updatedPlaylist)
      setTracks(currentTracks => currentTracks.filter(currentTrack => currentTrack.id !== track.id))

      const windowWithRefresh = window as { refreshSidebarPlaylists?: () => Promise<void> }
      await windowWithRefresh.refreshSidebarPlaylists?.()
    } catch (error) {
      console.error('Error removing track from playlist:', error)
      alert('Failed to remove track from playlist')
    }
  }

  // Playlist context menu handlers
  const handlePlaylistContextMenu = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    
    if (!playlist) return
    
    // Calculate position ensuring menu stays within viewport
    const menuWidth = 200
    const menuHeight = 120
    const padding = 8
    
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
    
    setPlaylistContextMenu({
      visible: true,
      x,
      y
    })
  }

  const handleEditPlaylist = () => {
    if (playlist) {
      if (playlist.type === 'smart') {
        navigate(`/smart-playlist/${playlist.id}/edit`)
        setPlaylistContextMenu({ visible: false, x: 0, y: 0 })
        return
      }
      setEditingPlaylist(playlist)
      setEditModalOpen(true)
    }
    setPlaylistContextMenu({ visible: false, x: 0, y: 0 })
  }

  const handleDeletePlaylist = () => {
    if (playlist) {
      setPlaylistToDelete(playlist)
      setDeleteModalOpen(true)
    }
    setPlaylistContextMenu({ visible: false, x: 0, y: 0 })
  }

  const handleEditSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!editingPlaylist || !id) return
    
    try {
      const updatedPlaylist = await playlistAPI.updatePlaylist(id, {
        name: editingPlaylist.name,
        description: editingPlaylist.description
      })
      
      if (updatedPlaylist) {
        setPlaylist(updatedPlaylist)
        setEditModalOpen(false)
        setEditingPlaylist(null)
        await (window as { refreshSidebarPlaylists?: () => Promise<void> }).refreshSidebarPlaylists?.()
      } else {
        alert('Failed to update playlist')
      }
    } catch (error) {
      console.error('Error updating playlist:', error)
      alert('Failed to update playlist')
    }
  }

  const handleDeleteConfirm = async () => {
    if (!playlistToDelete || !id) return
    
    try {
      const success = await playlistAPI.deletePlaylist(id)
      
      if (success) {
        await (window as { refreshSidebarPlaylists?: () => Promise<void> }).refreshSidebarPlaylists?.()
        setDeleteModalOpen(false)
        setPlaylistToDelete(null)
        navigate('/library')
      } else {
        alert('Failed to delete playlist')
      }
    } catch (error) {
      console.error('Error deleting playlist:', error)
      alert('Failed to delete playlist')
    }
  }

  const handleBackToLibrary = () => {
    navigate('/library')
  }

  if (loading) {
    return <LoadingMessage>Loading playlist...</LoadingMessage>
  }

  if (error || !playlist) {
    return (
      <PlaylistContainer>
        <ErrorMessage>{error || 'Playlist not found'}</ErrorMessage>
        <BackButton onClick={handleBackToLibrary}>
          <ArrowLeft size={16} />
          Back to Library
        </BackButton>
      </PlaylistContainer>
    )
  }

  const totalDuration = tracks.reduce((sum, track) => sum + track.duration, 0)
  const totalMinutes = Math.floor(totalDuration / 60)
  const totalHours = Math.floor(totalMinutes / 60)
  const durationText = totalHours > 0 
    ? `${totalHours} hr ${totalMinutes % 60} min`
    : `${totalMinutes} min`

  return (
    <PlaylistContainer>
      <PlaylistHeader>
        <PlaylistArt>
          {playlist.name.charAt(0).toUpperCase()}
        </PlaylistArt>
        <PlaylistInfo>
          <BackButton onClick={handleBackToLibrary}>
            <ArrowLeft size={16} />
            Back to Library
          </BackButton>
          <PlaylistType>{playlist.type === 'smart' ? 'Smart playlist' : 'Playlist'}</PlaylistType>
          <PlaylistTitle>{playlist.name}</PlaylistTitle>
          <PlaylistDescription>{playlist.description}</PlaylistDescription>
          <PlaylistMeta>
            <span>{playlist.type === 'smart' ? 'Updates automatically' : playlist.name}</span>
            <span>•</span>
            <span>{tracks.length} songs</span>
            <span>•</span>
            <span>{durationText}</span>
          </PlaylistMeta>
        </PlaylistInfo>
      </PlaylistHeader>

      <PlaylistActions>
        <PlayButton onClick={handlePlayAll}>
          <Play size={24} color="#000" />
        </PlayButton>
        <ShuffleButton onClick={handleShuffleAll} title="Shuffle Play">
          <Shuffle size={24} />
        </ShuffleButton>
        <LikeButton>
          <Heart size={20} />
        </LikeButton>
        <MoreButton onClick={handlePlaylistContextMenu}>
          <MoreHorizontal size={20} />
        </MoreButton>
        <MoreButton title="Export M3U playlist" onClick={() => void playlistAPI.exportM3U(playlist.id, playlist.name)}>
          <Download size={20} />
        </MoreButton>
      </PlaylistActions>

      {selectedTrackIds.size > 1 && (
        <SelectionStatus>{selectedTrackIds.size} tracks selected</SelectionStatus>
      )}
      <TrackList role="listbox" aria-label="Playlist tracks" aria-multiselectable="true">
        {tracks.map((track, index) => (
          <TrackItem 
            key={track.id}
            ref={element => { trackRowRefs.current[index] = element }}
            role="option"
            tabIndex={0}
            aria-selected={selectedTrackIds.has(track.id)}
            $selected={selectedTrackIds.has(track.id)}
            onMouseEnter={() => setHoveredTrackIndex(index)}
            onMouseLeave={() => setHoveredTrackIndex(null)}
            onClick={event => handleTrackSelection(index, event)}
            onKeyDown={event => handleTrackKeyDown(index, event)}
            onContextMenu={event => handleContextMenu(event, track, index)}
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
              <MusicIcon size={20} />
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
      <ContextMenu 
        $visible={contextMenu.visible} 
        $x={contextMenu.x} 
        $y={contextMenu.y}
      >
        {contextMenu.track && (
          <>
            {selectedTracks.length === 1 && (
              <ContextMenuItem onClick={() => handlePlayTrack(contextMenu.track!)}>
                <Play size={16} />
                Play
              </ContextMenuItem>
            )}
            <ContextMenuItem onClick={selectedTracks.length > 1 ? handleAddSelectionToQueue : () => handleAddToQueue(contextMenu.track!)}>
              <PlusCircle size={16} />
              {selectedTracks.length > 1 ? `Add ${selectedTracks.length} to Queue` : 'Add to Queue'}
            </ContextMenuItem>
            <ContextMenuItem onClick={() => {
              setTracksToAddToPlaylist(selectedTracks.length ? selectedTracks : [contextMenu.track!])
              setContextMenu({ visible: false, x: 0, y: 0, track: null })
            }}>
              <PlusCircle size={16} />
              {selectedTracks.length > 1 ? `Add ${selectedTracks.length} to Playlist` : 'Add to another Playlist'}
            </ContextMenuItem>
            {selectedTracks.length === 1 && (
              <>
                <ContextMenuItem onClick={() => void handleLikeTrack(contextMenu.track!)}>
                  <Heart size={16} />
                  Like
                </ContextMenuItem>
                <ContextMenuItem onClick={() => handleGoToArtist(contextMenu.track!)}>
                  <User size={16} />
                  Go to Artist
                </ContextMenuItem>
                <ContextMenuItem onClick={() => handleGoToAlbum(contextMenu.track!)}>
                  <Disc size={16} />
                  Go to Album
                </ContextMenuItem>
                <ContextMenuItem onClick={() => handleEditTrack(contextMenu.track!)}>
                  <Edit size={16} />
                  Edit
                </ContextMenuItem>
                {playlist.type !== 'smart' && (
                  <ContextMenuItem onClick={() => void handleRemoveTrack(contextMenu.track!)}>
                    <Trash2 size={16} />
                    Remove from Playlist
                  </ContextMenuItem>
                )}
              </>
            )}
          </>
        )}
      </ContextMenu>

      {/* Playlist Context Menu */}
      <ContextMenu 
        $visible={playlistContextMenu.visible} 
        $x={playlistContextMenu.x} 
        $y={playlistContextMenu.y}
      >
        <ContextMenuItem onClick={handleEditPlaylist}>
          {playlist.type === 'smart' ? <Sparkles size={16} /> : <Edit size={16} />}
          {playlist.type === 'smart' ? 'Edit Rules' : 'Edit Playlist'}
        </ContextMenuItem>
        <ContextMenuItem onClick={handleDeletePlaylist}>
          <Trash2 size={16} />
          Delete Playlist
        </ContextMenuItem>
      </ContextMenu>

      {/* Edit Modal */}
      <ModalOverlay $visible={editModalOpen}>
        <ModalContent>
          <ModalHeader>Edit Playlist</ModalHeader>
          <ModalForm onSubmit={handleEditSubmit}>
            <FormGroup>
              <FormLabel>Name</FormLabel>
              <FormInput
                type="text"
                value={editingPlaylist?.name || ''}
                onChange={(e) => setEditingPlaylist(prev => prev ? {...prev, name: e.target.value} : null)}
                placeholder="Playlist name"
                required
              />
            </FormGroup>
            <FormGroup>
              <FormLabel>Description</FormLabel>
              <FormTextarea
                value={editingPlaylist?.description || ''}
                onChange={(e) => setEditingPlaylist(prev => prev ? {...prev, description: e.target.value} : null)}
                placeholder="Playlist description (optional)"
              />
            </FormGroup>
            <ModalActions>
              <Button type="button" onClick={() => setEditModalOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" $variant="primary">
                Save Changes
              </Button>
            </ModalActions>
          </ModalForm>
        </ModalContent>
      </ModalOverlay>

      {/* Delete Confirmation Modal */}
      <ModalOverlay $visible={deleteModalOpen}>
        <ModalContent>
          <DeleteModalContent>
            <p>
              Are you sure you want to delete <strong>{playlistToDelete?.name}</strong>? 
              This action cannot be undone and all tracks will be removed from the playlist.
            </p>
            <ModalActions>
              <Button onClick={() => setDeleteModalOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleDeleteConfirm} $variant="danger">
                Delete Playlist
              </Button>
            </ModalActions>
          </DeleteModalContent>
        </ModalContent>
      </ModalOverlay>
      <AddToPlaylistDialog
        tracks={tracksToAddToPlaylist}
        open={tracksToAddToPlaylist.length > 0}
        excludePlaylistId={id}
        onClose={() => setTracksToAddToPlaylist([])}
      />
    </PlaylistContainer>
  )
}
