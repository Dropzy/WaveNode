export const playlistsChangedEvent = 'wavenode:playlists-changed'

export const notifyPlaylistsChanged = () => {
  window.dispatchEvent(new Event(playlistsChangedEvent))
}
