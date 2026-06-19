/// <reference types="vite/client" />

interface Window {
  MUSIC_SERVER_API_BASE_URL?: string
  WAVENODE_DESKTOP?: boolean
  WAVENODE_DESKTOP_BRIDGE?: {
    getServerUrl: () => Promise<string>
    setServerUrl: (url: string) => Promise<string>
    discoverServers: () => Promise<Array<{ name: string; url: string }>>
  }
}
