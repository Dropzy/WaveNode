import React, { useState, useEffect, useCallback } from 'react'
import styled from 'styled-components'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Play, Music, User, Shuffle } from 'lucide-react'
import { artistAPI, ArtistTracksResponse, Music as Track } from '../services/api'
import { useAuth } from '../contexts/AuthContext'
import { useAudio } from '../contexts/AudioContext'
import { getAlbumArtworkUrl, getArtworkGradient, getTrackArtworkUrl } from '../utils/mediaUrl'
import { TrackActionsMenu, TrackSelectionContextMenu } from '../components/TrackActionsMenu'
import { useTrackSelection } from '../hooks/useTrackSelection'

const ArtistContainer = styled.div`
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

const ArtistArt = styled.div<{ $imageUrl?: string; $fallback: string }>`
  width: 232px;
  height: 232px;
  background: ${props => props.$imageUrl ? `url("${props.$imageUrl}")` : props.$fallback};
  background-size: cover;
  background-position: center;
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

const ArtistInfo = styled.div`
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

const ArtistName = styled.h1`
  color: #fff;
  font-size: 48px;
  font-weight: 700;
  margin-bottom: 16px;
  
  @media (max-width: 768px) {
    font-size: 32px;
    margin-bottom: 12px;
  }
`

const ArtistStats = styled.div`
  display: flex;
  gap: 24px;
  margin-bottom: 24px;
  
  @media (max-width: 768px) {
    gap: 16px;
    margin-bottom: 16px;
  }
`

const Stat = styled.div`
  color: #b3b3b3;
  font-size: 14px;
  
  @media (max-width: 768px) {
    font-size: 12px;
  }
`

const StatValue = styled.div`
  color: #fff;
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 4px;
  
  @media (max-width: 768px) {
    font-size: 20px;
  }
`

const ArtistActions = styled.div`
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

const AlbumGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 24px;
  
  @media (max-width: 768px) {
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 16px;
  }
  
  @media (max-width: 480px) {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
`

const AlbumCard = styled.div`
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
  
  @media (max-width: 768px) {
    padding: 12px;
  }
`

const AlbumArt = styled.div<{ $imageUrl?: string; $fallback: string }>`
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
  
  @media (max-width: 768px) {
    font-size: 36px;
    margin-bottom: 12px;
  }
`

const AlbumName = styled.div`
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 13px;
  }
`

const AlbumYear = styled.div`
  color: #b3b3b3;
  font-size: 12px;
  
  @media (max-width: 768px) {
    font-size: 11px;
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

const formatDate = (dateString: string): string => {
  return new Date(dateString).getFullYear().toString()
}

export const Artist: React.FC = () => {
  const { artistId } = useParams<{ artistId: string }>()
  const { token } = useAuth()
  const { playTrack, playPlaylist, playPlaylistShuffled } = useAudio()
  const navigate = useNavigate()
  const [artistData, setArtistData] = useState<ArtistTracksResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [hoveredTrackIndex, setHoveredTrackIndex] = useState<number | null>(null)
  const [selectionMenu, setSelectionMenu] = useState<{ x: number; y: number } | null>(null)
  const trackSelection = useTrackSelection(artistData?.tracks || [])

  useEffect(() => {
    const fetchArtistData = async () => {
      if (!artistId || !token) {
        console.log('Missing artistId or token:', { artistId, token: !!token })
        setLoading(false)
        return
      }

      try {
        setLoading(true)
        console.log('Fetching artist data for ID:', artistId)
        const data = await artistAPI.getArtistTracksById(artistId)
        console.log('Artist data response:', data)
        if (data && data.artist && data.tracks) {
          setArtistData(data)
          setError(null)
        } else {
          console.error('Invalid artist data structure:', data)
          setError('Artist not found')
        }
      } catch (err) {
        console.error('Error fetching artist data:', err)
        setError('Failed to load artist data')
      } finally {
        setLoading(false)
      }
    }

    fetchArtistData()
  }, [artistId, token])

  const handleAlbumClick = (albumName: string) => {
    // Find the album from the artist data to get the correct ID
    const album = artistData?.albums.find(a => a.name === albumName)
    console.log('Album clicked:', albumName, 'Found album:', album)
    if (album) {
      console.log('Navigating to album with ID:', album.id)
      // Use the album's actual ID from the database
      navigate(`/album/${album.id}`)
    } else {
      console.error('Album not found:', albumName)
    }
  }

  const handlePlayTrack = useCallback((track: Track) => {
    playTrack(track)
  }, [playTrack])

  const handlePlayAll = useCallback(() => {
    if (artistData?.tracks && artistData.tracks.length > 0) {
      playPlaylist(artistData.tracks)
    }
  }, [artistData?.tracks, playPlaylist])

  const handleShuffleAll = useCallback(() => {
    if (artistData?.tracks && artistData.tracks.length > 0) {
      playPlaylistShuffled(artistData.tracks)
    }
  }, [artistData?.tracks, playPlaylistShuffled])

  const handleBackToLibrary = () => {
    navigate('/library')
  }

  if (loading) {
    return <LoadingMessage>Loading artist information...</LoadingMessage>
  }

  if (error || !artistData) {
    return (
      <ArtistContainer>
        <ErrorMessage>{error || 'Artist not found'}</ErrorMessage>
        <BackButton onClick={handleBackToLibrary}>
          <ArrowLeft size={16} />
          Back to Library
        </BackButton>
      </ArtistContainer>
    )
  }

  const artistArtworkUrl = artistData.tracks
    .map(getTrackArtworkUrl)
    .find(Boolean)

  return (
    <ArtistContainer>
      <Header>
        <ArtistArt
          $imageUrl={artistArtworkUrl}
          $fallback={getArtworkGradient(artistData.artist.name)}
        >
          {artistArtworkUrl ? null : <User size={96} />}
        </ArtistArt>
        
        <ArtistInfo>
          <BackButton onClick={handleBackToLibrary}>
            <ArrowLeft size={16} />
            Back to Library
          </BackButton>
          <ArtistName>{artistData.artist.name}</ArtistName>
          <ArtistStats>
            <Stat>
              <StatValue>{artistData.artist.track_count}</StatValue>
              <div>Tracks</div>
            </Stat>
            <Stat>
              <StatValue>{artistData.artist.album_count}</StatValue>
              <div>Albums</div>
            </Stat>
          </ArtistStats>
          <ArtistActions>
            <PlayButton onClick={handlePlayAll}>
              <Play size={16} />
              Play All
            </PlayButton>
            <ShuffleButton onClick={handleShuffleAll}>
              <Shuffle size={16} />
              Shuffle
            </ShuffleButton>
          </ArtistActions>
        </ArtistInfo>
      </Header>

      <Section>
        <SectionTitle>Popular Tracks</SectionTitle>
        <TrackList>
          {artistData.tracks.map((track, index) => (
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
                  {index + 1}
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
                  <span>{track.album}</span>
                  <span>•</span>
                  <span>{formatDate(track.release_date)}</span>
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
          ))}
        </TrackList>
      </Section>
      <TrackSelectionContextMenu
        tracks={trackSelection.selectedTracks}
        position={selectionMenu}
        onClose={() => setSelectionMenu(null)}
      />

      <Section>
        <SectionTitle>Albums</SectionTitle>
        <AlbumGrid>
          {artistData.albums.map((album) => (
            <AlbumCard 
              key={album.id}
              onClick={() => handleAlbumClick(album.name)}
            >
              <AlbumArt
                $imageUrl={getAlbumArtworkUrl(album, artistData.tracks)}
                $fallback={getArtworkGradient(`${album.name}|${album.artist}`)}
              >
                {!getAlbumArtworkUrl(album, artistData.tracks) && <Music size={48} />}
              </AlbumArt>
              <AlbumName>{album.name}</AlbumName>
              <AlbumYear>
                {album.year || artistData.tracks
                  .filter(track => track.album === album.name)
                  .map(track => formatDate(track.release_date))[0] || ''}
              </AlbumYear>
            </AlbumCard>
          ))}
        </AlbumGrid>
      </Section>
    </ArtistContainer>
  )
}
