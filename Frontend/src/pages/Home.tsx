import React, { useState, useEffect } from 'react'
import styled from 'styled-components'
import { Play, Music as MusicIcon, Clock, Radio } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useAudio } from '../contexts/AudioContext'
import { pluginsAPI, recentlyPlayedAPI, type Music, type PluginHomeRow, type PluginRowItem } from '../services/api'
import { getArtworkGradient, getTrackArtworkUrl } from '../utils/mediaUrl'
import { TrackActionsMenu } from '../components/TrackActionsMenu'

const HomeContainer = styled.div`
  padding: 24px;
  overflow-y: auto;
  
  @media (max-width: 768px) {
    padding: 16px;
    padding-top: 80px; // Account for mobile menu button
  }
`

const Section = styled.section`
  margin-bottom: 40px;
  
  @media (max-width: 768px) {
    margin-bottom: 32px;
  }
`

const SectionTitle = styled.h2`
  color: #fff;
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 24px;
  display: flex;
  align-items: center;
  gap: 12px;
  
  @media (max-width: 768px) {
    font-size: 20px;
    margin-bottom: 16px;
  }
`

const TrackGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
  
  @media (max-width: 768px) {
    grid-template-columns: 1fr;
    gap: 12px;
  }
`

const TrackCard = styled.div`
  background-color: #181818;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;

  &:hover {
    background-color: #282828;
  }

  &:hover .play-button {
    opacity: 1;
  }
  
  @media (max-width: 768px) {
    padding: 12px;
    gap: 10px;
  }
`

const TrackCoverArt = styled.div<{ $imageUrl?: string; $fallback: string }>`
  width: 60px;
  height: 60px;
  background: ${props => props.$imageUrl ? `url("${props.$imageUrl}")` : props.$fallback};
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 20px;
  flex-shrink: 0;
  position: relative;
  
  @media (max-width: 768px) {
    width: 50px;
    height: 50px;
    font-size: 18px;
  }
`

const PlayButton = styled.div`
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  background-color: #1db954;
  border-radius: 50%;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #000;
  opacity: 0;
  transition: all 0.2s ease;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);

  &:hover {
    background-color: #1ed760;
    transform: translate(-50%, -50%) scale(1.05);
  }

  &.play-button {
    opacity: 0;
  }
`

const TrackInfo = styled.div`
  flex: 1;
  min-width: 0;
`

const TrackName = styled.div`
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 14px;
  }
`

const TrackArtist = styled.div`
  color: #b3b3b3;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 12px;
  }
`

const TrackAlbum = styled.div`
  color: #b3b3b3;
  font-size: 12px;
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 11px;
  }
`

const EmptyState = styled.div`
  grid-column: 1 / -1;
  text-align: center;
  color: #b3b3b3;
  padding: 60px 20px;
  
  @media (max-width: 768px) {
    padding: 40px 16px;
  }
`

const EmptyStateIcon = styled.div`
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
  
  @media (max-width: 768px) {
    font-size: 40px;
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

const LoadingState = styled.div`
  grid-column: 1 / -1;
  text-align: center;
  color: #b3b3b3;
  padding: 40px;
  font-size: 16px;
`

const PluginRows = styled.div`
  display: grid;
  gap: 36px;
`

const RadioGrid = styled.div`
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(220px, 280px);
  gap: 16px;
  padding-bottom: 8px;
  overflow-x: auto;
  overscroll-behavior-inline: contain;
`

const RadioCard = styled.button`
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  align-items: center;
  gap: 14px;
  padding: 14px;
  border-radius: 10px;
  color: #fff;
  background: #181818;
  text-align: left;
  transition: background .2s ease, transform .2s ease;

  &:hover {
    background: #282828;
    transform: translateY(-2px);
  }
`

const RadioArtwork = styled.div<{ $imageUrl?: string; $fallback: string }>`
  width: 64px;
  height: 64px;
  display: grid;
  place-items: center;
  overflow: hidden;
  border-radius: 8px;
  background: ${props => props.$imageUrl ? `url("${props.$imageUrl}")` : props.$fallback};
  background-size: cover;
  background-position: center;
  color: #fff;
`

const RadioDetails = styled.div`
  min-width: 0;

  strong,
  span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    margin-bottom: 5px;
  }

  span {
    color: #aaa;
    font-size: 13px;
  }
`

const pluginItemToTrack = (pluginID: string, item: PluginRowItem): Music => ({
  id: `plugin:${pluginID}:${item.id}`,
  title: item.title,
  artist: item.subtitle || 'Internet radio',
  album: 'Live radio',
  genre: 'Radio',
  duration: 0,
  release_date: '',
  file_path: '',
  image_url: item.image_url,
  created_at: '',
  updated_at: '',
  stream_url: item.stream_url,
  is_external: true,
})

export const Home: React.FC = () => {
  const { isAuthenticated } = useAuth()
  const { playTrack } = useAudio()
  const [recentlyPlayed, setRecentlyPlayed] = useState<Music[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pluginRows, setPluginRows] = useState<PluginHomeRow[]>([])
  const [pluginError, setPluginError] = useState<string | null>(null)
  const [trackLimit, setTrackLimit] = useState(10) // Default to desktop

  // Update track limit based on screen size
  useEffect(() => {
    const updateTrackLimit = () => {
      setTrackLimit(window.innerWidth <= 768 ? 6 : 10)
    }

    // Set initial limit
    updateTrackLimit()

    // Listen for window resize
    window.addEventListener('resize', updateTrackLimit)
    
    return () => {
      window.removeEventListener('resize', updateTrackLimit)
    }
  }, [])

  // Fetch recently played tracks from API
  useEffect(() => {
    const fetchRecentlyPlayed = async () => {
      if (!isAuthenticated) {
        setLoading(false)
        return
      }

      try {
        setLoading(true)
        setError(null)
        const tracks = await recentlyPlayedAPI.getRecentlyPlayed()
        setRecentlyPlayed(tracks)
      } catch (err) {
        console.error('Failed to fetch recently played tracks:', err)
        setError('Failed to load recently played tracks')
      } finally {
        setLoading(false)
      }
    }

    fetchRecentlyPlayed()
  }, [isAuthenticated])

  useEffect(() => {
    if (!isAuthenticated) {
      setPluginRows([])
      return
    }

    let active = true
    void pluginsAPI.getHomeRows()
      .then(rows => {
        if (active) {
          setPluginRows(rows)
          setPluginError(null)
        }
      })
      .catch(err => {
        console.error('Failed to load plugin home rows:', err)
        if (active) {
          setPluginError('Plugin content could not be loaded')
        }
      })
    return () => {
      active = false
    }
  }, [isAuthenticated])

  const handlePlayTrack = async (track: Music) => {
    try {
      await playTrack(track)
    } catch (err) {
      console.error('Failed to play track:', err)
    }
  }

  if (!isAuthenticated) {
    return (
      <HomeContainer>
        <Section>
          <SectionTitle>
            <Clock size={24} />
            Recently played
          </SectionTitle>
          <EmptyState>
            <EmptyStateIcon>
              <Clock size={48} />
            </EmptyStateIcon>
            <EmptyStateText>Please log in to see your recently played tracks</EmptyStateText>
            <EmptyStateSubtext>Sign in to view your listening history</EmptyStateSubtext>
          </EmptyState>
        </Section>
      </HomeContainer>
    )
  }

  return (
    <HomeContainer>
      <Section>
        <SectionTitle>
          <Clock size={24} />
          Recently played
        </SectionTitle>
        <TrackGrid>
          {loading ? (
            <LoadingState>Loading recently played tracks...</LoadingState>
          ) : error ? (
            <EmptyState>
              <EmptyStateText>Error loading recently played tracks</EmptyStateText>
              <EmptyStateSubtext>{error}</EmptyStateSubtext>
            </EmptyState>
          ) : recentlyPlayed.length === 0 ? (
            <EmptyState>
              <EmptyStateIcon>
                <Clock size={48} />
              </EmptyStateIcon>
              <EmptyStateText>No recently played tracks yet</EmptyStateText>
              <EmptyStateSubtext>Start playing music to see your listening history here</EmptyStateSubtext>
            </EmptyState>
          ) : (
            recentlyPlayed.slice(0, trackLimit).map((track) => {
              const artworkUrl = getTrackArtworkUrl(track)

              return (
              <TrackCard 
                key={track.id}
                onClick={() => handlePlayTrack(track)}
              >
                <TrackCoverArt
                  $imageUrl={artworkUrl}
                  $fallback={getArtworkGradient(`${track.album}|${track.artist}`)}
                >
                  {artworkUrl ? null : <MusicIcon size={24} />}
                  <PlayButton className="play-button">
                    <Play size={16} />
                  </PlayButton>
                </TrackCoverArt>
                <TrackInfo>
                  <TrackName>{track.title}</TrackName>
                  <TrackArtist>{track.artist}</TrackArtist>
                  <TrackAlbum>{track.album}</TrackAlbum>
                </TrackInfo>
                <TrackActionsMenu track={track} />
              </TrackCard>
              )
            })
          )}
        </TrackGrid>
      </Section>

      {(pluginRows.length > 0 || pluginError) && (
        <PluginRows>
          {pluginError && (
            <Section>
              <SectionTitle><Radio size={24} /> Plugins</SectionTitle>
              <EmptyState>
                <EmptyStateText>{pluginError}</EmptyStateText>
              </EmptyState>
            </Section>
          )}
          {pluginRows.map(row => (
            <Section key={`${row.plugin_id}:${row.id}`}>
              <SectionTitle><Radio size={24} /> {row.title}</SectionTitle>
              {row.subtitle && <EmptyStateSubtext style={{ margin: '-16px 0 18px' }}>{row.subtitle}</EmptyStateSubtext>}
              <RadioGrid>
                {row.items.map(item => (
                  <RadioCard
                    key={item.id}
                    type="button"
                    title={item.description || `Play ${item.title}`}
                    onClick={() => void handlePlayTrack(pluginItemToTrack(row.plugin_id, item))}
                  >
                    <RadioArtwork
                      $imageUrl={item.image_url}
                      $fallback={getArtworkGradient(`${row.plugin_id}:${item.id}`)}
                    >
                      {item.image_url ? null : <Radio size={26} />}
                    </RadioArtwork>
                    <RadioDetails>
                      <strong>{item.title}</strong>
                      <span>{item.subtitle || 'Live radio'}</span>
                    </RadioDetails>
                  </RadioCard>
                ))}
              </RadioGrid>
            </Section>
          ))}
        </PluginRows>
      )}

      <Section>
        <SectionTitle>Made for you</SectionTitle>
        <TrackGrid>
          <EmptyState>
            <EmptyStateIcon>
              <MusicIcon size={48} />
            </EmptyStateIcon>
            <EmptyStateText>No personalized playlists available</EmptyStateText>
            <EmptyStateSubtext>Listen to more music to get personalized recommendations</EmptyStateSubtext>
          </EmptyState>
        </TrackGrid>
      </Section>
    </HomeContainer>
  )
}
