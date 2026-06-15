import { API_BASE_URL, Music } from './api';

export class AudioService {
  private audio: HTMLAudioElement;
  private currentObjectUrl?: string;
  private onTrackChange?: (track: Music) => void;
  private onTimeUpdate?: (currentTime: number, duration: number) => void;
  private onPlayStateChange?: (isPlaying: boolean) => void;
  private onTrackEnd?: () => void;
  private onPlaybackError?: (error: Error) => void;
  private isSettingTrack: boolean = false;
  private loadController?: AbortController;
  private loadSequence = 0;

  constructor() {
    this.audio = new Audio();
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

  setTrack(track: Music): Promise<void> {
    return new Promise((resolve, reject) => {
      const loadSequence = ++this.loadSequence;
      this.loadController?.abort();
      this.loadController = new AbortController();
      this.isSettingTrack = true;

      // Stop current playback
      this.audio.pause();
      
      // Clean up previous object URL if exists
      if (this.currentObjectUrl) {
        URL.revokeObjectURL(this.currentObjectUrl);
        this.currentObjectUrl = undefined;
      }

      // Clear current source
      this.audio.src = '';

      if (track.is_external && track.stream_url) {
        this.audio.src = track.stream_url;
        this.isSettingTrack = false;
        this.onTrackChange?.(track);
        this.audio.load();
        resolve();
        return;
      }

      // Get the streaming URL for the track with authentication
      const token = localStorage.getItem('token');
      const streamUrl = `${API_BASE_URL}/music/${track.id}/stream`;
      
      console.log('Attempting to load track:', track.title, 'ID:', track.id);
      console.log('Stream URL:', streamUrl);
      console.log('Token exists:', !!token);
      
      // Use fetch to get audio blob with proper authentication
      fetch(streamUrl, {
        signal: this.loadController.signal,
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      .then(response => {
        console.log('Fetch response:', response.status, response.statusText);
        console.log('Response headers:', Object.fromEntries(response.headers.entries()));
        
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status} ${response.statusText}`);
        }
        return response.blob();
      })
      .then(blob => {
        if (loadSequence !== this.loadSequence) {
          throw new DOMException('Superseded by a newer track', 'AbortError');
        }
        console.log('Received blob:', {
          type: blob.type,
          size: blob.size,
          isAudio: blob.type.startsWith('audio/')
        });
        
        if (!blob.type.startsWith('audio/')) {
          console.warn('Blob is not audio type:', blob.type);
        }
        
        const objectUrl = URL.createObjectURL(blob);
        this.currentObjectUrl = objectUrl;
        
        // Set up event listeners before setting src
        const onCanPlay = () => {
          if (loadSequence !== this.loadSequence) {
            return;
          }
          console.log('Audio can play triggered');
          this.audio.removeEventListener('canplay', onCanPlay);
          this.audio.removeEventListener('error', onError);
          this.isSettingTrack = false;
          if (this.onTrackChange) {
            this.onTrackChange(track);
          }
          resolve();
        };

        const onError = (e: Event) => {
          console.error('Audio load error:', e);
          this.audio.removeEventListener('canplay', onCanPlay);
          this.audio.removeEventListener('error', onError);
          this.isSettingTrack = false;
          if (this.currentObjectUrl) {
            URL.revokeObjectURL(this.currentObjectUrl);
            this.currentObjectUrl = undefined;
          }
          const error = new Error(`Could not load ${track.title}`);
          this.onPlaybackError?.(error);
          reject(error);
        };

        this.audio.addEventListener('canplay', onCanPlay);
        this.audio.addEventListener('error', onError);
        
        // Small delay to ensure event listeners are properly attached
        setTimeout(() => {
          // Set src after event listeners are in place
          this.audio.src = objectUrl;
          console.log('Set audio src to:', objectUrl);
          this.audio.load();
        }, 10);
      })
      .catch(error => {
        if (error instanceof DOMException && error.name === 'AbortError') {
          reject(error);
          return;
        }
        console.error('Fetch error:', error);
        this.isSettingTrack = false;
        this.onPlaybackError?.(error instanceof Error ? error : new Error('Audio request failed'));
        reject(error);
      });
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

  setPlaybackErrorCallback(callback: (error: Error) => void): void {
    this.onPlaybackError = callback;
  }

  // Cleanup
  destroy(): void {
    this.audio.pause();
    this.loadController?.abort();
    this.loadController = undefined;
    this.audio.src = '';
    if (this.currentObjectUrl) {
      URL.revokeObjectURL(this.currentObjectUrl);
      this.currentObjectUrl = undefined;
    }
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
