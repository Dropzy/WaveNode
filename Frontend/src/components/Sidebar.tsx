import React, { useState, useEffect, useCallback, useRef } from 'react'
import { createPortal } from 'react-dom'
import { NavLink, useNavigate } from 'react-router-dom'
import styled from 'styled-components'
import { Home, Search, Library, Plus, Heart, Settings, X, ListMusic, UserRound, Sparkles, Clock3, Upload } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { playlistAPI, Playlist } from '../services/api'
import { playlistsChangedEvent } from '../utils/playlistEvents'

const SidebarContainer = styled.aside.withConfig({
  shouldForwardProp: (prop) => prop !== 'isOpen',
})<{ isOpen: boolean }>`
  width: 240px;
  background-color: #000000;
  padding: 24px 16px;
  display: flex;
  flex-direction: column;
  gap: 24px;
  position: relative;
  z-index: 1000;
  
  @media (max-width: 768px) {
    position: fixed;
    top: 0;
    left: 0;
    height: 100vh;
    transform: translateX(${props => props.isOpen ? '0' : '-100%'});
    transition: transform 0.3s ease;
    z-index: 1000;
  }
`

const Logo = styled.div`
  font-size: 24px;
  font-weight: bold;
  color: #1db954;
  margin-bottom: 24px;
  
  @media (max-width: 768px) {
    margin-top: 60px; // Account for mobile menu button
  }
`

const Navigation = styled.nav`
  display: flex;
  flex-direction: column;
  gap: 8px;
`

const NavItem = styled(NavLink)`
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  color: #b3b3b3;
  text-decoration: none;
  border-radius: 4px;
  transition: all 0.2s ease;
  font-size: 14px;
  font-weight: 600;

  &:hover {
    color: #fff;
    background-color: #282828;
  }

  &.active {
    color: #fff;
    background-color: #282828;
  }

  svg {
    width: 24px;
    height: 24px;
  }
`

const Divider = styled.hr`
  border: none;
  border-top: 1px solid #282828;
  margin: 0;
`

const PlaylistSection = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
`

const PlaylistHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
`

const PlaylistTitle = styled.h3`
  color: #b3b3b3;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 1.5px;
`

const CreatePlaylistButton = styled.button`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 16px;
  color: #fff;
  background-color: #1db954;
  border: none;
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 14px;
  font-weight: 600;
  width: 100%;

  &:hover {
    background-color: #1ed760;
    transform: scale(1.02);
  }

  &:disabled {
    background-color: #5e5e5e;
    cursor: not-allowed;
    transform: none;
  }

  svg {
    width: 20px;
    height: 20px;
  }
`

const CreateSmartPlaylistButton = styled(CreatePlaylistButton)`
  color: #d8fbe4;
  background: #15271c;
  border: 1px solid #2f6f46;

  &:hover {
    background: #1b3525;
    border-color: #1ed760;
  }
`

const PlaylistList = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 300px;
  overflow-y: auto;
  
  &::-webkit-scrollbar {
    width: 6px;
  }
  
  &::-webkit-scrollbar-track {
    background: transparent;
  }
  
  &::-webkit-scrollbar-thumb {
    background: #404040;
    border-radius: 3px;
  }
  
  &::-webkit-scrollbar-thumb:hover {
    background: #5a5a5a;
  }
`

const PlaylistItem = styled(NavLink)`
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  color: #b3b3b3;
  text-decoration: none;
  border-radius: 4px;
  transition: all 0.2s ease;
  font-size: 13px;
  font-weight: 500;

  &:hover {
    color: #fff;
    background-color: #282828;
  }

  &.active {
    color: #fff;
    background-color: #282828;
  }

  svg {
    width: 18px;
    height: 18px;
    flex-shrink: 0;
  }
`

const PlaylistName = styled.span`
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
`

const PlaylistTrackCount = styled.span`
  color: #808080;
  font-size: 11px;
  white-space: nowrap;
`

const LikedSongsButton = styled.button`
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  color: #b3b3b3;
  background: none;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 14px;
  font-weight: 600;

  &:hover {
    color: #fff;
    background-color: #282828;
  }

  svg {
    width: 24px;
    height: 24px;
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
  z-index: 3000;
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
    border-color: #1db954;
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
    border-color: #1db954;
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

const Button = styled.button<{ $variant?: 'primary' | 'secondary' }>`
  padding: 12px 24px;
  border: none;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  min-width: 100px;

  ${props => props.$variant === 'primary' ? `
    background-color: #1db954;
    color: #fff;

    &:hover {
      background-color: #1ed760;
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

interface SidebarProps {
  isOpen: boolean
  onClose: () => void
}

export const Sidebar: React.FC<SidebarProps> = ({ isOpen, onClose }) => {
  const { user, isAuthenticated } = useAuth()
  const navigate = useNavigate()
  
  // Modal state
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [playlistName, setPlaylistName] = useState('')
  const [playlistDescription, setPlaylistDescription] = useState('')
  const [isCreating, setIsCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  
  // Playlists state
  const [playlists, setPlaylists] = useState<Playlist[]>([])
  const importPlaylistInput = useRef<HTMLInputElement>(null)

  const refreshPlaylists = useCallback(async () => {
      if (!isAuthenticated) {
        setPlaylists([])
        return
      }

      try {
        const playlistsData = await playlistAPI.getAllPlaylists()
        // Sort playlists by created_at in descending order (newest first)
        const sortedPlaylists = playlistsData.sort((a, b) => 
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )
        setPlaylists(sortedPlaylists)
      } catch (error) {
        console.error('Error fetching playlists:', error)
        setPlaylists([])
      }
  }, [isAuthenticated])

  // Fetch playlists when component mounts, auth changes, or another view mutates playlists.
  useEffect(() => {
    void refreshPlaylists()
    const handlePlaylistsChanged = () => void refreshPlaylists()
    window.addEventListener(playlistsChangedEvent, handlePlaylistsChanged)
    return () => window.removeEventListener(playlistsChangedEvent, handlePlaylistsChanged)
  }, [refreshPlaylists])

  // Expose a refresh function for parent components
  useEffect(() => {
    // Store refresh function on window for external access
    ;(window as { refreshSidebarPlaylists?: () => Promise<void> }).refreshSidebarPlaylists = refreshPlaylists

    // Cleanup
    return () => {
      delete (window as { refreshSidebarPlaylists?: () => Promise<void> }).refreshSidebarPlaylists
    }
  }, [refreshPlaylists])
  
  const handleNavClick = () => {
    // Close sidebar on mobile when navigation item is clicked
    if (window.innerWidth <= 768) {
      onClose()
    }
  }

  const handleLikedSongsClick = () => {
    navigate('/liked-songs')
    // Close sidebar on mobile when navigation item is clicked
    if (window.innerWidth <= 768) {
      onClose()
    }
  }

  // Create playlist handlers
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
        // Add new playlist to beginning of list (newest first)
        setPlaylists([newPlaylist, ...playlists])
        setShowCreateModal(false)
        setPlaylistName('')
        setPlaylistDescription('')
        setCreateError(null)
        // Navigate to library to see new playlist
        navigate('/library')
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

  const handlePlaylistClick = (playlistId: string) => {
    navigate(`/playlist/${playlistId}`)
    // Close sidebar on mobile when playlist is clicked
    if (window.innerWidth <= 768) {
      onClose()
    }
  }

  const handleOpenCreateModal = () => {
    setPlaylistName('')
    setPlaylistDescription('')
    setCreateError(null)
    setShowCreateModal(true)
    // Close sidebar on mobile when opening modal
    if (window.innerWidth <= 768) {
      onClose()
    }
  }

  const handleCloseCreateModal = () => {
    if (!isCreating) {
      setShowCreateModal(false)
      setPlaylistName('')
      setPlaylistDescription('')
      setCreateError(null)
    }
  }

  const handleImportPlaylist = async (file?: File) => {
    if (!file) return
    try {
      const playlist = await playlistAPI.importM3U(file)
      if (playlist) {
        await refreshPlaylists()
        navigate(`/playlist/${playlist.id}`)
      }
    } catch (error) {
      console.error('Playlist import failed:', error)
      window.alert('No tracks in that M3U file matched this library.')
    } finally {
      if (importPlaylistInput.current) importPlaylistInput.current.value = ''
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleCreatePlaylist()
    }
  }

  return (
    <>
      <SidebarContainer isOpen={isOpen}>
        <Logo>WaveNode</Logo>
        
        <Navigation>
          <NavItem to="/" end onClick={handleNavClick}>
            <Home size={24} />
            Home
          </NavItem>
          <NavItem to="/search" onClick={handleNavClick}>
            <Search size={24} />
            Search
          </NavItem>
          <NavItem to="/library" onClick={handleNavClick}>
            <Library size={24} />
            Your Library
          </NavItem>
          <LikedSongsButton onClick={handleLikedSongsClick}>
            <Heart size={24} />
            Liked Songs
          </LikedSongsButton>
          {user?.role === 'admin' && (
            <NavItem to="/admin" onClick={handleNavClick}>
              <Settings size={24} />
            Admin Dashboard
            </NavItem>
          )}
          <NavItem to="/account" onClick={handleNavClick}>
            <UserRound size={24} />
            Account
          </NavItem>
          <NavItem to="/history" onClick={handleNavClick}>
            <Clock3 size={24} />
            Listening History
          </NavItem>
        </Navigation>

        <Divider />

        <PlaylistSection>
          <PlaylistHeader>
            <PlaylistTitle>Playlists</PlaylistTitle>
          </PlaylistHeader>
          <CreatePlaylistButton onClick={handleOpenCreateModal}>
            <Plus size={20} />
            Create Playlist
          </CreatePlaylistButton>
          <CreateSmartPlaylistButton onClick={() => navigate('/smart-playlist/new')}>
            <Sparkles size={18} />
            Create Smart Playlist
          </CreateSmartPlaylistButton>
          <CreateSmartPlaylistButton onClick={() => importPlaylistInput.current?.click()}>
            <Upload size={18} />
            Import M3U
          </CreateSmartPlaylistButton>
          <input
            ref={importPlaylistInput}
            type="file"
            accept=".m3u,.m3u8,audio/x-mpegurl"
            hidden
            onChange={event => void handleImportPlaylist(event.target.files?.[0])}
          />
          {isAuthenticated && playlists.length > 0 && (
            <PlaylistList>
              {playlists.map((playlist) => (
                <PlaylistItem
                  key={playlist.id}
                  to={`/playlist/${playlist.id}`}
                  onClick={() => handlePlaylistClick(playlist.id)}
                >
                  {playlist.type === 'smart' ? <Sparkles size={18} /> : <ListMusic size={18} />}
                  <PlaylistName>{playlist.name}</PlaylistName>
                  <PlaylistTrackCount>{playlist.track_ids?.length || 0}</PlaylistTrackCount>
                </PlaylistItem>
              ))}
            </PlaylistList>
          )}
        </PlaylistSection>
      </SidebarContainer>

      {/* Create Playlist Modal */}
      {showCreateModal && createPortal(
        <ModalOverlay onClick={handleCloseCreateModal}>
          <ModalContent
            role="dialog"
            aria-modal="true"
            aria-labelledby="create-playlist-title"
            onClick={(e) => e.stopPropagation()}
          >
            <ModalHeader>
              <ModalTitle id="create-playlist-title">Create Playlist</ModalTitle>
              <CloseButton onClick={handleCloseCreateModal} disabled={isCreating}>
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
                placeholder="My Awesome Playlist"
                onKeyDown={handleKeyPress}
                disabled={isCreating}
                autoFocus
              />
            </FormGroup>

            <FormGroup>
              <Label htmlFor="playlist-description">Description (optional)</Label>
              <Textarea
                id="playlist-description"
                value={playlistDescription}
                onChange={(e) => setPlaylistDescription(e.target.value)}
                placeholder="Add a description to your playlist..."
                disabled={isCreating}
              />
            </FormGroup>

            {createError && (
              <div style={{ color: '#f44336', fontSize: '14px', marginBottom: '16px' }}>
                {createError}
              </div>
            )}

            <ModalActions>
              <Button
                type="button"
                onClick={handleCloseCreateModal}
                disabled={isCreating}
                $variant="secondary"
              >
                Cancel
              </Button>
              <Button
                type="button"
                onClick={handleCreatePlaylist}
                disabled={isCreating || !playlistName.trim()}
                $variant="primary"
              >
                {isCreating ? 'Creating...' : 'Create'}
              </Button>
            </ModalActions>
          </ModalContent>
        </ModalOverlay>,
        document.body,
      )}
    </>
  )
}
