import React, { createContext, useCallback, useContext, useEffect, useState, ReactNode, useRef } from 'react';
import { AudiobookProgress, Music, audiobooksAPI, musicAPI, playbackConnectAPI, podcastsAPI, recentlyPlayedAPI, scrobbleAPI } from '../services/api';
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
  playFromQueueAt: (tracks: Music[], index: number, positionSeconds: number) => void;
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
	skipBy: (seconds: number) => void;
	skipBackSeconds: number;
	skipForwardSeconds: number;
	playbackRate: number;
	setPlaybackRate: (rate: number) => void;
	sleepTimerRemaining: number;
	setSleepTimer: (minutes: number | null) => void;
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
  connectedPlaybackSessionId: string | null;
  connectedPlaybackDeviceName: string;
  controlledByPlaybackSessionId: string | null;
  connectPlaybackTo: (targetSessionId: string, deviceName: string) => Promise<void>;
  returnPlaybackToThisDevice: () => Promise<void>;
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
  const [connectedPlaybackSessionId, setConnectedPlaybackSessionId] = useState<string | null>(null);
  const [connectedPlaybackDeviceName, setConnectedPlaybackDeviceName] = useState('');
  const [controlledByPlaybackSessionId, setControlledByPlaybackSessionId] = useState<string | null>(null);
	const [playbackRate, setPlaybackRateState] = useState(1);
	const [podcastPreferences, setPodcastPreferences] = useState({ default_playback_speed: 1, skip_back_seconds: 15, skip_forward_seconds: 30, auto_delete_played: true });
	const [sleepTimerDeadline, setSleepTimerDeadline] = useState<number | null>(null);
	const [sleepTimerRemaining, setSleepTimerRemaining] = useState(0);

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

	useEffect(() => {
		if (authLoading || !user || !token) return;
		void podcastsAPI.getPreferences().then(preferences => {
			setPodcastPreferences(preferences);
			setPlaybackRateState(preferences.default_playback_speed);
			audioService.setPlaybackRate(preferences.default_playback_speed);
		}).catch(error => console.error('Failed to load podcast preferences:', error));
	}, [authLoading, token, user]);

	useEffect(() => {
		if (!sleepTimerDeadline) {
			setSleepTimerRemaining(0);
			return;
		}
		const update = () => {
			const remaining = Math.max(0, Math.ceil((sleepTimerDeadline - Date.now()) / 1000));
			setSleepTimerRemaining(remaining);
			if (remaining === 0) {
				audioService.pause();
				setSleepTimerDeadline(null);
			}
		};
		update();
		const timer = window.setInterval(update, 1000);
		return () => window.clearInterval(timer);
	}, [sleepTimerDeadline]);
  
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
  const connectedPlaybackSessionIdRef = useRef<string | null>(null);
  const remoteProgressIntervalRef = useRef<number | null>(null);
  const durationRef = useRef(duration);
	const podcastQueueLoadedRef = useRef(false);
  const podcastProgressRef = useRef<{ track: Music; position: number; duration: number } | null>(null);
  const lastPodcastReportRef = useRef<{ key: string; position: number }>({ key: '', position: -1 });

  useEffect(() => {
    connectedPlaybackSessionIdRef.current = connectedPlaybackSessionId;
  }, [connectedPlaybackSessionId]);

  useEffect(() => {
    durationRef.current = duration;
  }, [duration]);

  const reportPodcastProgress = useCallback(async () => {
    const snapshot = podcastProgressRef.current;
    if (!snapshot || !['podcast', 'audiobook'].includes(snapshot.track.external_kind || '')) {
      return;
    }
    const position = Math.max(0, Math.round(snapshot.position));
    const totalDuration = Math.max(0, Math.round(snapshot.duration || snapshot.track.duration || 0));
    const isAudiobook = snapshot.track.external_kind === 'audiobook';
    if (isAudiobook && (!snapshot.track.audiobook_id || !snapshot.track.audiobook_chapter_id)) return;
    if (!isAudiobook && (!snapshot.track.podcast_id || !snapshot.track.podcast_episode_id)) return;
    const key = isAudiobook
      ? `${snapshot.track.audiobook_id}:${snapshot.track.audiobook_chapter_id}`
      : `${snapshot.track.podcast_id}:${snapshot.track.podcast_episode_id}`;
    const lastReport = lastPodcastReportRef.current;
    if (lastReport.key === key && lastReport.position === position) {
      return;
    }
    lastPodcastReportRef.current = { key, position };
    try {
      if (isAudiobook) {
        const saved: AudiobookProgress = await audiobooksAPI.updateProgress({
          book_id: snapshot.track.audiobook_id!,
          chapter_id: snapshot.track.audiobook_chapter_id!,
          book_title: snapshot.track.audiobook_title || snapshot.track.album || 'Audiobook',
          author: snapshot.track.audiobook_author || snapshot.track.artist || '',
          chapter_title: snapshot.track.title,
          chapter_number: snapshot.track.audiobook_chapter_number || 0,
          description: snapshot.track.audiobook_description || '',
          image_url: snapshot.track.image_url || '',
          audio_url: snapshot.track.stream_url || '',
          website_url: snapshot.track.audiobook_website_url || '',
          duration_seconds: totalDuration,
          position_seconds: position,
        });
        window.dispatchEvent(new CustomEvent('wavenode:audiobook-progress', { detail: saved }));
      } else {
        const saved = await podcastsAPI.updateProgress({
          podcast_id: snapshot.track.podcast_id!,
          episode_id: snapshot.track.podcast_episode_id!,
          podcast_title: snapshot.track.podcast_title || snapshot.track.album || 'Podcast',
          publisher: snapshot.track.podcast_publisher || '',
          episode_title: snapshot.track.title,
          description: snapshot.track.podcast_description || '',
          image_url: snapshot.track.image_url || '',
          audio_url: snapshot.track.stream_url || '',
          website_url: snapshot.track.podcast_website_url || '',
          published_at: snapshot.track.release_date || undefined,
          duration_seconds: totalDuration,
          position_seconds: position,
        });
	    await podcastsAPI.updateQueue({
		  items: queueRef.current.filter(item => item.external_kind === 'podcast'),
		  current_index: currentTrackIndexRef.current,
		  position_seconds: position,
	    });
        window.dispatchEvent(new CustomEvent('wavenode:podcast-progress', { detail: saved }));
      }
    } catch (error) {
      console.error('Failed to save podcast progress:', error);
      lastPodcastReportRef.current = { key: '', position: -1 };
    }
  }, []);

  useEffect(() => {
    const previous = podcastProgressRef.current;
    if (previous && previous.track.id !== currentTrack?.id) {
      void reportPodcastProgress();
    }
    podcastProgressRef.current = currentTrack && ['podcast', 'audiobook'].includes(currentTrack.external_kind || '')
      ? { track: currentTrack, position: currentTime, duration }
      : null;
  }, [currentTime, currentTrack, duration, reportPodcastProgress]);

  useEffect(() => {
    if (!isPlaying) {
      void reportPodcastProgress();
    }
  }, [isPlaying, reportPodcastProgress]);

  useEffect(() => {
    const interval = window.setInterval(() => void reportPodcastProgress(), 10_000);
    const flushWhenHidden = () => {
      if (document.visibilityState === 'hidden') void reportPodcastProgress();
    };
    window.addEventListener('pagehide', reportPodcastProgress);
    document.addEventListener('visibilitychange', flushWhenHidden);
    return () => {
      window.clearInterval(interval);
      window.removeEventListener('pagehide', reportPodcastProgress);
      document.removeEventListener('visibilitychange', flushWhenHidden);
      void reportPodcastProgress();
    };
  }, [reportPodcastProgress]);

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

  const sendPlaybackCommand = useCallback(async (
    targetSessionId: string,
    tracks: Music[],
    index: number,
    positionSeconds = 0,
    action: 'play_queue' | 'toggle_play_pause' | 'seek' | 'stop' = 'play_queue',
  ) => {
    const safeIndex = tracks.length > 0 ? Math.max(0, Math.min(index, tracks.length - 1)) : 0;
    const positionMs = tracks[safeIndex]?.external_kind === 'radio'
      ? 0
      : Math.max(0, Math.round(positionSeconds * 1000));
    await playbackConnectAPI.createHandoff(
      targetSessionId,
      tracks,
      safeIndex,
      positionMs,
      action,
    );
  }, []);

  const stopRemoteProgressClock = useCallback(() => {
    if (remoteProgressIntervalRef.current) {
      window.clearInterval(remoteProgressIntervalRef.current);
      remoteProgressIntervalRef.current = null;
    }
  }, []);

  const startRemoteProgressClock = useCallback(() => {
    stopRemoteProgressClock();
    if (!connectedPlaybackSessionIdRef.current) {
      return;
    }
    remoteProgressIntervalRef.current = window.setInterval(() => {
      if (!connectedPlaybackSessionIdRef.current) {
        stopRemoteProgressClock();
        return;
      }
      setCurrentTime(previous => {
        const next = previous + 1;
        const currentDuration = durationRef.current;
        return currentDuration > 0 ? Math.min(next, currentDuration) : next;
      });
    }, 1000);
  }, [stopRemoteProgressClock]);

  const playRemoteQueue = useCallback(async (tracks: Music[], index: number, positionSeconds = 0): Promise<boolean> => {
    const targetSessionId = connectedPlaybackSessionIdRef.current;
    if (!targetSessionId || tracks.length === 0) {
      return false;
    }

    const safeIndex = Math.max(0, Math.min(index, tracks.length - 1));
    const nextTrack = tracks[safeIndex];
    setPlaylistQueue(tracks);
    setCurrentTrackIndex(safeIndex);
    setCurrentTrack(nextTrack);
    currentTrackIdRef.current = nextTrack.id;
    setCurrentTime(positionSeconds);
    setDuration(nextTrack.duration || 0);
    setIsLoading(false);
    setControlledByPlaybackSessionId(null);
    audioService.pause();
    setIsPlaying(true);
    startRemoteProgressClock();

    try {
      await sendPlaybackCommand(targetSessionId, tracks, safeIndex, positionSeconds);
      return true;
    } catch (error) {
      console.error('Failed to send playback to connected device:', error);
      setConnectedPlaybackSessionId(null);
      setConnectedPlaybackDeviceName('');
      connectedPlaybackSessionIdRef.current = null;
      stopRemoteProgressClock();
      return false;
    }
  }, [sendPlaybackCommand, startRemoteProgressClock, stopRemoteProgressClock]);

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
      stopRemoteProgressClock();
    };
  }, [stopRemoteProgressClock]);

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
    if (await playRemoteQueue([track], 0, 0)) {
      return;
    }

    setControlledByPlaybackSessionId(null);

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
  }, [addToRecentlyPlayed, playRemoteQueue]);

  const playTrackLocally = useCallback(async (track: Music, startTime = 0) => {
    if (currentTrackIdRef.current === track.id) {
      audioService.seekTo(startTime);
      await audioService.play();
      return;
    }

    requestedTrackIdRef.current = track.id;
    isSettingTrackRef.current = true;
    setIsLoading(true);

    try {
      await audioService.setTrack(track, true, startTime);
      if (requestedTrackIdRef.current !== track.id) {
        return;
      }

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
        console.error('Failed to play track locally:', error);
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
    const targetSessionId = connectedPlaybackSessionIdRef.current;
    if (targetSessionId) {
      try {
        await sendPlaybackCommand(targetSessionId, [], 0, 0, 'toggle_play_pause');
        setIsPlaying(previous => {
          const next = !previous;
          if (next) {
            startRemoteProgressClock();
          } else {
            stopRemoteProgressClock();
          }
          return next;
        });
      } catch (error) {
        console.error('Failed to toggle connected playback:', error);
      }
      return;
    }

    try {
      setControlledByPlaybackSessionId(null);
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

  const seekTo = useCallback((time: number) => {
    const targetSessionId = connectedPlaybackSessionIdRef.current;
    if (targetSessionId) {
      setCurrentTime(time);
      if (isPlaying) {
        startRemoteProgressClock();
      }
      void sendPlaybackCommand(targetSessionId, [], 0, time, 'seek').catch(error => {
        console.error('Failed to seek connected playback:', error);
      });
      return;
    }

    setControlledByPlaybackSessionId(null);
    audioService.seekTo(time);
  }, [isPlaying, sendPlaybackCommand, startRemoteProgressClock]);

	const skipBy = useCallback((seconds: number) => {
		const maximum = durationRef.current > 0 ? durationRef.current : Number.MAX_SAFE_INTEGER;
		seekTo(Math.max(0, Math.min(maximum, currentTime + seconds)));
	}, [currentTime, seekTo]);

	const setPlaybackRate = useCallback((rate: number) => {
		const nextRate = Math.max(0.5, Math.min(3, rate));
		setPlaybackRateState(nextRate);
		audioService.setPlaybackRate(nextRate);
		const nextPreferences = { ...podcastPreferences, default_playback_speed: nextRate };
		setPodcastPreferences(nextPreferences);
		void podcastsAPI.updatePreferences(nextPreferences).catch(error => console.error('Failed to save podcast speed:', error));
	}, [podcastPreferences]);

	const setSleepTimer = useCallback((minutes: number | null) => {
		setSleepTimerDeadline(minutes && minutes > 0 ? Date.now() + minutes * 60_000 : null);
	}, []);

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
      void playRemoteQueue(playlistQueue, nextIndex, 0).then(sent => {
        if (!sent) {
          void playTrack(playlistQueue[nextIndex]);
        }
      });
      return;
    }

    const nextIndex = (currentTrackIndex + 1) % playlistQueue.length;
    setCurrentTrackIndex(nextIndex);
    void playRemoteQueue(playlistQueue, nextIndex, 0).then(sent => {
      if (!sent) {
        void playTrack(playlistQueue[nextIndex]);
      }
    });
  }, [currentTrackIndex, isShuffled, playRemoteQueue, playTrack, playlistQueue]);

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
      if (connectedPlaybackSessionIdRef.current) {
        seekTo(0);
        return;
      }
      audioService.seekTo(0);
      return;
    }

    if (playlistQueue.length > 0) {
      const previousIndex = currentTrackIndex === 0 ? playlistQueue.length - 1 : currentTrackIndex - 1;
      setCurrentTrackIndex(previousIndex);
      void playRemoteQueue(playlistQueue, previousIndex, 0).then(sent => {
        if (!sent) {
          void playTrack(playlistQueue[previousIndex]);
        }
      });
    }
  }, [currentTrackIndex, playRemoteQueue, playTrack, playlistQueue, seekTo]);

  const playPlaylist = (tracks: Music[]) => {
    if (tracks.length > 0) {
      playFromQueue(tracks, 0);
    }
  };

  const playFromQueue = useCallback((tracks: Music[], index: number) => {
    if (tracks.length === 0) {
      return;
    }
    const safeIndex = Math.max(0, Math.min(index, tracks.length - 1));
    void playRemoteQueue(tracks, safeIndex, 0).then(sent => {
      if (!sent) {
        setPlaylistQueue(tracks);
        setCurrentTrackIndex(safeIndex);
        void playTrack(tracks[safeIndex]);
      }
    });
  }, [playRemoteQueue, playTrack]);

  const playLocalFromQueueAt = useCallback((tracks: Music[], index: number, positionSeconds: number) => {
    if (tracks.length === 0) {
      return;
    }
    const safeIndex = Math.max(0, Math.min(index, tracks.length - 1));
    setPlaylistQueue(tracks);
    setCurrentTrackIndex(safeIndex);
    void playTrackLocally(tracks[safeIndex], positionSeconds);
  }, [playTrackLocally]);

  const playFromQueueAt = useCallback((tracks: Music[], index: number, positionSeconds: number) => {
    const safePosition = Math.max(0, positionSeconds);
    void playRemoteQueue(tracks, index, safePosition).then(sent => {
      if (!sent) playLocalFromQueueAt(tracks, index, safePosition);
    });
  }, [playLocalFromQueueAt, playRemoteQueue]);

	useEffect(() => {
		if (authLoading || !user || !token || podcastQueueLoadedRef.current) return;
		podcastQueueLoadedRef.current = true;
		void podcastsAPI.getQueue().then(saved => {
			if (!saved.items.length || (currentTrack && currentTrack.external_kind !== 'podcast')) return;
			const safeIndex = Math.max(0, Math.min(saved.current_index, saved.items.length - 1));
			setPlaylistQueue(saved.items);
			setCurrentTrackIndex(safeIndex);
			setCurrentTrack(saved.items[safeIndex]);
			setCurrentTime(saved.position_seconds);
			setDuration(saved.items[safeIndex].duration || 0);
			void audioService.setTrack(saved.items[safeIndex], false, saved.position_seconds);
		}).catch(error => console.error('Failed to restore podcast queue:', error));
	}, [authLoading, currentTrack, token, user]);

	useEffect(() => {
		if (!podcastQueueLoadedRef.current || currentTrack?.external_kind !== 'podcast') return;
		const timer = window.setTimeout(() => {
			const position = podcastProgressRef.current?.position ?? audioService.getCurrentTime();
			void podcastsAPI.updateQueue({
				items: playlistQueue.filter(item => item.external_kind === 'podcast'),
				current_index: currentTrackIndex,
				position_seconds: Math.max(0, Math.round(position)),
			}).catch(error => console.error('Failed to sync podcast queue:', error));
		}, 1000);
		return () => window.clearTimeout(timer);
	}, [currentTrack?.external_kind, currentTrackIndex, playlistQueue]);

  const connectPlaybackTo = useCallback(async (targetSessionId: string, deviceName: string) => {
    const activeQueue = queueRef.current.length > 0
      ? queueRef.current
      : currentTrack
        ? [currentTrack]
        : [];
    if (activeQueue.length === 0) {
      throw new Error('Select a track before connecting to another device');
    }

    const activeIndex = Math.max(0, Math.min(currentTrackIndexRef.current, activeQueue.length - 1));
    const positionSeconds = audioService.getCurrentTime() || currentTime || 0;
    const previousTarget = connectedPlaybackSessionIdRef.current;
    await sendPlaybackCommand(targetSessionId, activeQueue, activeIndex, positionSeconds);
    if (previousTarget && previousTarget !== targetSessionId) {
      await sendPlaybackCommand(previousTarget, [], 0, 0, 'stop');
    }
    setConnectedPlaybackSessionId(targetSessionId);
    setConnectedPlaybackDeviceName(deviceName);
    setControlledByPlaybackSessionId(null);
    connectedPlaybackSessionIdRef.current = targetSessionId;
    setPlaylistQueue(activeQueue);
    setCurrentTrackIndex(activeIndex);
    setCurrentTrack(activeQueue[activeIndex]);
    setCurrentTime(positionSeconds);
    setDuration(activeQueue[activeIndex]?.duration || duration);
    audioService.pause();
    setIsPlaying(true);
    startRemoteProgressClock();
  }, [currentTime, currentTrack, duration, sendPlaybackCommand, startRemoteProgressClock]);

  const returnPlaybackToThisDevice = useCallback(async () => {
    const previousTarget = connectedPlaybackSessionIdRef.current;
    const wasConnected = Boolean(previousTarget);
	if (previousTarget) {
		try {
			await sendPlaybackCommand(previousTarget, [], 0, 0, 'stop');
		} catch (error) {
			console.warn('Could not stop the previous playback device:', error);
		}
	}
    setConnectedPlaybackSessionId(null);
    setConnectedPlaybackDeviceName('');
    setControlledByPlaybackSessionId(null);
    connectedPlaybackSessionIdRef.current = null;
    stopRemoteProgressClock();
    if (!wasConnected || !currentTrack) {
      return;
    }

    try {
      await audioService.setTrack(currentTrack, isPlaying);
      if (currentTime > 0) {
        audioService.seekTo(currentTime);
      }
    } catch (error) {
      console.error('Failed to return playback to this device:', error);
    }
  }, [currentTime, currentTrack, isPlaying, sendPlaybackCommand, stopRemoteProgressClock]);

  useEffect(() => {
    if (authLoading || !user || !token) {
      return;
    }
    let cancelled = false;
    let inFlight = false;

    const pollForHandoff = async () => {
      if (inFlight || cancelled) {
        return;
      }
      inFlight = true;
      try {
        const command = await playbackConnectAPI.consumePending();
        if (!command || cancelled) {
          return;
        }
        const previousTarget = connectedPlaybackSessionIdRef.current;
        if (command.action === 'play_queue' && previousTarget && previousTarget !== command.source_session_id) {
          await sendPlaybackCommand(previousTarget, [], 0, 0, 'stop');
        }
        setConnectedPlaybackSessionId(null);
        setConnectedPlaybackDeviceName('');
        connectedPlaybackSessionIdRef.current = null;
        stopRemoteProgressClock();
        if (command.source_session_id) {
          setControlledByPlaybackSessionId(command.source_session_id);
        }
        if (command.action === 'toggle_play_pause') {
          await audioService.togglePlayPause();
          return;
        }
        if (command.action === 'seek') {
          audioService.seekTo((command.position_ms ?? 0) / 1000);
          return;
        }
        if (command.action === 'stop') {
          setControlledByPlaybackSessionId(null);
          audioService.pause();
          setIsPlaying(false);
          return;
        }
        if (command.track_ids.length === 0) {
          return;
        }
        let queue = command.tracks ?? [];
        if (queue.length === 0) {
          const library = await musicAPI.getAllMusic();
          if (cancelled) {
            return;
          }
          const tracksById = new Map(library.map(track => [track.id, track]));
          queue = command.track_ids
            .map(trackId => tracksById.get(trackId))
            .filter((track): track is Music => Boolean(track));
        }
        if (queue.length > 0) {
          playLocalFromQueueAt(queue, command.start_index, (command.position_ms ?? 0) / 1000);
        }
      } catch (error) {
        console.warn('Could not receive playback handoff:', error);
      } finally {
        inFlight = false;
      }
    };

    void pollForHandoff();
    const interval = window.setInterval(() => void pollForHandoff(), 3000);
    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, [authLoading, playLocalFromQueueAt, sendPlaybackCommand, stopRemoteProgressClock, token, user]);

  const playPlaylistShuffled = (tracks: Music[]) => {
    if (tracks.length > 0) {
      // Create a shuffled copy of tracks array
      const shuffledTracks = [...tracks].sort(() => Math.random() - 0.5);
      playFromQueue(shuffledTracks, 0);
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
      playFromQueue(playlistQueue, index);
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
    playFromQueueAt,
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
	skipBy,
	skipBackSeconds: podcastPreferences.skip_back_seconds,
	skipForwardSeconds: podcastPreferences.skip_forward_seconds,
	playbackRate,
	setPlaybackRate,
	sleepTimerRemaining,
	setSleepTimer,
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
    connectedPlaybackSessionId,
    connectedPlaybackDeviceName,
    controlledByPlaybackSessionId,
    connectPlaybackTo,
    returnPlaybackToThisDevice,
  };

  return <AudioContext.Provider value={value}>{children}</AudioContext.Provider>;
};
