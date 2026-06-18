import React, { useState, useEffect, useCallback } from 'react'
import styled from 'styled-components'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Play, Disc, Shuffle } from 'lucide-react'
import { albumAPI, AlbumTracksResponse, AlbumTracksFallbackResponse, SimilarAlbum, Music as Track } from '../services/api'
import { useAuth } from '../contexts/AuthContext'
import { useAudio } from '../contexts/AudioContext'
import { generateAlbumHash } from '../utils/albumUtils'
import { getAlbumArtworkUrl, getArtworkGradient } from '../utils/mediaUrl'
import { TrackActionsMenu, TrackSelectionContextMenu } from '../components/TrackActionsMenu'
import { useTrackSelection } from '../hooks/useTrackSelection'

const AlbumContainer = styled.div`
  padding: 24px;
  overflow-y: auto;
  
  @media (max-width: 768px) {
    padding: 16px;
    padding-top: 80px; // Account for mobile menu button
  }
`

const Header = styled.div`
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

const AlbumArt = styled.div<{ $imageUrl?: string; $fallback: string }>`
  width: 232px;
  height: 232px;
  background: ${props => props.$imageUrl ? `url("${props.$imageUrl}")` : props.$fallback};
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 96px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  
  @media (max-width: 768px) {
    width: 192px;
    height: 192px;
    font-size: 80px;
  }
`

const AlbumInfo = styled.div`
  flex: 1;
  padding-top: 20px;
  
  @media (max-width: 768px) {
    padding-top: 0;
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

const AlbumTitle = styled.h1`
  color: #fff;
  font-size: 48px;
  font-weight: 700;
  margin-bottom: 16px;
  
  @media (max-width: 768px) {
    font-size: 32px;
    margin-bottom: 12px;
  }
`

const AlbumMeta = styled.div`
  display: flex;
  gap: 24px;
  margin-bottom: 24px;
  
  @media (max-width: 768px) {
    gap: 16px;
    margin-bottom: 16px;
  }
`

const MetaItem = styled.div`
  color: #b3b3b3;
  font-size: 14px;
  
  @media (max-width: 768px) {
    font-size: 12px;
  }
`

const MetaValue = styled.div`
  color: #fff;
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 4px;
  
  @media (max-width: 768px) {
    font-size: 20px;
  }
`

const AlbumActions = styled.div`
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  
  @media (max-width: 768px) {
    justify-content: center;
    margin-bottom: 20px;
  }
`

const PlayButton = styled.button`
  background-color: #1db954;
  border: none;
  border-radius: 500px;
  color: #fff;
  padding: 12px 32px;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.2s ease;

  &:hover {
    background-color: #1ed760;
    transform: scale(1.05);
  }
  
  @media (max-width: 768px) {
    padding: 10px 24px;
    font-size: 13px;
  }
`

const ShuffleButton = styled.button`
  background-color: transparent;
  border: 1px solid #b3b3b3;
  border-radius: 500px;
  color: #b3b3b3;
  padding: 12px 24px;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.2s ease;

  &:hover {
    border-color: #1db954;
    color: #1db954;
    transform: scale(1.05);
  }
  
  @media (max-width: 768px) {
    padding: 10px 20px;
    font-size: 13px;
  }
`

const Section = styled.div`
  margin-bottom: 32px;
  
  @media (max-width: 768px) {
    margin-bottom: 24px;
  }
`

const SectionTitle = styled.h2`
  color: #fff;
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 16px;
  
  @media (max-width: 768px) {
    font-size: 20px;
    margin-bottom: 12px;
  }
`

const TrackList = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
  
  @media (max-width: 768px) {
    gap: 6px;
  }
`

const DiscHeading = styled.h3`
  margin: 20px 4px 8px;
  color: #c8cec9;
  font-size: 14px;
  text-transform: uppercase;
  letter-spacing: 1px;
`

const TrackItem = styled.div<{ $selected?: boolean }>`
  background-color: ${props => props.$selected ? '#3a3a3a' : '#181818'};
  border-radius: 8px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 16px;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    background-color: #282828;
  }
  
  @media (max-width: 768px) {
    padding: 10px 12px;
    gap: 12px;
  }
`

const TrackNumberContainer = styled.div`
  width: 30px;
  display: flex;
  justify-content: center;
  align-items: center;
  position: relative;
  
  @media (max-width: 768px) {
    width: 25px;
  }
`

const TrackNumber = styled.span<{ $hidden?: boolean }>`
  color: #b3b3b3;
  font-size: 14px;
  text-align: center;
  opacity: ${props => props.$hidden ? 0 : 1};
  transition: opacity 0.2s ease;
  min-width: 30px;
  
  @media (max-width: 768px) {
    font-size: 12px;
    min-width: 25px;
  }
`

const TrackPlayIcon = styled.div<{ $visible?: boolean }>`
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

const TrackInfo = styled.div`
  flex: 1;
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

const TrackMeta = styled.div`
  color: #b3b3b3;
  font-size: 12px;
  display: flex;
  gap: 8px;
  align-items: center;
  
  @media (max-width: 768px) {
    font-size: 11px;
    gap: 6px;
  }
`

const TrackDuration = styled.span`
  color: #b3b3b3;
  font-size: 14px;
  
  @media (max-width: 768px) {
    font-size: 12px;
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

const formatDuration = (seconds: number): string => {
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
}

const SimilarAlbumsContainer = styled.div`
  margin-top: 32px;
`

const SimilarAlbumsTitle = styled.h3`
  color: #b3b3b3;
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 16px;
`

const SimilarAlbumsList = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  
  @media (max-width: 768px) {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 12px;
  }
`

const SimilarAlbumItem = styled.div`
  background-color: #181818;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  
  &:hover {
    background-color: #282828;
    transform: scale(1.02);
  }
`

const SimilarAlbumName = styled.div`
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`

const SimilarAlbumArtist = styled.div`
  color: #b3b3b3;
  font-size: 12px;
  margin-bottom: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`

const SimilarAlbumYear = styled.div`
  color: #b3b3b3;
  font-size: 11px;
`

const SimilarAlbumTracks = styled.div`
  color: #1db954;
  font-size: 11px;
  margin-top: 4px;
`

export const Album: React.FC = () => {
  const { albumName } = useParams<{ albumName: string }>()
  const { token } = useAuth()
  const { playFromQueue, playPlaylist, playPlaylistShuffled } = useAudio()
  const navigate = useNavigate()
  const [albumData, setAlbumData] = useState<AlbumTracksResponse | null>(null)
  const [fallbackData, setFallbackData] = useState<AlbumTracksFallbackResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [hoveredTrackIndex, setHoveredTrackIndex] = useState<number | null>(null)
  const [selectionMenu, setSelectionMenu] = useState<{ x: number; y: number } | null>(null)
  const trackSelection = useTrackSelection(albumData?.tracks || [])

  useEffect(() => {
    const fetchAlbumData = async () => {
      if (!albumName || !token) {
        setLoading(false)
        return
      }

      try {
        setLoading(true)
        // albumName is now a hash, so we don't need to encode it
        const data = await albumAPI.getAlbumTracks(albumName)
        if (data) {
          // Check if it's a fallback response or normal response
          if ('success' in data && data.success && data.similarAlbums) {
            // This is a fallback response
            setFallbackData(data)
            setAlbumData(null)
            setError(data.message || 'Album not found')
          } else if ('album' in data && data.album) {
            // This is a normal response
            setAlbumData(data as AlbumTracksResponse)
            setFallbackData(null)
            setError(null)
          } else {
            setError('Album not found')
          }
        } else {
          setError('Album not found')
        }
      } catch (err) {
        setError('Failed to load album data')
        console.error('Error fetching album data:', err)
      } finally {
        setLoading(false)
      }
    }

    fetchAlbumData()
  }, [albumName, token])

  const handleArtistClick = (artistName: string) => {
    const artistId = albumData?.tracks.find(
      (track) => track.artist === artistName && track.artist_id,
    )?.artist_id
    navigate(`/artist/${encodeURIComponent(artistId || artistName)}`)
  }

  const handlePlayTrack = useCallback((track: Track) => {
    if (!albumData?.tracks?.length) {
      return
    }
    const index = albumData.tracks.findIndex(item => item.id === track.id)
    playFromQueue(albumData.tracks, index === -1 ? 0 : index)
  }, [albumData?.tracks, playFromQueue])

  const handlePlayAll = useCallback(() => {
    if (albumData?.tracks && albumData.tracks.length > 0) {
      playPlaylist(albumData.tracks)
    }
  }, [albumData?.tracks, playPlaylist])

  const handleShuffleAll = useCallback(() => {
    if (albumData?.tracks && albumData.tracks.length > 0) {
      playPlaylistShuffled(albumData.tracks)
    }
  }, [albumData?.tracks, playPlaylistShuffled])

  const handleSimilarAlbumClick = (similarAlbum: SimilarAlbum) => {
    // Generate hash for the album to navigate to it
    const albumHash = generateAlbumHash(similarAlbum.name, similarAlbum.artist)
    navigate(`/album/${albumHash}`)
  }

  const handleBackToLibrary = () => {
    navigate('/library')
  }

  if (loading) {
    return <LoadingMessage>Loading album information...</LoadingMessage>
  }

  // Show error with similar albums if we have fallback data
  if (error || !albumData) {
    return (
      <AlbumContainer>
        <ErrorMessage>{error || 'Album not found'}</ErrorMessage>
        
        {fallbackData && fallbackData.similarAlbums && (
          <SimilarAlbumsContainer>
            <SimilarAlbumsTitle>Similar albums you might like:</SimilarAlbumsTitle>
            <SimilarAlbumsList>
              {fallbackData.similarAlbums.map((similarAlbum, index) => (
                <SimilarAlbumItem 
                  key={index} 
                  onClick={() => handleSimilarAlbumClick(similarAlbum)}
                >
                  <SimilarAlbumName>{similarAlbum.name}</SimilarAlbumName>
                  <SimilarAlbumArtist>{similarAlbum.artist}</SimilarAlbumArtist>
                  <SimilarAlbumYear>{similarAlbum.year}</SimilarAlbumYear>
                  <SimilarAlbumTracks>{similarAlbum.track_count} tracks</SimilarAlbumTracks>
                </SimilarAlbumItem>
              ))}
            </SimilarAlbumsList>
          </SimilarAlbumsContainer>
        )}
        
        <BackButton onClick={handleBackToLibrary}>
          <ArrowLeft size={16} />
          Back to Library
        </BackButton>
      </AlbumContainer>
    )
  }

  const artworkUrl = getAlbumArtworkUrl(albumData.album, albumData.tracks)

  return (
    <AlbumContainer>
      <Header>
        <AlbumArt
          $imageUrl={artworkUrl}
          $fallback={getArtworkGradient(`${albumData.album.name}|${albumData.album.artist}`)}
        >
          {!artworkUrl && <Disc size={96} />}
        </AlbumArt>
        
        <AlbumInfo>
          <BackButton onClick={handleBackToLibrary}>
            <ArrowLeft size={16} />
            Back to Library
          </BackButton>
          <AlbumTitle>{albumData.album.name}</AlbumTitle>
          <AlbumMeta>
            <MetaItem>
              <MetaValue>{albumData.tracks.length}</MetaValue>
              <div>Tracks</div>
            </MetaItem>
            <MetaItem>
              <MetaValue>{albumData.album.artist}</MetaValue>
              <div>Artist</div>
            </MetaItem>
            <MetaItem>
              <MetaValue>{albumData.album.year}</MetaValue>
              <div>Release Year</div>
            </MetaItem>
          </AlbumMeta>
          <AlbumActions>
            <PlayButton onClick={handlePlayAll}>
              <Play size={16} />
              Play All
            </PlayButton>
            <ShuffleButton onClick={handleShuffleAll}>
              <Shuffle size={16} />
              Shuffle
            </ShuffleButton>
          </AlbumActions>
        </AlbumInfo>
      </Header>

      <Section>
        <SectionTitle>Tracks</SectionTitle>
        <TrackList>
          {albumData.tracks.map((track, index) => (
            <React.Fragment key={track.id}>
            {(albumData.tracks.some(item => (item.disc_total || item.disc_number || 1) > 1)) &&
              (index === 0 || (albumData.tracks[index - 1].disc_number || 1) !== (track.disc_number || 1)) && (
                <DiscHeading>Disc {track.disc_number || 1}{track.disc_total && track.disc_total > 1 ? ` of ${track.disc_total}` : ''}</DiscHeading>
              )}
            <TrackItem 
              key={track.id}
              ref={element => { trackSelection.rowRefs.current[index] = element }}
              role="option"
              tabIndex={0}
              aria-selected={trackSelection.selectedIds.has(track.id)}
              $selected={trackSelection.selectedIds.has(track.id)}
              onClick={event => trackSelection.selectIndex(index, event)}
              onDoubleClick={() => handlePlayTrack(track)}
              onKeyDown={event => trackSelection.handleKeyDown(index, event, () => handlePlayTrack(track))}
              onContextMenu={event => {
                event.preventDefault()
                trackSelection.ensureSelected(index)
                setSelectionMenu({ x: event.clientX, y: event.clientY })
              }}
              onMouseEnter={() => setHoveredTrackIndex(index)}
              onMouseLeave={() => setHoveredTrackIndex(null)}
            >
              <TrackNumberContainer>
                <TrackNumber $hidden={hoveredTrackIndex === index}>
                  {track.track_number || index + 1}
                </TrackNumber>
                <TrackPlayIcon 
                  $visible={hoveredTrackIndex === index}
                  onClick={(e) => {
                    e.stopPropagation()
                    handlePlayTrack(track)
                  }}
                >
                  <Play size={16} />
                </TrackPlayIcon>
              </TrackNumberContainer>
              <TrackInfo>
                <TrackName>{track.title}</TrackName>
                <TrackMeta>
                  <span 
                    onClick={(e) => {
                      e.stopPropagation()
                      handleArtistClick(track.artist)
                    }}
                    style={{ cursor: 'pointer', textDecoration: 'underline' }}
                  >
                    {track.artist}
                  </span>
                  <span>•</span>
                  <span>{track.genre}</span>
                </TrackMeta>
              </TrackInfo>
              <TrackDuration>{formatDuration(track.duration)}</TrackDuration>
              <TrackActionsMenu
                track={track}
                tracks={trackSelection.selectedIds.has(track.id) ? trackSelection.selectedTracks : []}
              />
            </TrackItem>
            </React.Fragment>
          ))}
        </TrackList>
      </Section>
      <TrackSelectionContextMenu
        tracks={trackSelection.selectedTracks}
        position={selectionMenu}
        onClose={() => setSelectionMenu(null)}
      />
    </AlbumContainer>
  )
}
