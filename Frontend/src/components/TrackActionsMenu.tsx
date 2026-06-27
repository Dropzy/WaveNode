import React, { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import styled from 'styled-components'
import { Download, ListPlus, MoreVertical, X } from 'lucide-react'
import { musicAPI, playlistAPI, pluginsAPI, type Music, type Playlist, type PluginTrackAction } from '../services/api'
import { playlistsChangedEvent } from '../utils/playlistEvents'

let trackActionsCache: PluginTrackAction[] | null = null

const getPluginTrackActions = async () => {
  if (trackActionsCache) {
    return trackActionsCache
  }
  trackActionsCache = await pluginsAPI.getTrackActions()
  return trackActionsCache
}

const fallbackDownloadFilename = (track: Music) => {
  const filePathName = track.file_path?.split(/[\\/]/).pop()
  return filePathName || `${track.artist} - ${track.title}`
}

const downloadTrack = async (track: Music, action: PluginTrackAction) => {
  if (action.action_type !== 'download') {
    return
  }

  await musicAPI.downloadMusic(track.id, fallbackDownloadFilename(track))
}

const Trigger = styled.button`
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 0;
  border-radius: 50%;
  color: #b3b3b3;
  background: transparent;
  cursor: pointer;

  &:hover,
  &:focus-visible {
    color: #fff;
    background: #333;
    outline: none;
  }
`

const Menu = styled.div<{ $x: number; $y: number }>`
  position: fixed;
  left: ${props => props.$x}px;
  top: ${props => props.$y}px;
  min-width: 190px;
  padding: 6px;
  border: 1px solid #3a3a3a;
  border-radius: 8px;
  background: #282828;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.45);
  z-index: 4000;
`

const MenuItem = styled.button`
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 0;
  border-radius: 5px;
  color: #fff;
  background: transparent;
  cursor: pointer;
  text-align: left;

  &:hover,
  &:focus-visible {
    background: #3a3a3a;
    outline: none;
  }
`

const Overlay = styled.div`
  position: fixed;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgba(0, 0, 0, 0.72);
  backdrop-filter: blur(4px);
  z-index: 4100;
`

const Dialog = styled.div`
  width: min(460px, 100%);
  padding: 24px;
  border: 1px solid #3a3a3a;
  border-radius: 12px;
  background: #242424;
  box-shadow: 0 18px 50px rgba(0, 0, 0, 0.5);
`

const Header = styled.div`
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
`

const Title = styled.h2`
  margin: 0;
  color: #fff;
  font-size: 22px;
`

const Subtitle = styled.p`
  margin: 5px 0 0;
  color: #b3b3b3;
  font-size: 13px;
`

const CloseButton = styled.button`
  display: grid;
  place-items: center;
  padding: 6px;
  border: 0;
  border-radius: 50%;
  color: #b3b3b3;
  background: transparent;
  cursor: pointer;

  &:hover {
    color: #fff;
    background: #383838;
  }
`

const PlaylistList = styled.div`
  max-height: 320px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow-y: auto;
`

const PlaylistButton = styled.button`
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 13px 14px;
  border: 1px solid #3a3a3a;
  border-radius: 8px;
  color: #fff;
  background: #181818;
  cursor: pointer;
  text-align: left;

  &:hover:not(:disabled) {
    border-color: ${({ theme }) => theme.colors.accentHover};
    background: ${({ theme }) => theme.colors.accentSoft};
  }

  &:disabled {
    opacity: 0.65;
    cursor: wait;
  }
`

const Message = styled.p<{ $error?: boolean }>`
  margin: 8px 0 0;
  color: ${props => props.$error ? '#ff6b6b' : '#b3b3b3'};
  font-size: 14px;
`

interface AddToPlaylistDialogProps {
  track?: Music | null
  tracks?: Music[]
  open: boolean
  onClose: () => void
  excludePlaylistId?: string
}

export const AddToPlaylistDialog: React.FC<AddToPlaylistDialogProps> = ({
  track,
  tracks = [],
  open,
  onClose,
  excludePlaylistId,
}) => {
  const [playlists, setPlaylists] = useState<Playlist[]>([])
  const [loading, setLoading] = useState(false)
  const [addingTo, setAddingTo] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const selectedTracks = tracks.length > 0 ? tracks : track ? [track] : []

  useEffect(() => {
    if (!open) return
    let current = true
    const loadPlaylists = () => {
      setLoading(true)
      setError(null)
      void playlistAPI.getAllPlaylists()
        .then(data => {
          if (current) {
            setPlaylists(data.filter(playlist =>
              playlist.id !== excludePlaylistId && playlist.type !== 'smart'))
          }
        })
        .catch(() => {
          if (current) setError('Unable to load playlists')
        })
        .finally(() => {
          if (current) setLoading(false)
        })
    }

    loadPlaylists()
    window.addEventListener(playlistsChangedEvent, loadPlaylists)
    return () => {
      current = false
      window.removeEventListener(playlistsChangedEvent, loadPlaylists)
    }
  }, [excludePlaylistId, open])

  if (!open || selectedTracks.length === 0) {
    return null
  }

  const addTrack = async (playlist: Playlist) => {
    setAddingTo(playlist.id)
    setError(null)
    try {
      const updated = await playlistAPI.addTracksToPlaylist(
        playlist.id,
        selectedTracks.map(selectedTrack => selectedTrack.id),
      )
      if (!updated) {
        throw new Error('Updated playlist was not returned')
      }
      await (window as { refreshSidebarPlaylists?: () => Promise<void> }).refreshSidebarPlaylists?.()
      onClose()
    } catch {
      setError('Failed to add track to playlist')
    } finally {
      setAddingTo(null)
    }
  }

  return createPortal(
    <Overlay onClick={onClose}>
      <Dialog
        role="dialog"
        aria-modal="true"
        aria-labelledby="add-track-to-playlist-title"
        onClick={event => event.stopPropagation()}
      >
        <Header>
          <div>
            <Title id="add-track-to-playlist-title">Add to Playlist</Title>
            <Subtitle>
              {selectedTracks.length === 1
                ? `${selectedTracks[0].title} by ${selectedTracks[0].artist}`
                : `${selectedTracks.length} selected tracks`}
            </Subtitle>
          </div>
          <CloseButton onClick={onClose} aria-label="Close">
            <X size={20} />
          </CloseButton>
        </Header>

        {loading ? (
          <Message>Loading playlists...</Message>
        ) : playlists.length === 0 ? (
          <Message>No other playlists are available.</Message>
        ) : (
          <PlaylistList>
            {playlists.map(playlist => (
              <PlaylistButton
                key={playlist.id}
                disabled={addingTo !== null}
                onClick={() => void addTrack(playlist)}
              >
                <span>{playlist.name}</span>
                <span>{playlist.track_ids?.length || 0} tracks</span>
              </PlaylistButton>
            ))}
          </PlaylistList>
        )}
        {error && <Message $error>{error}</Message>}
      </Dialog>
    </Overlay>,
    document.body,
  )
}

interface TrackActionsMenuProps {
  track: Music
  tracks?: Music[]
  excludePlaylistId?: string
}

export const TrackActionsMenu: React.FC<TrackActionsMenuProps> = ({ track, tracks = [], excludePlaylistId }) => {
  const [menuPosition, setMenuPosition] = useState<{ x: number; y: number } | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [pluginActions, setPluginActions] = useState<PluginTrackAction[]>(trackActionsCache || [])

  useEffect(() => {
    let current = true
    void getPluginTrackActions()
      .then(actions => {
        if (current) setPluginActions(actions)
      })
      .catch(error => console.error('Failed to load plugin track actions:', error))
    return () => {
      current = false
    }
  }, [])

  useEffect(() => {
    if (!menuPosition) {
      return
    }

    const close = () => setMenuPosition(null)
    document.addEventListener('click', close)
    window.addEventListener('resize', close)
    window.addEventListener('scroll', close, true)
    return () => {
      document.removeEventListener('click', close)
      window.removeEventListener('resize', close)
      window.removeEventListener('scroll', close, true)
    }
  }, [menuPosition])

  const openMenu = (event: React.MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation()
    const rect = event.currentTarget.getBoundingClientRect()
    const width = 190
    const x = Math.min(rect.right - width, window.innerWidth - width - 8)
    const y = Math.min(rect.bottom + 4, window.innerHeight - 58)
    setMenuPosition({ x: Math.max(8, x), y: Math.max(8, y) })
  }

  return (
    <>
      <Trigger onClick={openMenu} aria-label={`More options for ${track.title}`}>
        <MoreVertical size={18} />
      </Trigger>
      {menuPosition && createPortal(
        <Menu $x={menuPosition.x} $y={menuPosition.y} onClick={event => event.stopPropagation()}>
          <MenuItem
            onClick={() => {
              setMenuPosition(null)
              setDialogOpen(true)
            }}
          >
            <ListPlus size={17} />
            {tracks.length > 1 ? `Add ${tracks.length} to Playlist` : 'Add to Playlist'}
          </MenuItem>
          {pluginActions.map(action => (
            <MenuItem
              key={`${action.plugin_id}:${action.id}`}
              onClick={() => {
                setMenuPosition(null)
                void downloadTrack(track, action).catch(error => {
                  console.error('Failed to download track:', error)
                })
              }}
            >
              <Download size={17} />
              {action.label}
            </MenuItem>
          ))}
        </Menu>,
        document.body,
      )}
      <AddToPlaylistDialog
        track={track}
        tracks={tracks}
        open={dialogOpen}
        excludePlaylistId={excludePlaylistId}
        onClose={() => setDialogOpen(false)}
      />
    </>
  )
}

interface TrackSelectionContextMenuProps {
  tracks: Music[]
  position: { x: number; y: number } | null
  onClose: () => void
  excludePlaylistId?: string
}

export const TrackSelectionContextMenu: React.FC<TrackSelectionContextMenuProps> = ({
  tracks,
  position,
  onClose,
  excludePlaylistId,
}) => {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [pluginActions, setPluginActions] = useState<PluginTrackAction[]>(trackActionsCache || [])

  useEffect(() => {
    let current = true
    void getPluginTrackActions()
      .then(actions => {
        if (current) setPluginActions(actions)
      })
      .catch(error => console.error('Failed to load plugin track actions:', error))
    return () => {
      current = false
    }
  }, [])

  useEffect(() => {
    if (!position) return
    const close = () => onClose()
    document.addEventListener('click', close)
    window.addEventListener('resize', close)
    window.addEventListener('scroll', close, true)
    return () => {
      document.removeEventListener('click', close)
      window.removeEventListener('resize', close)
      window.removeEventListener('scroll', close, true)
    }
  }, [onClose, position])

  return (
    <>
      {position && tracks.length > 0 && createPortal(
        <Menu
          $x={Math.max(8, Math.min(position.x, window.innerWidth - 198))}
          $y={Math.max(8, Math.min(position.y, window.innerHeight - 58))}
          onClick={event => event.stopPropagation()}
        >
          <MenuItem
            onClick={() => {
              onClose()
              setDialogOpen(true)
            }}
          >
            <ListPlus size={17} />
            {tracks.length > 1 ? `Add ${tracks.length} to Playlist` : 'Add to Playlist'}
          </MenuItem>
          {pluginActions.map(action => (
            <MenuItem
              key={`${action.plugin_id}:${action.id}`}
              onClick={() => {
                onClose()
                void Promise.all(tracks.map(track => downloadTrack(track, action))).catch(error => {
                  console.error('Failed to download selected tracks:', error)
                })
              }}
            >
              <Download size={17} />
              {tracks.length > 1 ? `${action.label} (${tracks.length})` : action.label}
            </MenuItem>
          ))}
        </Menu>,
        document.body,
      )}
      <AddToPlaylistDialog
        tracks={tracks}
        open={dialogOpen}
        excludePlaylistId={excludePlaylistId}
        onClose={() => setDialogOpen(false)}
      />
    </>
  )
}
