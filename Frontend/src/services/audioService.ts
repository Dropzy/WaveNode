import { API_BASE_URL, Music } from './api';

export class AudioService {
  private audio: HTMLAudioElement;
  private onTrackChange?: (track: Music) => void;
  private onTimeUpdate?: (currentTime: number, duration: number) => void;
  private onPlayStateChange?: (isPlaying: boolean) => void;
  private onTrackEnd?: () => void;
  private onPlaybackError?: (error: Error) => void;
  private isSettingTrack: boolean = false;
  private loadSequence = 0;

  constructor() {
    this.audio = new Audio();
	this.audio.setAttribute('x-webkit-airplay', 'allow');
    this.audio.preload = 'none'; // Prevent preloading to avoid conflicts
    this.setupEventListeners();
  }

  private setupEventListeners() {
    this.audio.addEventListener('timeupdate', () => {
      if (this.onTimeUpdate) {
        this.onTimeUpdate(this.audio.currentTime, this.audio.duration || 0);
      }
    });

    this.audio.addEventListener('loadedmetadata', () => {
      if (this.onTimeUpdate) {
        this.onTimeUpdate(this.audio.currentTime, this.audio.duration || 0);
      }
    });

    this.audio.addEventListener('play', () => {
      if (this.onPlayStateChange) {
        this.onPlayStateChange(true);
      }
    });

    this.audio.addEventListener('pause', () => {
      if (this.onPlayStateChange) {
        this.onPlayStateChange(false);
      }
    });

    this.audio.addEventListener('ended', () => {
      if (this.onTrackEnd) {
        this.onTrackEnd();
      }
    });

    this.audio.addEventListener('error', (e) => {
      // Only handle errors if we're not in the middle of setting a track
      if (!this.isSettingTrack) {
        console.error('Audio playback error:', e);
        console.error('Audio error details:', {
          src: this.audio.src,
          error: this.audio.error,
          networkState: this.audio.networkState,
          readyState: this.audio.readyState
        });
        if (this.onPlayStateChange) {
          this.onPlayStateChange(false);
        }
        this.onPlaybackError?.(new Error('Audio playback failed'));
      }
    });
  }

  setTrack(track: Music, autoplay = false, startTime = 0): Promise<void> {
    return new Promise((resolve, reject) => {
      const loadSequence = ++this.loadSequence;
      this.isSettingTrack = true;

      // Stop current playback
      this.audio.pause();

      // Clear current source
      this.audio.src = '';

      const token = localStorage.getItem('token');
      const streamUrl = track.is_external && track.stream_url
        ? track.stream_url
        : `${API_BASE_URL}/music/${track.id}/stream?token=${encodeURIComponent(token || '')}`;

      const onCanPlay = () => {
        if (loadSequence !== this.loadSequence) {
          return;
        }
        this.audio.removeEventListener('canplay', onCanPlay);
        this.audio.removeEventListener('error', onError);
        if (startTime > 0 && this.audio.duration && !isNaN(this.audio.duration)) {
          this.audio.currentTime = Math.max(0, Math.min(startTime, this.audio.duration));
        }
        this.isSettingTrack = false;
        resolve();
      };

      const onError = () => {
        if (loadSequence !== this.loadSequence) {
          return;
        }
        this.audio.removeEventListener('canplay', onCanPlay);
        this.audio.removeEventListener('error', onError);
        this.isSettingTrack = false;
        const error = new Error(`Could not load ${track.title}`);
        this.onPlaybackError?.(error);
        reject(error);
      };

      this.audio.addEventListener('canplay', onCanPlay);
      this.audio.addEventListener('error', onError);
      this.audio.src = streamUrl;
      this.onTrackChange?.(track);
      this.audio.load();

      if (autoplay) {
        const playWhenReady = () => {
          if (loadSequence !== this.loadSequence) {
            return;
          }
          this.audio.removeEventListener('canplay', playWhenReady);
          void this.audio.play().catch(error => {
            this.audio.removeEventListener('canplay', onCanPlay);
            this.audio.removeEventListener('error', onError);
            this.isSettingTrack = false;
            reject(error);
          });
        };
        this.audio.addEventListener('canplay', playWhenReady, { once: true });
      }
    });
  }

  async play(): Promise<void> {
    try {
      console.log('Attempting to play audio...');
      await this.audio.play();
      console.log('Audio playing successfully');
    } catch (error) {
      console.error('Failed to play audio:', error);
      throw error;
    }
  }

  pause(): void {
    this.audio.pause();
  }

  togglePlayPause(): Promise<void> {
    if (this.audio.paused) {
      return this.play();
    } else {
      this.pause();
      return Promise.resolve();
    }
  }

  setVolume(volume: number): void {
    this.audio.volume = Math.max(0, Math.min(1, volume));
  }

  getVolume(): number {
    return this.audio.volume;
  }

  seekTo(time: number): void {
    if (this.audio.duration && !isNaN(this.audio.duration)) {
      this.audio.currentTime = Math.max(0, Math.min(time, this.audio.duration));
    }
  }

  getCurrentTime(): number {
    return this.audio.currentTime;
  }

  getDuration(): number {
    return this.audio.duration || 0;
  }

  isPlaying(): boolean {
    return !this.audio.paused;
  }

  isPaused(): boolean {
    return this.audio.paused;
  }

  isEnded(): boolean {
    return this.audio.ended;
  }

  // Event listener setters
  setTrackChangeCallback(callback: (track: Music) => void): void {
    this.onTrackChange = callback;
  }

  setTimeUpdateCallback(callback: (currentTime: number, duration: number) => void): void {
    this.onTimeUpdate = callback;
  }

  setPlayStateChangeCallback(callback: (isPlaying: boolean) => void): void {
    this.onPlayStateChange = callback;
  }

  setTrackEndCallback(callback: () => void): void {
    this.onTrackEnd = callback;
  }

	supportsAirPlay(): boolean {
		return typeof (this.audio as HTMLAudioElement & { webkitShowPlaybackTargetPicker?: () => void }).webkitShowPlaybackTargetPicker === 'function';
	}

	showAirPlayPicker(): void {
		const picker = (this.audio as HTMLAudioElement & { webkitShowPlaybackTargetPicker?: () => void }).webkitShowPlaybackTargetPicker;
		if (!picker) throw new Error('AirPlay is not available in this browser');
		picker.call(this.audio);
	}

	setPlaybackRate(rate: number): void {
		this.audio.playbackRate = Math.max(0.5, Math.min(3, rate));
	}

	getPlaybackRate(): number {
		return this.audio.playbackRate;
	}

  setPlaybackErrorCallback(callback: (error: Error) => void): void {
    this.onPlaybackError = callback;
  }

  // Cleanup
  destroy(): void {
    this.audio.pause();
    this.audio.src = '';
    this.onTrackChange = undefined;
    this.onTimeUpdate = undefined;
    this.onPlayStateChange = undefined;
    this.onTrackEnd = undefined;
    this.onPlaybackError = undefined;
    this.isSettingTrack = false;
  }
}

// Singleton instance
export const audioService = new AudioService();
