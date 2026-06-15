import { API_BASE_URL } from '../services/api'
import type { Album, Music } from '../services/api'

export const resolveMediaUrl = (value?: string): string | undefined => {
  const trimmed = value?.trim()
  if (!trimmed || trimmed.toLowerCase().includes('default-track.png')) {
    return undefined
  }

  try {
    const apiBase = new URL(API_BASE_URL, window.location.origin)

    if (trimmed.startsWith('/artwork/')) {
      return new URL(`/api${trimmed}`, apiBase.origin).toString()
    }

    if (trimmed.startsWith('/api/')) {
      return new URL(trimmed, apiBase.origin).toString()
    }

    return new URL(trimmed, `${apiBase.toString().replace(/\/+$/, '')}/`).toString()
  } catch {
    return undefined
  }
}

export const getTrackArtworkUrl = (track?: Partial<Music> | null): string | undefined =>
  resolveMediaUrl(
    track?.image_url ||
    track?.cover_art_large_url ||
    track?.cover_art_medium_url ||
    track?.cover_art_small_url ||
    track?.cover_art_url,
  )

export const getAlbumArtworkUrl = (
  album?: Partial<Album>,
  tracks: Array<Partial<Music>> = [],
): string | undefined => {
  const trackArtwork = tracks
    .filter((track) => !album?.name || track.album === album.name)
    .map(getTrackArtworkUrl)
    .find(Boolean)

  return trackArtwork || resolveMediaUrl(
    album?.cover_art_large_url ||
    album?.cover_art_medium_url ||
    album?.cover_art_small_url ||
    album?.cover_art_url,
  )
}

const artworkGradients = [
  ['#164e63', '#0891b2'],
  ['#4c1d95', '#8b5cf6'],
  ['#7c2d12', '#ea580c'],
  ['#14532d', '#16a34a'],
  ['#831843', '#db2777'],
  ['#1e3a8a', '#2563eb'],
  ['#3f3f46', '#71717a'],
  ['#713f12', '#ca8a04'],
]

export const getArtworkGradient = (seed: string): string => {
  let hash = 0
  for (let index = 0; index < seed.length; index += 1) {
    hash = ((hash << 5) - hash + seed.charCodeAt(index)) | 0
  }

  const [start, end] = artworkGradients[Math.abs(hash) % artworkGradients.length]
  return `linear-gradient(135deg, ${start}, ${end})`
}
