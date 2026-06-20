import React, { useEffect, useRef, useState } from 'react'
import styled from 'styled-components'
import { useAudio } from '../contexts/AudioContext'
import { Queue } from './Queue'
import { accountAPI, likedTracksAPI, pluginsAPI, ratingsAPI, UserSession } from '../services/api'
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
  Smartphone,
  Cast,
  MoreHorizontal
} from 'lucide-react'

const PlayerContainer = styled.footer`
  height: 104px;
  background: ${({ theme }) => theme.colors.playerBg};
  border-top: 1px solid ${({ theme }) => theme.colors.border};
  box-shadow: 0 -18px 55px ${({ theme }) => theme.colors.shadow}, 0 -1px 24px ${({ theme }) => theme.colors.playerGlow};
  backdrop-filter: blur(18px);
  display: flex;
  align-items: center;
  padding: 12px 18px;
  gap: 18px;
  position: relative;
  
  @media (max-width: 768px) {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: 1000;
    height: auto;
    padding: 10px 14px;
    flex-direction: column;
    gap: 10px;
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
  padding: 6px;
  border-radius: 18px;
  background: ${({ theme }) => theme.colors.controlBg};
`

const MobileAlbumArt = styled.div`
  width: 44px;
  height: 44px;
  background: ${({ theme }) => theme.colors.surfaceStrong};
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: ${({ theme }) => theme.colors.muted};
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
  color: ${({ theme }) => theme.colors.text};
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`

const MobileArtistName = styled.div`
  color: ${({ theme }) => theme.colors.muted};
  font-size: 10px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`

const MobileMoreButton = styled.button`
  background: ${({ theme }) => theme.colors.controlBg};
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 14px;
  color: ${({ theme }) => theme.colors.muted};
  cursor: pointer;
  padding: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  
  &:hover {
    color: ${({ theme }) => theme.colors.text};
    border-color: ${({ theme }) => theme.colors.borderStrong};
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
  gap: 18px;
  
  @media (max-width: 768px) {
    display: none;
  }
`

const TrackInfo = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 260px;
  max-width: 460px;
  flex: 1.15;
  padding: 10px 12px;
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 24px;
  background: ${({ theme }) => theme.colors.surfaceSoft};
`

const AlbumArt = styled.div`
  width: 64px;
  height: 64px;
  background: ${({ theme }) => theme.colors.surfaceStrong};
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: ${({ theme }) => theme.colors.muted};
  font-size: 12px;
  overflow: hidden;
  box-shadow: 0 14px 32px ${({ theme }) => theme.colors.shadow};

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
`

const TrackDetails = styled.div`
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
  flex: 1;
`

const NowPlayingLabel = styled.div`
  color: ${({ theme }) => theme.colors.accent};
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
`

const TrackName = styled.div`
  color: ${({ theme }) => theme.colors.text};
  font-size: 14px;
  font-weight: 800;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`

const ArtistName = styled.div`
  color: ${({ theme }) => theme.colors.muted};
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
  gap: 10px;
  padding: 6px 10px;
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 999px;
  background: ${({ theme }) => theme.colors.controlBg};
  
  @media (max-width: 768px) {
    gap: 12px;
  }
`

const ControlButton = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.muted};
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    color: ${({ theme }) => theme.colors.text};
    transform: scale(1.1);
  }

  &:disabled {
    color: ${({ theme }) => theme.colors.subtle};
    opacity: 0.45;
    cursor: not-allowed;
    transform: none;
  }

  &.play-button {
    width: 44px;
    height: 44px;
    background: ${({ theme }) => theme.colors.accentGradient};
    border-radius: 16px;
    color: ${({ theme }) => theme.colors.accentText};
    box-shadow: 0 14px 28px ${({ theme }) => theme.colors.playerGlow};

    &:hover {
      transform: scale(1.05);
      filter: brightness(1.08);
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
  color: ${({ theme }) => theme.colors.muted};
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
  height: 6px;
  background: ${({ theme }) => theme.colors.progressTrack};
  border-radius: 999px;
  position: relative;
  cursor: pointer;

  &:hover {
    background: ${({ theme }) => theme.colors.borderStrong};
  }
  
  @media (max-width: 768px) {
    height: 3px;
  }
`

const ProgressFill = styled.div`
  height: 100%;
  background: ${({ theme }) => theme.colors.accentGradient};
  border-radius: 999px;
  width: 30%;
  position: relative;
`

const ProgressHandle = styled.div`
  position: absolute;
  right: -6px;
  top: -4px;
  width: 12px;
  height: 12px;
  background-color: ${({ theme }) => theme.colors.accentHover};
  box-shadow: 0 0 0 5px ${({ theme }) => theme.colors.accentSoft};
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
  gap: 10px;
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
  padding: 10px 12px;
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 20px;
  background: ${({ theme }) => theme.colors.surfaceSoft};
  
  @media (max-width: 768px) {
    display: none;
  }
`

const VolumeControl = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  padding-left: 4px;
`

const VolumeSlider = styled.input`
  appearance: none;
  -webkit-appearance: none;
  width: 100px;
  height: 4px;
  background: linear-gradient(
    to right,
    ${({ theme }) => theme.colors.accent} 0%,
    ${({ theme }) => theme.colors.accent} var(--volume-percent),
    ${({ theme }) => theme.colors.progressTrack} var(--volume-percent),
    ${({ theme }) => theme.colors.progressTrack} 100%
  );
  border-radius: 2px;
  cursor: pointer;
  padding: 0;

  &:hover {
    background: linear-gradient(
      to right,
      ${({ theme }) => theme.colors.accentHover} 0%,
      ${({ theme }) => theme.colors.accentHover} var(--volume-percent),
      ${({ theme }) => theme.colors.borderStrong} var(--volume-percent),
      ${({ theme }) => theme.colors.borderStrong} 100%
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
    background-color: ${({ theme }) => theme.colors.accentHover};
    opacity: 0;
  }

  &:hover::-webkit-slider-thumb,
  &:focus-visible::-webkit-slider-thumb {
    opacity: 1;
  }

  &::-moz-range-track {
    height: 4px;
    border-radius: 2px;
    background: ${({ theme }) => theme.colors.progressTrack};
  }

  &::-moz-range-progress {
    height: 4px;
    border-radius: 2px;
    background: ${({ theme }) => theme.colors.accent};
  }

  &::-moz-range-thumb {
    width: 12px;
    height: 12px;
    border: 0;
    border-radius: 50%;
    background-color: ${({ theme }) => theme.colors.accentHover};
  }
`

const IconButton = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.muted};
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    color: ${({ theme }) => theme.colors.text};
    transform: scale(1.1);
  }

  &.active {
    color: ${({ theme }) => theme.colors.accent};
  }

  &:disabled {
    color: ${({ theme }) => theme.colors.subtle};
    opacity: 0.45;
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

const ConnectControl = styled.div`
  position: relative;
  display: flex;
`

const ConnectButton = styled(IconButton)`
  &.connected {
    color: ${({ theme }) => theme.colors.accent};
  }
`

const ConnectMenu = styled.div`
  position: absolute;
  right: -12px;
  bottom: calc(100% + 14px);
  width: 320px;
  max-height: 420px;
  overflow-y: auto;
  background: ${({ theme }) => theme.colors.surface};
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 18px;
  box-shadow: 0 22px 70px ${({ theme }) => theme.colors.shadow};
  padding: 10px;
  z-index: 1200;
`

const ConnectMenuHeader = styled.div`
  padding: 8px 10px 10px;
  color: ${({ theme }) => theme.colors.text};
  font-size: 14px;
  font-weight: 800;
`

const ConnectMenuHint = styled.div`
  padding: 0 10px 8px;
  color: ${({ theme }) => theme.colors.muted};
  font-size: 11px;
  line-height: 1.4;
`

const ConnectDeviceButton = styled.button<{ $active?: boolean }>`
  width: 100%;
  border: 0;
  border-radius: 14px;
  background: ${({ theme, $active }) => $active ? theme.colors.accentSoft : 'transparent'};
  color: ${({ theme, $active }) => $active ? theme.colors.accent : theme.colors.text};
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 10px;
  text-align: left;
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease;

  &:hover {
    background: ${({ theme }) => theme.colors.controlBg};
  }

  svg {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
  }
`

const ConnectDeviceText = styled.span`
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
`

const ConnectDeviceName = styled.span`
  font-size: 13px;
  font-weight: 800;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`

const ConnectDeviceMeta = styled.span`
  color: ${({ theme }) => theme.colors.muted};
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`

const ConnectMessage = styled.div`
  padding: 10px;
  color: ${({ theme }) => theme.colors.muted};
  font-size: 12px;
`

export const Player: React.FC = () => {
  const [isQueueOpen, setIsQueueOpen] = useState(false);
  const [isLiked, setIsLiked] = useState(false);
  const [rating, setRating] = useState(0);
  const [radioStreamTitle, setRadioStreamTitle] = useState('');
  const [isConnectOpen, setIsConnectOpen] = useState(false);
  const [sessions, setSessions] = useState<UserSession[]>([]);
  const [currentSessionId, setCurrentSessionId] = useState('');
  const [connectError, setConnectError] = useState('');
  const [isLoadingSessions, setIsLoadingSessions] = useState(false);
  const previousVolumeRef = useRef(1);
  const connectControlRef = useRef<HTMLDivElement>(null);
  
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
    setRepeatMode,
    connectedPlaybackSessionId,
    connectedPlaybackDeviceName,
    controlledByPlaybackSessionId,
    connectPlaybackTo,
    returnPlaybackToThisDevice,
  } = useAudio();

  const artworkUrl = getTrackArtworkUrl(currentTrack);
  const isRadioStream = Boolean(currentTrack?.is_external && currentTrack.stream_url);
  const displayDuration = isRadioStream ? duration : (Number.isFinite(duration) && duration > 0 ? duration : currentTrack?.duration || 0);
  const displayTitle = isRadioStream && radioStreamTitle ? radioStreamTitle : currentTrack?.title;
  const displaySubtitle = isRadioStream && radioStreamTitle
    ? currentTrack?.title
    : currentTrack?.artist;
  const isConnectActive = Boolean(connectedPlaybackSessionId || controlledByPlaybackSessionId);
  const connectButtonTitle = connectedPlaybackSessionId
    ? `Playing on ${connectedPlaybackDeviceName}`
    : controlledByPlaybackSessionId
      ? 'Being controlled from another device'
      : 'Connect to a device';

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

  useEffect(() => {
    setRadioStreamTitle('');
    if (!currentTrack?.is_external || !currentTrack.stream_url || !isPlaying) {
      return;
    }

    let active = true;
    const loadRadioMetadata = async () => {
      try {
        const metadata = await pluginsAPI.getRadioMetadata(currentTrack.stream_url || '');
        if (active) {
          setRadioStreamTitle(metadata?.stream_title || '');
        }
      } catch {
        if (active) {
          setRadioStreamTitle('');
        }
      }
    };

    void loadRadioMetadata();
    const interval = window.setInterval(() => void loadRadioMetadata(), 20000);
    return () => {
      active = false;
      window.clearInterval(interval);
    };
  }, [currentTrack, isPlaying]);

  useEffect(() => {
    if (!radioStreamTitle || !currentTrack?.is_external || !('mediaSession' in navigator)) {
      return;
    }

    navigator.mediaSession.metadata = new MediaMetadata({
      title: radioStreamTitle,
      artist: currentTrack.title,
      album: currentTrack.album,
      artwork: artworkUrl ? [{ src: artworkUrl }] : [],
    });
  }, [artworkUrl, currentTrack, radioStreamTitle]);

  useEffect(() => {
    if (!isConnectOpen) {
      return;
    }

    let active = true;
    const loadSessions = async () => {
      setIsLoadingSessions(true);
      setConnectError('');
      try {
        const result = await accountAPI.getSessions();
        if (active) {
          setSessions(result.sessions || []);
          setCurrentSessionId(result.current_session_id || '');
        }
      } catch {
        if (active) {
          setConnectError('Could not load devices');
        }
      } finally {
        if (active) {
          setIsLoadingSessions(false);
        }
      }
    };

    void loadSessions();
    return () => {
      active = false;
    };
  }, [isConnectOpen]);

  useEffect(() => {
    if (!isConnectOpen) {
      return;
    }

    const handlePointerDown = (event: PointerEvent) => {
      if (!connectControlRef.current?.contains(event.target as Node)) {
        setIsConnectOpen(false);
      }
    };

    document.addEventListener('pointerdown', handlePointerDown);
    return () => document.removeEventListener('pointerdown', handlePointerDown);
  }, [isConnectOpen]);

  const formatTime = (seconds: number): string => {
    if (!Number.isFinite(seconds)) return 'LIVE';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const handleProgressClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!Number.isFinite(displayDuration) || displayDuration <= 0) {
      return;
    }
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const percentage = x / rect.width;
    const newTime = percentage * displayDuration;
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

  const getDeviceIcon = (session: UserSession) => {
    const label = `${session.device_name} ${session.user_agent}`.toLowerCase();
    return label.includes('android') || label.includes('mobile') || label.includes('okhttp')
      ? <Smartphone size={20} />
      : <Monitor size={20} />;
  };

  const displayDeviceName = (session: UserSession) => {
    const label = `${session.device_name} ${session.user_agent}`.toLowerCase();
    if (label.includes('wavenode desktop') || label.includes('electron')) {
      return 'WaveNode';
    }
    if (label.includes('wavenode android') || label.includes('okhttp')) {
      return 'WaveNode';
    }
    return session.device_name || 'WaveNode device';
  };

  const displayDeviceMeta = (session: UserSession) => {
    const label = `${session.device_name} ${session.user_agent}`.toLowerCase();
    if (label.includes('wavenode desktop') || label.includes('electron')) {
      return 'Desktop app';
    }
    if (label.includes('wavenode android') || label.includes('okhttp')) {
      return 'Mobile app';
    }
    if (label.includes('iphone') || label.includes('ipad')) {
      return 'Apple mobile device';
    }
    if (label.includes('android') || label.includes('mobile')) {
      return 'Mobile browser';
    }
    return session.device_name || 'WaveNode device';
  };

  const connectSessionKey = (session: UserSession) => {
    const label = `${session.device_name} ${session.user_agent}`.toLowerCase();
    const isDesktopAppSession = label.includes('wavenode desktop') || label.includes('electron');
    const isBrowserSession = !isDesktopAppSession && (label.includes('browser') || label.includes('mozilla'));
    const isMobileSession = label.includes('android') || label.includes('iphone') || label.includes('ipad') || label.includes('mobile') || label.includes('okhttp');
    if (isBrowserSession && !isMobileSession) {
      return `browser|${session.ip_address.trim()}`;
    }
    return `${session.device_name.trim().toLowerCase()}|${session.user_agent.trim().toLowerCase()}|${session.ip_address.trim()}`;
  };

  const activeSessions = sessions
    .filter(session => {
      if (session.revoked_at) {
        return false;
      }
      if (session.id === currentSessionId) {
        return true;
      }
      const lastSeen = Date.parse(session.last_seen_at);
      return Number.isFinite(lastSeen) && Date.now() - lastSeen < 15 * 60 * 1000;
    })
    .sort((a, b) => {
      if (a.id === currentSessionId) return -1;
      if (b.id === currentSessionId) return 1;
      return Date.parse(b.last_seen_at) - Date.parse(a.last_seen_at);
    })
    .filter((session, index, visibleSessions) => {
      if (session.id === currentSessionId) {
        return true;
      }
      const sessionKey = connectSessionKey(session);
      return visibleSessions.findIndex(candidate => {
        if (candidate.id === currentSessionId) {
          return false;
        }
        return connectSessionKey(candidate) === sessionKey;
      }) === index;
    });

  const handleConnectToSession = async (session: UserSession) => {
    setConnectError('');
    try {
      if (session.id === currentSessionId) {
        await returnPlaybackToThisDevice();
      } else {
        await connectPlaybackTo(session.id, displayDeviceName(session));
      }
      setIsConnectOpen(false);
    } catch (error) {
      setConnectError(error instanceof Error ? error.message : 'Could not connect to device');
    }
  };

  const progressPercentage = Number.isFinite(displayDuration) && displayDuration > 0 ? (currentTime / displayDuration) * 100 : 0;
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
              {currentTrack ? displayTitle : 'No track playing'}
            </MobileTrackName>
            <MobileArtistName>
              {currentTrack ? displaySubtitle : 'Select a track to play'}
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
          <Time>{formatTime(displayDuration)}</Time>
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
            <NowPlayingLabel>Now playing</NowPlayingLabel>
            <TrackName>
              {currentTrack ? displayTitle : 'No track playing'}
            </TrackName>
            <ArtistName>
              {currentTrack ? displaySubtitle : 'Select a track to play'}
            </ArtistName>
          </TrackDetails>
          {currentTrack && !currentTrack.is_external && (
            <>
              <IconButton
                className={isLiked ? 'active' : ''}
                onClick={handleToggleLike}
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
                    title={value === rating ? 'Remove rating' : `Rate ${value} out of 5`}
                  >
                    <Star size={13} fill={value <= rating ? 'currentColor' : 'none'} />
                  </IconButton>
                ))}
              </RatingControls>
            </>
          )}
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
            <Time>{formatTime(displayDuration)}</Time>
          </ProgressBar>
        </DesktopPlayerControls>

        <ExtraControls>
          <IconButton disabled title="Lyrics are not available yet">
            <Mic2 size={16} />
          </IconButton>
          <ConnectControl ref={connectControlRef}>
            <ConnectButton
              className={isConnectActive ? 'connected' : ''}
              onClick={() => setIsConnectOpen(open => !open)}
              title={connectButtonTitle}
            >
              <Cast size={16} />
            </ConnectButton>
            {isConnectOpen && (
              <ConnectMenu>
                <ConnectMenuHeader>Connect to a device</ConnectMenuHeader>
                <ConnectMenuHint>
                  Send the current queue to another signed-in WaveNode session. This player becomes the remote control.
                </ConnectMenuHint>
                {isLoadingSessions && <ConnectMessage>Loading devices...</ConnectMessage>}
                {connectError && <ConnectMessage>{connectError}</ConnectMessage>}
                {!isLoadingSessions && activeSessions.length === 0 && (
                  <ConnectMessage>No other active devices found.</ConnectMessage>
                )}
                {!isLoadingSessions && activeSessions.map(session => {
                  const isThisDevice = session.id === currentSessionId;
                  const isActiveTarget = connectedPlaybackSessionId === session.id || (isThisDevice && !connectedPlaybackSessionId);
                  return (
                    <ConnectDeviceButton
                      key={session.id}
                      type="button"
                      $active={isActiveTarget}
                      onClick={() => void handleConnectToSession(session)}
                    >
                      {isThisDevice ? <Monitor size={20} /> : getDeviceIcon(session)}
                      <ConnectDeviceText>
                        <ConnectDeviceName>
                          {isThisDevice ? 'This device' : displayDeviceName(session)}
                        </ConnectDeviceName>
                        <ConnectDeviceMeta>
                          {isThisDevice
                            ? 'Current playback device'
                            : displayDeviceMeta(session)}
                        </ConnectDeviceMeta>
                      </ConnectDeviceText>
                    </ConnectDeviceButton>
                  );
                })}
              </ConnectMenu>
            )}
          </ConnectControl>
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
