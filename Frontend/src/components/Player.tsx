import React, { useEffect, useRef, useState } from 'react'
import styled from 'styled-components'
import { useAudio } from '../contexts/AudioContext'
import { Queue } from './Queue'
import { likedTracksAPI, ratingsAPI } from '../services/api'
import { getTrackArtworkUrl } from '../utils/mediaUrl'
import { 
  Play, 
  Pause,
  SkipBack, 
  SkipForward, 
  Volume2, 
  VolumeX,
  Repeat, 
  Shuffle, 
  Heart,
  Star,
  Maximize2,
  Mic2,
  List,
  Monitor,
  MoreHorizontal
} from 'lucide-react'

const PlayerContainer = styled.footer`
  height: 90px;
  background-color: #181818;
  border-top: 1px solid #282828;
  display: flex;
  align-items: center;
  padding: 0 16px;
  gap: 16px;
  position: relative;
  
  @media (max-width: 768px) {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: 1000;
    height: auto;
    padding: 8px 16px;
    flex-direction: column;
    gap: 8px;
  }
`

const TopRow = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  
  @media (min-width: 769px) {
    display: none;
  }
`

const MobileTrackInfo = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
`

const MobileAlbumArt = styled.div`
  width: 40px;
  height: 40px;
  background-color: #282828;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #b3b3b3;
  font-size: 10px;
  flex-shrink: 0;
  overflow: hidden;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
`

const MobileTrackDetails = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
`

const MobileTrackName = styled.div`
  color: #fff;
  font-size: 12px;
  font-weight: 400;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`

const MobileArtistName = styled.div`
  color: #b3b3b3;
  font-size: 10px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`

const MobileMoreButton = styled.button`
  background: none;
  border: none;
  color: #b3b3b3;
  cursor: pointer;
  padding: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  
  &:hover {
    color: #fff;
  }
  
  svg {
    width: 20px;
    height: 20px;
  }
`

const DesktopLayout = styled.div`
  display: flex;
  align-items: center;
  width: 100%;
  gap: 16px;
  
  @media (max-width: 768px) {
    display: none;
  }
`

const TrackInfo = styled.div`
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 180px;
  flex: 1;
`

const AlbumArt = styled.div`
  width: 56px;
  height: 56px;
  background-color: #282828;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #b3b3b3;
  font-size: 12px;
  overflow: hidden;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
`

const TrackDetails = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  flex: 1;
`

const TrackName = styled.div`
  color: #fff;
  font-size: 14px;
  font-weight: 400;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`

const ArtistName = styled.div`
  color: #b3b3b3;
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`

const PlayerControls = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  flex: 2;
  
  @media (min-width: 769px) {
    display: none;
  }
  
  @media (max-width: 768px) {
    order: -1;
    flex: none;
    gap: 4px;
  }
`

const ControlButtons = styled.div`
  display: flex;
  align-items: center;
  gap: 16px;
  
  @media (max-width: 768px) {
    gap: 12px;
  }
`

const ControlButton = styled.button`
  background: none;
  border: none;
  color: #b3b3b3;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    color: #fff;
    transform: scale(1.1);
  }

  &:disabled {
    color: #5a5a5a;
    cursor: not-allowed;
    transform: none;
  }

  &.play-button {
    width: 32px;
    height: 32px;
    background-color: #fff;
    border-radius: 50%;
    color: #000;

    &:hover {
      transform: scale(1.05);
      background-color: #f0f0f0;
    }
    
    @media (max-width: 768px) {
      width: 40px;
      height: 40px;
    }
  }

  svg {
    width: 16px;
    height: 16px;

    &.large {
      width: 20px;
      height: 20px;
    }
    
    @media (max-width: 768px) {
      width: 20px;
      height: 20px;
      
      &.large {
        width: 24px;
        height: 24px;
      }
    }
  }
  
  @media (max-width: 768px) {
    &.hide-on-mobile {
      display: none;
    }
  }
`

const ProgressBar = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  max-width: 600px;
  
  @media (max-width: 768px) {
    max-width: none;
    gap: 4px;
  }
`

const Time = styled.span`
  color: #b3b3b3;
  font-size: 11px;
  min-width: 40px;
  text-align: center;
  
  @media (max-width: 768px) {
    font-size: 10px;
    min-width: 35px;
  }
`

const ProgressTrack = styled.div`
  flex: 1;
  height: 4px;
  background-color: #535353;
  border-radius: 2px;
  position: relative;
  cursor: pointer;

  &:hover {
    background-color: #6a6a6a;
  }
  
  @media (max-width: 768px) {
    height: 3px;
  }
`

const ProgressFill = styled.div`
  height: 100%;
  background-color: #fff;
  border-radius: 2px;
  width: 30%;
  position: relative;
`

const ProgressHandle = styled.div`
  position: absolute;
  right: -6px;
  top: -4px;
  width: 12px;
  height: 12px;
  background-color: #fff;
  border-radius: 50%;
  opacity: 0;
  transition: opacity 0.2s ease;
  
  @media (max-width: 768px) {
    display: none;
  }
`

const DesktopPlayerControls = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  flex: 2;
  
  @media (max-width: 768px) {
    display: none;
  }
`

const ExtraControls = styled.div`
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex: 1;
  min-width: 180px;
  
  @media (max-width: 768px) {
    display: none;
  }
`

const VolumeControl = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`

const VolumeSlider = styled.input`
  appearance: none;
  -webkit-appearance: none;
  width: 100px;
  height: 4px;
  background: linear-gradient(
    to right,
    #fff 0%,
    #fff var(--volume-percent),
    #535353 var(--volume-percent),
    #535353 100%
  );
  border-radius: 2px;
  cursor: pointer;
  padding: 0;

  &:hover {
    background: linear-gradient(
      to right,
      #1ed760 0%,
      #1ed760 var(--volume-percent),
      #6a6a6a var(--volume-percent),
      #6a6a6a 100%
    );
  }

  &::-webkit-slider-runnable-track {
    height: 4px;
    border-radius: 2px;
  }

  &::-webkit-slider-thumb {
    appearance: none;
    -webkit-appearance: none;
    width: 12px;
    height: 12px;
    margin-top: -4px;
    border: 0;
    border-radius: 50%;
    background-color: #fff;
    opacity: 0;
  }

  &:hover::-webkit-slider-thumb,
  &:focus-visible::-webkit-slider-thumb {
    opacity: 1;
  }

  &::-moz-range-track {
    height: 4px;
    border-radius: 2px;
    background: #535353;
  }

  &::-moz-range-progress {
    height: 4px;
    border-radius: 2px;
    background: #fff;
  }

  &::-moz-range-thumb {
    width: 12px;
    height: 12px;
    border: 0;
    border-radius: 50%;
    background-color: #fff;
  }
`

const IconButton = styled.button`
  background: none;
  border: none;
  color: #b3b3b3;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    color: #fff;
    transform: scale(1.1);
  }

  &.active {
    color: #1db954;
  }

  &:disabled {
    color: #535353;
    cursor: not-allowed;
    transform: none;
  }

  svg {
    width: 16px;
    height: 16px;
  }
`

const RatingControls = styled.div`
  display: flex;
  align-items: center;
  gap: 1px;

  button {
    padding: 2px;
  }
`

export const Player: React.FC = () => {
  const [isQueueOpen, setIsQueueOpen] = useState(false);
  const [isLiked, setIsLiked] = useState(false);
  const [rating, setRating] = useState(0);
  const previousVolumeRef = useRef(1);
  
  const { 
    currentTrack, 
    isPlaying, 
    currentTime, 
    duration, 
    volume, 
    state, 
    togglePlayPause, 
    nextTrack, 
    previousTrack, 
    setVolume, 
    seekTo, 
    toggleShuffle, 
    setRepeatMode 
  } = useAudio();

  const artworkUrl = getTrackArtworkUrl(currentTrack);

  useEffect(() => {
    let isCurrent = true;

    const loadLikedState = async () => {
      if (!currentTrack) {
        setIsLiked(false);
        setRating(0);
        return;
      }
      if (currentTrack.is_external) {
        setIsLiked(false);
        setRating(0);
        return;
      }

      try {
        const [liked, savedRating] = await Promise.all([
          likedTracksAPI.isTrackLiked(currentTrack.id),
          ratingsAPI.getRating(currentTrack.id),
        ]);
        if (isCurrent) {
          setIsLiked(liked);
          setRating(savedRating);
        }
      } catch {
        if (isCurrent) {
          setIsLiked(false);
          setRating(0);
        }
      }
    };

    void loadLikedState();
    return () => {
      isCurrent = false;
    };
  }, [currentTrack]);

  const formatTime = (seconds: number): string => {
    if (!Number.isFinite(seconds)) return 'LIVE';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const handleProgressClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!Number.isFinite(duration) || duration <= 0) {
      return;
    }
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const percentage = x / rect.width;
    const newTime = percentage * duration;
    seekTo(newTime);
  };

  const handleToggleMute = () => {
    if (volume > 0) {
      previousVolumeRef.current = volume;
      setVolume(0);
    } else {
      setVolume(previousVolumeRef.current || 1);
    }
  };

  const handleVolumePointerDown = (event: React.PointerEvent<HTMLInputElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    if (rect.width <= 0) {
      return;
    }
    const percentage = (event.clientX - rect.left) / rect.width;
    setVolume(Math.max(0, Math.min(1, percentage)));
  };

  const handleToggleLike = async () => {
    if (!currentTrack || currentTrack.is_external) {
      return;
    }

    const nextLiked = !isLiked;
    setIsLiked(nextLiked);
    try {
      if (nextLiked) {
        await likedTracksAPI.likeTrack(currentTrack.id);
      } else {
        await likedTracksAPI.unlikeTrack(currentTrack.id);
      }
    } catch {
      setIsLiked(!nextLiked);
    }
  };

  const handleToggleFullscreen = async () => {
    if (document.fullscreenElement) {
      await document.exitFullscreen();
    } else {
      await document.documentElement.requestFullscreen();
    }
  };

  const handleSetRating = async (nextRating: number) => {
    if (!currentTrack || currentTrack.is_external) {
      return;
    }
    const previousRating = rating;
    const value = nextRating === rating ? 0 : nextRating;
    setRating(value);
    try {
      setRating(await ratingsAPI.setRating(currentTrack.id, value));
    } catch {
      setRating(previousRating);
    }
  };

  const progressPercentage = Number.isFinite(duration) && duration > 0 ? (currentTime / duration) * 100 : 0;
  const volumePercentage = volume * 100;

  return (
    <PlayerContainer>
      {/* Mobile Top Row */}
      <TopRow>
        <MobileTrackInfo>
          <MobileAlbumArt>
            {artworkUrl ? (
              <img
                src={artworkUrl}
                alt={currentTrack?.title || 'Current track'}
              />
            ) : (
              <Monitor size={16} />
            )}
          </MobileAlbumArt>
          <MobileTrackDetails>
            <MobileTrackName>
              {currentTrack ? currentTrack.title : 'No track playing'}
            </MobileTrackName>
            <MobileArtistName>
              {currentTrack ? currentTrack.artist : 'Select a track to play'}
            </MobileArtistName>
          </MobileTrackDetails>
        </MobileTrackInfo>
        <MobileMoreButton onClick={() => setIsQueueOpen(true)} title="Open queue">
          <MoreHorizontal size={20} />
        </MobileMoreButton>
      </TopRow>

      {/* Player Controls - Centered on mobile */}
      <PlayerControls>
        <ControlButtons>
          <IconButton 
            className={`hide-on-mobile ${state.isShuffled ? 'active' : ''}`}
            onClick={toggleShuffle}
            title={state.isShuffled ? 'Disable shuffle' : 'Enable shuffle'}
          >
            <Shuffle size={16} />
          </IconButton>
          <ControlButton onClick={previousTrack} disabled={!currentTrack} title="Previous">
            <SkipBack size={20} className="large" />
          </ControlButton>
          <ControlButton className="play-button" onClick={togglePlayPause} disabled={!currentTrack} title={isPlaying ? 'Pause' : 'Play'}>
            {isPlaying ? <Pause size={16} /> : <Play size={16} />}
          </ControlButton>
          <ControlButton onClick={nextTrack} disabled={!currentTrack} title="Next">
            <SkipForward size={20} className="large" />
          </ControlButton>
          <IconButton 
            className={`hide-on-mobile ${state.repeatMode !== 'none' ? 'active' : ''}`}
            onClick={() => setRepeatMode(
              state.repeatMode === 'none' ? 'all' :
              state.repeatMode === 'all' ? 'one' : 'none'
            )}
            title={`Repeat: ${state.repeatMode}`}
          >
            <Repeat size={16} />
          </IconButton>
        </ControlButtons>
        <ProgressBar>
          <Time>{formatTime(currentTime)}</Time>
          <ProgressTrack onClick={handleProgressClick}>
            <ProgressFill style={{ width: `${progressPercentage}%` }}>
              <ProgressHandle />
            </ProgressFill>
          </ProgressTrack>
          <Time>{formatTime(duration)}</Time>
        </ProgressBar>
      </PlayerControls>

      {/* Desktop Layout */}
      <DesktopLayout>
        <TrackInfo>
          <AlbumArt>
            {artworkUrl ? (
              <img
                src={artworkUrl}
                alt={currentTrack?.title || 'Current track'}
              />
            ) : (
              <Monitor size={24} />
            )}
          </AlbumArt>
          <TrackDetails>
            <TrackName>
              {currentTrack ? currentTrack.title : 'No track playing'}
            </TrackName>
            <ArtistName>
              {currentTrack ? currentTrack.artist : 'Select a track to play'}
            </ArtistName>
          </TrackDetails>
          <IconButton
            className={isLiked ? 'active' : ''}
            onClick={handleToggleLike}
            disabled={!currentTrack || currentTrack.is_external}
            title={isLiked ? 'Remove from liked songs' : 'Save to liked songs'}
          >
            <Heart size={16} fill={isLiked ? 'currentColor' : 'none'} />
          </IconButton>
          <RatingControls aria-label={`Rating: ${rating || 'not rated'}`}>
            {[1, 2, 3, 4, 5].map(value => (
              <IconButton
                key={value}
                className={value <= rating ? 'active' : ''}
                onClick={() => void handleSetRating(value)}
                disabled={!currentTrack || currentTrack.is_external}
                title={value === rating ? 'Remove rating' : `Rate ${value} out of 5`}
              >
                <Star size={13} fill={value <= rating ? 'currentColor' : 'none'} />
              </IconButton>
            ))}
          </RatingControls>
        </TrackInfo>

        <DesktopPlayerControls>
          <ControlButtons>
            <IconButton 
              className={state.isShuffled ? 'active' : ''} 
              onClick={toggleShuffle}
              title={state.isShuffled ? 'Disable shuffle' : 'Enable shuffle'}
            >
              <Shuffle size={16} />
            </IconButton>
            <ControlButton onClick={previousTrack} disabled={!currentTrack} title="Previous">
              <SkipBack size={20} className="large" />
            </ControlButton>
            <ControlButton className="play-button" onClick={togglePlayPause} disabled={!currentTrack} title={isPlaying ? 'Pause' : 'Play'}>
              {isPlaying ? <Pause size={16} /> : <Play size={16} />}
            </ControlButton>
            <ControlButton onClick={nextTrack} disabled={!currentTrack} title="Next">
              <SkipForward size={20} className="large" />
            </ControlButton>
            <IconButton 
              className={state.repeatMode !== 'none' ? 'active' : ''} 
              onClick={() => setRepeatMode(
                state.repeatMode === 'none' ? 'all' : 
                state.repeatMode === 'all' ? 'one' : 'none'
              )}
              title={`Repeat: ${state.repeatMode}`}
            >
              <Repeat size={16} />
            </IconButton>
          </ControlButtons>
          <ProgressBar>
            <Time>{formatTime(currentTime)}</Time>
            <ProgressTrack onClick={handleProgressClick}>
              <ProgressFill style={{ width: `${progressPercentage}%` }}>
                <ProgressHandle />
              </ProgressFill>
            </ProgressTrack>
            <Time>{formatTime(duration)}</Time>
          </ProgressBar>
        </DesktopPlayerControls>

        <ExtraControls>
          <IconButton disabled title="Lyrics are not available yet">
            <Mic2 size={16} />
          </IconButton>
          <IconButton
            className={isQueueOpen ? 'active' : ''}
            onClick={() => setIsQueueOpen(!isQueueOpen)}
            title={isQueueOpen ? 'Close queue' : 'Open queue'}
          >
            <List size={16} />
          </IconButton>
          <IconButton onClick={handleToggleFullscreen} title="Toggle fullscreen">
            <Maximize2 size={16} />
          </IconButton>
          <VolumeControl>
            <IconButton onClick={handleToggleMute} title={volume > 0 ? 'Mute' : 'Unmute'}>
              {volume > 0 ? <Volume2 size={16} /> : <VolumeX size={16} />}
            </IconButton>
            <VolumeSlider
              type="range"
              min="0"
              max="1"
              step="0.01"
              value={volume}
              onPointerDown={handleVolumePointerDown}
              onChange={(event) => setVolume(Number(event.currentTarget.value))}
              aria-label="Volume"
              title={`Volume: ${Math.round(volumePercentage)}%`}
              style={{ '--volume-percent': `${volumePercentage}%` } as React.CSSProperties}
            />
          </VolumeControl>
        </ExtraControls>
      </DesktopLayout>
      
      <Queue isOpen={isQueueOpen} onClose={() => setIsQueueOpen(false)} />
    </PlayerContainer>
  )
}
