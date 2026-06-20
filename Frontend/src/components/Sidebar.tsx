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
  background:
    linear-gradient(180deg, ${({ theme }) => theme.colors.backgroundElevated}, ${({ theme }) => theme.colors.background});
  border-right: 1px solid ${({ theme }) => theme.colors.border};
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
  background: ${({ theme }) => theme.colors.accentGradient};
  background-clip: text;
  -webkit-background-clip: text;
  color: transparent;
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
  color: ${({ theme }) => theme.colors.muted};
  text-decoration: none;
  border-radius: 14px;
  transition: all 0.2s ease;
  font-size: 14px;
  font-weight: 600;

  &:hover {
    color: ${({ theme }) => theme.colors.text};
    background: ${({ theme }) => theme.colors.controlBg};
  }

  &.active {
    color: ${({ theme }) => theme.colors.text};
    background: ${({ theme }) => theme.colors.surfaceSoft};
    box-shadow: inset 0 0 0 1px ${({ theme }) => theme.colors.border};
  }

  svg {
    width: 24px;
    height: 24px;
  }
`

const Divider = styled.hr`
  border: none;
  border-top: 1px solid ${({ theme }) => theme.colors.border};
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
  color: ${({ theme }) => theme.colors.muted};
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 1.5px;
`

const PlaylistHeaderAction = styled.button`
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 999px;
  color: ${({ theme }) => theme.colors.muted};
  background: transparent;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    color: ${({ theme }) => theme.colors.text};
    border-color: ${({ theme }) => theme.colors.borderStrong};
    background: ${({ theme }) => theme.colors.controlBg};
  }
`

const CreatePlaylistButton = styled.button`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 16px;
  color: ${({ theme }) => theme.colors.accentText};
  background: ${({ theme }) => theme.colors.accentGradient};
  border: 1px solid transparent;
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 14px;
  font-weight: 600;
  width: 100%;

  &:hover {
    filter: brightness(1.08);
    transform: scale(1.02);
  }

  &:disabled {
    background: ${({ theme }) => theme.colors.surfaceStrong};
    color: ${({ theme }) => theme.colors.subtle};
    cursor: not-allowed;
    transform: none;
  }

  svg {
    width: 20px;
    height: 20px;
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
    background: ${({ theme }) => theme.colors.borderStrong};
    border-radius: 3px;
  }
  
  &::-webkit-scrollbar-thumb:hover {
    background: ${({ theme }) => theme.colors.accent};
  }
`

const PlaylistItem = styled(NavLink)`
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  color: ${({ theme }) => theme.colors.muted};
  text-decoration: none;
  border-radius: 12px;
  transition: all 0.2s ease;
  font-size: 13px;
  font-weight: 500;

  &:hover {
    color: ${({ theme }) => theme.colors.text};
    background: ${({ theme }) => theme.colors.controlBg};
  }

  &.active {
    color: ${({ theme }) => theme.colors.text};
    background: ${({ theme }) => theme.colors.surfaceSoft};
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
  color: ${({ theme }) => theme.colors.subtle};
  font-size: 11px;
  white-space: nowrap;
`

const LikedSongsButton = styled.button`
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  color: ${({ theme }) => theme.colors.muted};
  background: none;
  border: none;
  border-radius: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 14px;
  font-weight: 600;

  &:hover {
    color: ${({ theme }) => theme.colors.text};
    background: ${({ theme }) => theme.colors.controlBg};
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
  background-color: ${({ theme }) => theme.colors.overlay};
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 3000;
  backdrop-filter: blur(4px);
`

const ModalContent = styled.div`
  background: ${({ theme }) => theme.colors.surface};
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 18px;
  padding: 32px;
  width: 90%;
  max-width: 500px;
  box-shadow: 0 24px 70px ${({ theme }) => theme.colors.shadow};
  
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
  color: ${({ theme }) => theme.colors.text};
  font-size: 24px;
  font-weight: 700;
  margin: 0;
`

const CloseButton = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.muted};
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    background-color: ${({ theme }) => theme.colors.controlBg};
    color: ${({ theme }) => theme.colors.text};
  }
`

const FormGroup = styled.div`
  margin-bottom: 20px;
`

const Label = styled.label`
  display: block;
  color: ${({ theme }) => theme.colors.text};
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
`

const Input = styled.input`
  width: 100%;
  padding: 12px 16px;
  background-color: ${({ theme }) => theme.colors.backgroundElevated};
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 12px;
  color: ${({ theme }) => theme.colors.text};
  font-size: 14px;
  transition: all 0.2s ease;

  &:focus {
    outline: none;
    border-color: ${({ theme }) => theme.colors.accent};
    background-color: ${({ theme }) => theme.colors.surfaceStrong};
  }

  &::placeholder {
    color: ${({ theme }) => theme.colors.subtle};
  }
`

const Textarea = styled.textarea`
  width: 100%;
  padding: 12px 16px;
  background-color: ${({ theme }) => theme.colors.backgroundElevated};
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 12px;
  color: ${({ theme }) => theme.colors.text};
  font-size: 14px;
  font-family: inherit;
  resize: vertical;
  min-height: 100px;
  transition: all 0.2s ease;

  &:focus {
    outline: none;
    border-color: ${({ theme }) => theme.colors.accent};
    background-color: ${({ theme }) => theme.colors.surfaceStrong};
  }

  &::placeholder {
    color: ${({ theme }) => theme.colors.subtle};
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
    background: ${props.theme.colors.accentGradient};
    color: ${props.theme.colors.accentText};

    &:hover {
      filter: brightness(1.08);
      transform: scale(1.05);
    }

    &:disabled {
      background: ${props.theme.colors.surfaceStrong};
      color: ${props.theme.colors.subtle};
      cursor: not-allowed;
      transform: none;
    }
  ` : `
    background-color: transparent;
    color: ${props.theme.colors.muted};
    border: 1px solid ${props.theme.colors.border};

    &:hover {
      background-color: ${props.theme.colors.controlBg};
      color: ${props.theme.colors.text};
      border-color: ${props.theme.colors.borderStrong};
    }
  `}
`

const ErrorText = styled.div`
  color: ${({ theme }) => theme.colors.danger};
  font-size: 14px;
  margin-bottom: 16px;
`

const PlaylistTypeGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;

  @media (max-width: 520px) {
    grid-template-columns: 1fr;
  }
`

const PlaylistTypeButton = styled.button<{ $selected?: boolean }>`
  display: grid;
  gap: 6px;
  min-height: 92px;
  padding: 14px;
  border: 1px solid ${({ theme, $selected }) => $selected ? theme.colors.borderStrong : theme.colors.border};
  border-radius: 14px;
  color: ${({ theme }) => theme.colors.text};
  background: ${({ theme, $selected }) => $selected ? theme.colors.accentSoft : theme.colors.backgroundElevated};
  text-align: left;
  transition: all 0.2s ease;

  strong {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  span {
    color: ${({ theme }) => theme.colors.muted};
    font-size: 12px;
    line-height: 1.4;
  }

  &:hover {
    border-color: ${({ theme }) => theme.colors.borderStrong};
    background: ${({ theme }) => theme.colors.controlBg};
  }
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
  const [playlistType, setPlaylistType] = useState<'manual' | 'smart'>('manual')
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

    if (playlistType === 'smart') {
      const params = new URLSearchParams({
        name: playlistName.trim(),
      })
      if (playlistDescription.trim()) {
        params.set('description', playlistDescription.trim())
      }
      setShowCreateModal(false)
      setPlaylistName('')
      setPlaylistDescription('')
      setPlaylistType('manual')
      setCreateError(null)
      navigate(`/smart-playlist/new?${params.toString()}`)
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
    setPlaylistType('manual')
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
      setPlaylistType('manual')
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
            <PlaylistHeaderAction
              type="button"
              onClick={() => importPlaylistInput.current?.click()}
              aria-label="Import M3U playlist"
              title="Import M3U playlist"
            >
              <Upload size={16} />
            </PlaylistHeaderAction>
          </PlaylistHeader>
          <CreatePlaylistButton onClick={handleOpenCreateModal}>
            <Plus size={20} />
            Create Playlist
          </CreatePlaylistButton>
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
              <Label>Playlist type</Label>
              <PlaylistTypeGrid>
                <PlaylistTypeButton
                  type="button"
                  $selected={playlistType === 'manual'}
                  onClick={() => setPlaylistType('manual')}
                  disabled={isCreating}
                >
                  <strong><ListMusic size={17} /> Standard playlist</strong>
                  <span>Add and reorder tracks yourself.</span>
                </PlaylistTypeButton>
                <PlaylistTypeButton
                  type="button"
                  $selected={playlistType === 'smart'}
                  onClick={() => setPlaylistType('smart')}
                  disabled={isCreating}
                >
                  <strong><Sparkles size={17} /> Smart playlist</strong>
                  <span>Build rules that update the playlist automatically.</span>
                </PlaylistTypeButton>
              </PlaylistTypeGrid>
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
              <ErrorText>{createError}</ErrorText>
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
                {isCreating ? 'Creating...' : playlistType === 'smart' ? 'Continue' : 'Create'}
              </Button>
            </ModalActions>
          </ModalContent>
        </ModalOverlay>,
        document.body,
      )}
    </>
  )
}
