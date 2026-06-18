import React, { createContext, useCallback, useContext, useEffect, useState, ReactNode, useRef } from 'react';
import { Music, recentlyPlayedAPI, scrobbleAPI } from '../services/api';
import { audioService } from '../services/audioService';
import { useAuth } from './AuthContext';
import { getTrackArtworkUrl } from '../utils/mediaUrl';

const playbackStorageKey = 'wavenode-playback-session';

interface SavedPlaybackSession {
  currentTrack: Music | null;
  queue: Music[];
  currentTrackIndex: number;
  currentTime: number;
  volume: number;
  isShuffled: boolean;
  repeatMode: 'none' | 'one' | 'all';
}

const loadPlaybackSession = (): SavedPlaybackSession => {
  const fallback: SavedPlaybackSession = {
    currentTrack: null,
    queue: [],
    currentTrackIndex: 0,
    currentTime: 0,
    volume: 1,
    isShuffled: false,
    repeatMode: 'none',
  };
  try {
    const stored = localStorage.getItem(playbackStorageKey);
    return stored ? { ...fallback, ...JSON.parse(stored) } : fallback;
  } catch {
    return fallback;
  }
};

interface AudioContextType {
  currentTrack: Music | null;
  isPlaying: boolean;
  currentTime: number;
  duration: number;
  volume: number;
  isLoading: boolean;
  playTrack: (track: Music) => Promise<void>;
  playFromQueue: (tracks: Music[], index: number) => void;
  playPlaylist: (tracks: Music[]) => void;
  playPlaylistShuffled: (tracks: Music[]) => void;
  addToQueue: (track: Music) => void;
  removeFromQueue: (index: number) => void;
  reorderQueue: (fromIndex: number, toIndex: number) => void;
  clearQueue: () => void;
  playTrackFromQueue: (index: number) => void;
  togglePlayPause: () => Promise<void>;
  setVolume: (volume: number) => void;
  seekTo: (time: number) => void;
  nextTrack: () => void;
  previousTrack: () => void;
  toggleShuffle: () => void;
  setRepeatMode: (mode: 'none' | 'one' | 'all') => void;
  state: {
    isShuffled: boolean;
    repeatMode: 'none' | 'one' | 'all';
  };
  recentlyPlayedRefreshTrigger: number;
  recentlyPlayed: Music[];
  queue: Music[];
  currentTrackIndex: number;
}

const AudioContext = createContext<AudioContextType | undefined>(undefined);

export const useAudio = () => {
  const context = useContext(AudioContext);
  if (!context) {
    throw new Error('useAudio must be used within an AudioProvider');
  }
  return context;
};

interface AudioProviderProps {
  children: ReactNode;
  queue?: Music[];
  currentIndex?: number;
}

export const AudioProvider: React.FC<AudioProviderProps> = ({ 
  children, 
  queue = [], 
  currentIndex = 0 
}) => {
  const initialSessionRef = useRef<SavedPlaybackSession>(loadPlaybackSession());
  const initialSession = initialSessionRef.current;
  const { user, token, isLoading: authLoading } = useAuth();
  const [currentTrack, setCurrentTrack] = useState<Music | null>(initialSession.currentTrack);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(initialSession.currentTime);
  const [duration, setDuration] = useState(0);
  const [volume, setVolumeState] = useState(initialSession.volume);
  const [isLoading, setIsLoading] = useState(false);
  const [isShuffled, setIsShuffled] = useState(initialSession.isShuffled);
  const [repeatMode, setRepeatModeState] = useState<'none' | 'one' | 'all'>(initialSession.repeatMode);
  const [recentlyPlayedRefreshTrigger, setRecentlyPlayedRefreshTrigger] = useState(0);
  const [recentlyPlayed, setRecentlyPlayed] = useState<Music[]>([]);

  // Load recently played tracks when trigger changes and user is authenticated
  useEffect(() => {
    const loadRecentlyPlayed = async () => {
      // Only make API calls if user is authenticated and token is available
      if (!authLoading && user && token) {
        try {
          const tracks = await recentlyPlayedAPI.getRecentlyPlayed();
          setRecentlyPlayed(tracks);
        } catch (error) {
          console.error('Failed to load recently played tracks:', error);
        }
      }
    };

    loadRecentlyPlayed();
  }, [recentlyPlayedRefreshTrigger, authLoading, user, token]);
  
  // Internal queue management
  const [playlistQueue, setPlaylistQueue] = useState<Music[]>(initialSession.queue.length ? initialSession.queue : queue);
  const [currentTrackIndex, setCurrentTrackIndex] = useState(
    initialSession.queue.length ? initialSession.currentTrackIndex : currentIndex,
  );
  
  // Refs to prevent infinite loops
  const isSettingTrackRef = useRef(false);
  const requestedTrackIdRef = useRef<string | null>(null);
  const currentTrackIdRef = useRef<string | null>(null);
  const playTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const queueRef = useRef<Music[]>(initialSession.queue.length ? initialSession.queue : queue);
  const currentTrackIndexRef = useRef(initialSession.currentTrackIndex);
  const isShuffledRef = useRef(initialSession.isShuffled);
  const repeatModeRef = useRef<'none' | 'one' | 'all'>(initialSession.repeatMode);
  const listenedScrobbledTrackRef = useRef<string | null>(null);
  const nowPlayingScrobbledTrackRef = useRef<string | null>(null);

  useEffect(() => {
    queueRef.current = playlistQueue;
  }, [playlistQueue]);

  useEffect(() => {
    currentTrackIndexRef.current = currentTrackIndex;
  }, [currentTrackIndex]);

  // Function to add track to recently played
  const addToRecentlyPlayed = useCallback(async (track: Music) => {
    try {
      await recentlyPlayedAPI.addRecentlyPlayed(track.id);
      // Trigger refresh of recently played data
      setRecentlyPlayedRefreshTrigger(prev => prev + 1);
    } catch (error) {
      console.error('Failed to add track to recently played:', error);
      // Don't throw error - this is a non-critical feature
    }
  }, []);

  useEffect(() => {
    // Set up audio service callbacks once
    audioService.setTrackChangeCallback((track: Music) => {
      setCurrentTrack(track);
      currentTrackIdRef.current = track.id;
      setIsLoading(false);
      isSettingTrackRef.current = false;
    });

    audioService.setPlayStateChangeCallback((playing: boolean) => {
      setIsPlaying(playing);
    });

    audioService.setTimeUpdateCallback((time: number, dur: number) => {
      setCurrentTime(time);
      setDuration(dur);
    });

    audioService.setPlaybackErrorCallback(() => {
      setIsLoading(false);
      setIsPlaying(false);
      isSettingTrackRef.current = false;
    });

    // Cleanup only on unmount
    const playTimeout = playTimeoutRef;
    return () => {
      if (playTimeout.current) {
        clearTimeout(playTimeout.current);
      }
    };
  }, []); // Empty dependency array - run only once

  useEffect(() => {
    const restoredSession = initialSessionRef.current;
    audioService.setVolume(restoredSession.volume);
    if (!restoredSession.currentTrack) {
      return;
    }

    let active = true;
    setIsLoading(true);
    void audioService.setTrack(restoredSession.currentTrack)
      .then(() => {
        if (active) {
          audioService.seekTo(restoredSession.currentTime);
          setIsLoading(false);
        }
      })
      .catch(() => {
        if (active) {
          setCurrentTrack(null);
          setCurrentTime(0);
          setIsLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, []);

  const playTrack = useCallback(async (track: Music) => {
    if (currentTrackIdRef.current === track.id) {
      audioService.seekTo(0);
      await audioService.play();
      return;
    }

    requestedTrackIdRef.current = track.id;
    isSettingTrackRef.current = true;
    setIsLoading(true);

    try {
      await audioService.setTrack(track, true);
      if (requestedTrackIdRef.current !== track.id) {
        return;
      }
      
      // External live streams are not library tracks and cannot be added to track history.
      if (!track.is_external) {
        await addToRecentlyPlayed(track);
        listenedScrobbledTrackRef.current = null;
        if (nowPlayingScrobbledTrackRef.current !== track.id) {
          nowPlayingScrobbledTrackRef.current = track.id;
          void scrobbleAPI.nowPlaying(track.id).catch(error => {
            console.error('Failed to update now playing:', error);
          });
        }
      }
    } catch (error) {
      if (!(error instanceof DOMException && error.name === 'AbortError')) {
        console.error('Failed to play track:', error);
      }
      if (requestedTrackIdRef.current === track.id) {
        setIsLoading(false);
        isSettingTrackRef.current = false;
      }
    }
  }, [addToRecentlyPlayed]);

  useEffect(() => {
    if (!currentTrack || currentTrack.is_external || !isPlaying || duration <= 0) {
      return;
    }
    if (listenedScrobbledTrackRef.current === currentTrack.id) {
      return;
    }
    const threshold = Math.min(duration / 2, 240);
    if (currentTime < threshold) {
      return;
    }
    listenedScrobbledTrackRef.current = currentTrack.id;
    void scrobbleAPI.listened(currentTrack.id).catch(error => {
      console.error('Failed to scrobble track:', error);
    });
  }, [currentTrack, currentTime, duration, isPlaying]);

  const togglePlayPause = async () => {
    try {
      await audioService.togglePlayPause();
    } catch (error) {
      console.error('Failed to toggle play/pause:', error);
    }
  };

  const setVolume = (newVolume: number) => {
    const clampedVolume = Math.max(0, Math.min(1, newVolume));
    setVolumeState(clampedVolume);
    audioService.setVolume(clampedVolume);
  };

  const seekTo = (time: number) => {
    audioService.seekTo(time);
  };

  const nextTrack = useCallback(() => {
    if (playlistQueue.length === 0) {
      return;
    }

    if (isShuffled && playlistQueue.length > 1) {
      let nextIndex = currentTrackIndex;
      while (nextIndex === currentTrackIndex) {
        nextIndex = Math.floor(Math.random() * playlistQueue.length);
      }
      setCurrentTrackIndex(nextIndex);
      void playTrack(playlistQueue[nextIndex]);
      return;
    }

    const nextIndex = (currentTrackIndex + 1) % playlistQueue.length;
    setCurrentTrackIndex(nextIndex);
    void playTrack(playlistQueue[nextIndex]);
  }, [currentTrackIndex, isShuffled, playTrack, playlistQueue]);

  useEffect(() => {
    audioService.setTrackEndCallback(() => {
      const activeQueue = queueRef.current;
      const activeIndex = currentTrackIndexRef.current;

      if (repeatModeRef.current === 'one') {
        audioService.seekTo(0);
        void audioService.play();
        return;
      }

      if (activeQueue.length === 0) {
        return;
      }

      if (isShuffledRef.current && activeQueue.length > 1) {
        let nextIndex = activeIndex;
        while (nextIndex === activeIndex) {
          nextIndex = Math.floor(Math.random() * activeQueue.length);
        }
        setCurrentTrackIndex(nextIndex);
        void playTrack(activeQueue[nextIndex]);
        return;
      }

      const hasNextTrack = activeIndex < activeQueue.length - 1;
      if (hasNextTrack || repeatModeRef.current === 'all') {
        const nextIndex = hasNextTrack ? activeIndex + 1 : 0;
        setCurrentTrackIndex(nextIndex);
        void playTrack(activeQueue[nextIndex]);
      }
    });
  }, [playTrack]);

  const previousTrack = useCallback(() => {
    if (audioService.getCurrentTime() > 3) {
      audioService.seekTo(0);
      return;
    }

    if (playlistQueue.length > 0) {
      const previousIndex = currentTrackIndex === 0 ? playlistQueue.length - 1 : currentTrackIndex - 1;
      setCurrentTrackIndex(previousIndex);
      void playTrack(playlistQueue[previousIndex]);
    }
  }, [currentTrackIndex, playTrack, playlistQueue]);

  const playPlaylist = (tracks: Music[]) => {
    if (tracks.length > 0) {
      setPlaylistQueue(tracks);
      setCurrentTrackIndex(0);
      playTrack(tracks[0]);
    }
  };

  const playFromQueue = (tracks: Music[], index: number) => {
    if (tracks.length === 0) {
      return;
    }
    const safeIndex = Math.max(0, Math.min(index, tracks.length - 1));
    setPlaylistQueue(tracks);
    setCurrentTrackIndex(safeIndex);
    void playTrack(tracks[safeIndex]);
  };

  const playPlaylistShuffled = (tracks: Music[]) => {
    if (tracks.length > 0) {
      // Create a shuffled copy of tracks array
      const shuffledTracks = [...tracks].sort(() => Math.random() - 0.5);
      setPlaylistQueue(shuffledTracks);
      setCurrentTrackIndex(0);
      playTrack(shuffledTracks[0]);
    }
  };

  const addToQueue = (track: Music) => {
    setPlaylistQueue(prev => [...prev, track]);
  };

  const removeFromQueue = (index: number) => {
    setPlaylistQueue(prev => {
      const newQueue = prev.filter((_, i) => i !== index);
      // If removing the current track, adjust the current index
      if (index === currentTrackIndex) {
        if (newQueue.length > 0) {
          const newIndex = Math.min(index, newQueue.length - 1);
          setCurrentTrackIndex(newIndex);
          playTrack(newQueue[newIndex]);
        } else {
          setCurrentTrack(null);
          setCurrentTrackIndex(0);
        }
      } else if (index < currentTrackIndex) {
        // Adjust current index if removing a track before the current one
        setCurrentTrackIndex(prev => prev - 1);
      }
      return newQueue;
    });
  };

  const reorderQueue = (fromIndex: number, toIndex: number) => {
    setPlaylistQueue(prev => {
      const newQueue = [...prev];
      const [movedTrack] = newQueue.splice(fromIndex, 1);
      newQueue.splice(toIndex, 0, movedTrack);
      
      // Adjust current track index if needed
      if (fromIndex === currentTrackIndex) {
        setCurrentTrackIndex(toIndex);
      } else if (fromIndex < currentTrackIndex && toIndex >= currentTrackIndex) {
        setCurrentTrackIndex(prev => prev - 1);
      } else if (fromIndex > currentTrackIndex && toIndex <= currentTrackIndex) {
        setCurrentTrackIndex(prev => prev + 1);
      }
      
      return newQueue;
    });
  };

  const clearQueue = () => {
    setPlaylistQueue([]);
    setCurrentTrackIndex(0);
    setCurrentTrack(null);
    setIsPlaying(false);
  };

  const playTrackFromQueue = (index: number) => {
    if (index >= 0 && index < playlistQueue.length) {
      setCurrentTrackIndex(index);
      playTrack(playlistQueue[index]);
    }
  };

  const toggleShuffle = () => {
    setIsShuffled(previous => {
      const next = !previous;
      isShuffledRef.current = next;
      return next;
    });
  };

  const setRepeatMode = (mode: 'none' | 'one' | 'all') => {
    repeatModeRef.current = mode;
    setRepeatModeState(mode);
  };

  useEffect(() => {
    const session: SavedPlaybackSession = {
      currentTrack,
      queue: playlistQueue,
      currentTrackIndex,
      currentTime,
      volume,
      isShuffled,
      repeatMode,
    };
    localStorage.setItem(playbackStorageKey, JSON.stringify(session));
  }, [currentTime, currentTrack, currentTrackIndex, isShuffled, playlistQueue, repeatMode, volume]);

  useEffect(() => {
    if (!('mediaSession' in navigator)) {
      return;
    }
    if (!currentTrack) {
      navigator.mediaSession.metadata = null;
      return;
    }

    const artwork = getTrackArtworkUrl(currentTrack);
    navigator.mediaSession.metadata = new MediaMetadata({
      title: currentTrack.title,
      artist: currentTrack.artist,
      album: currentTrack.album,
      artwork: artwork ? [{ src: artwork }] : [],
    });
    navigator.mediaSession.setActionHandler('play', () => void audioService.play());
    navigator.mediaSession.setActionHandler('pause', () => audioService.pause());
    navigator.mediaSession.setActionHandler('previoustrack', previousTrack);
    navigator.mediaSession.setActionHandler('nexttrack', nextTrack);
    navigator.mediaSession.setActionHandler('seekto', details => {
      if (typeof details.seekTime === 'number') {
        audioService.seekTo(details.seekTime);
      }
    });

    return () => {
      navigator.mediaSession.setActionHandler('play', null);
      navigator.mediaSession.setActionHandler('pause', null);
      navigator.mediaSession.setActionHandler('previoustrack', null);
      navigator.mediaSession.setActionHandler('nexttrack', null);
      navigator.mediaSession.setActionHandler('seekto', null);
    };
  }, [currentTrack, nextTrack, previousTrack]);

  const value: AudioContextType = {
    currentTrack,
    isPlaying,
    currentTime,
    duration,
    volume,
    isLoading,
    playTrack,
    playFromQueue,
    playPlaylist,
    playPlaylistShuffled,
    addToQueue,
    removeFromQueue,
    reorderQueue,
    clearQueue,
    playTrackFromQueue,
    togglePlayPause,
    setVolume,
    seekTo,
    nextTrack,
    previousTrack,
    toggleShuffle,
    setRepeatMode,
    state: {
      isShuffled,
      repeatMode,
    },
    recentlyPlayedRefreshTrigger,
    recentlyPlayed,
    queue: playlistQueue,
    currentTrackIndex,
  };

  return <AudioContext.Provider value={value}>{children}</AudioContext.Provider>;
};
