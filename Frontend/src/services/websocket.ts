// Helper function to get valid token
import { API_ORIGIN, tokenUtils } from './api';

export interface ScanStatus {
  id: string; // Backend sends string ID
  scan_id?: number;
  type: string;
  status: 'pending' | 'running' | 'stopping' | 'completed' | 'completed_with_errors' | 'failed' | 'stopped';
  progress: number;
  processed: number;
  files_scanned: number;
  files_found: number;
  total_files: number;
  songs_added: number;
  songs_updated: number;
  tracks_skipped: number;
  duplicates: number;
  current_file?: string;
  error?: string;
  errors?: string[];
  started_at?: string;
  completed_at?: string;
}

export type ScanUpdateCallback = (scan: ScanStatus) => void;

class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private callbacks: Set<ScanUpdateCallback> = new Set();
  private isConnecting = false;
  private connectionPromise: Promise<void> | null = null;
  private heartbeatInterval: NodeJS.Timeout | null = null;
  private lastMessageTime = 0;

  constructor() {
    // Don't connect immediately - wait for explicit call
  }

  public async connect(): Promise<void> {
    // Return existing connection promise if connecting
    if (this.connectionPromise) {
      console.log('WebSocket: Connection already in progress, returning existing promise');
      return this.connectionPromise;
    }

    // If already connected, return immediately
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      console.log('WebSocket: Already connected, returning immediately');
      return Promise.resolve();
    }

    console.log('WebSocket: Starting new connection attempt');
    this.connectionPromise = this.establishConnection();
    return this.connectionPromise;
  }

  private async establishConnection(): Promise<void> {
    if (this.isConnecting) {
      console.log('WebSocket: Already connecting, skipping');
      return;
    }

    this.isConnecting = true;

    try {
      const token = await tokenUtils.getValidToken();
      if (!token) {
        console.warn('No valid authentication token available for WebSocket connection');
        this.isConnecting = false;
        this.connectionPromise = null;
        return;
      }

      const wsEndpoint = new URL('/ws', API_ORIGIN);
      wsEndpoint.protocol = wsEndpoint.protocol === 'https:' ? 'wss:' : 'ws:';
      wsEndpoint.searchParams.set('token', token);
      const wsUrl = wsEndpoint.toString();
      console.log(`WebSocket: Connecting to ${wsUrl}`);
      
      return new Promise<void>((resolve, reject) => {
        let connectionTimeout: NodeJS.Timeout | null = null;

        try {
          this.ws = new WebSocket(wsUrl);

          connectionTimeout = setTimeout(() => {
            console.error('WebSocket: Connection timeout after 10 seconds');
            this.isConnecting = false;
            this.connectionPromise = null;
            reject(new Error('WebSocket connection timeout'));
          }, 10000); // 10 second timeout

          this.ws.onopen = () => {
            if (connectionTimeout) {
              clearTimeout(connectionTimeout);
            }
            console.log('WebSocket: Connected successfully for scan updates');
            this.isConnecting = false;
            this.reconnectAttempts = 0;
            this.connectionPromise = null;
            this.startHeartbeat();
            resolve();
          };

          this.ws.onmessage = (event) => {
            this.lastMessageTime = Date.now();
            console.log('WebSocket: Received message:', event.data);
            
            try {
              const message = JSON.parse(event.data);
              console.log('WebSocket: Parsed message:', message);
              
              // Handle different message types
              if (message.type === 'scan_update') {
                let scan: ScanStatus | null = null;
                
                // Debug: Log the raw message data to understand the structure
                console.log('WebSocket: Raw message.data type:', typeof message.data);
                console.log('WebSocket: Raw message.data:', JSON.stringify(message.data, null, 2));
                
                // Handle both object and string data more robustly
                if (message.data) {
                  try {
                    // If message.data is an object, use it directly
                    scan = message.data as ScanStatus;
                    
                    // Debug: Log the parsed scan status
                    console.log('WebSocket: Extracted scan status (object):', scan.status);
                  } catch {
                    // If direct assignment fails, try parsing as string
                    console.warn('WebSocket: Failed to process message.data as object, trying string parse');
                    if (typeof message.data === 'string') {
                      scan = JSON.parse(message.data) as ScanStatus;
                      console.log('WebSocket: Extracted scan status (string fallback):', scan.status);
                    }
                  }
                } else {
                  console.warn('WebSocket: No message.data available');
                  return;
                }

                if (!scan) {
                  console.warn('WebSocket: Unable to build scan update from message:', message);
                  return;
                }
                
                // Ensure required fields exist
                if (!scan.id) {
                  console.warn('Scan update missing ID, using timestamp:', message);
                  scan.id = Date.now().toString();
                }
                
                console.log('WebSocket: Processed scan update:', scan);
                this.notifyCallbacks(scan);
              } else if (message.type === 'pong') {
                console.log('WebSocket: Received pong response');
              } else {
                console.log('WebSocket: Received unknown message type:', message.type);
              }
            } catch (error) {
              console.error('WebSocket: Error parsing message:', error);
              console.error('WebSocket: Raw message:', event.data);
            }
          };

          this.ws.onclose = (event) => {
            if (connectionTimeout) {
              clearTimeout(connectionTimeout);
            }
            console.log(`WebSocket: Disconnected - Code: ${event.code}, Reason: ${event.reason}`);
            this.isConnecting = false;
            this.ws = null;
            this.connectionPromise = null;
            this.stopHeartbeat();
            
            // Attempt to reconnect if not a normal closure
            if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
              this.attemptReconnect();
            }
          };

          this.ws.onerror = (error) => {
            if (connectionTimeout) {
              clearTimeout(connectionTimeout);
            }
            console.error('WebSocket: Error occurred:', error);
            this.isConnecting = false;
            this.connectionPromise = null;
            this.stopHeartbeat();
            reject(error);
          };

        } catch (error) {
          if (connectionTimeout) {
            clearTimeout(connectionTimeout);
          }
          console.error('WebSocket: Error creating connection:', error);
          this.isConnecting = false;
          this.connectionPromise = null;
          this.stopHeartbeat();
          reject(error);
        }
      });

    } catch (error) {
      this.isConnecting = false;
      this.connectionPromise = null;
      this.stopHeartbeat();
      throw error;
    }
  }

  private startHeartbeat() {
    this.stopHeartbeat(); // Clear any existing interval
    this.heartbeatInterval = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        const timeSinceLastMessage = Date.now() - this.lastMessageTime;
        console.log(`WebSocket: Heartbeat - Time since last message: ${timeSinceLastMessage}ms`);
        
        // Send ping every 30 seconds
        this.ws.send(JSON.stringify({ type: 'ping' }));
      } else {
        console.warn('WebSocket: Heartbeat failed - connection not open');
        this.stopHeartbeat();
      }
    }, 30000); // 30 seconds
  }

  private stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('WebSocket: Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    
    console.log(`WebSocket: Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
    
    setTimeout(() => {
      this.connect().catch(error => {
        console.error('WebSocket: Reconnection failed:', error);
      });
    }, delay);
  }

  private notifyCallbacks(scan: ScanStatus) {
    console.log(`WebSocket: Notifying ${this.callbacks.size} callbacks`);
    this.callbacks.forEach(callback => {
      try {
        callback(scan);
      } catch (error) {
        console.error('WebSocket: Error in scan update callback:', error);
      }
    });
  }

  public onScanUpdate(callback: ScanUpdateCallback) {
    console.log('WebSocket: Adding scan update callback');
    this.callbacks.add(callback);
    
    // Return unsubscribe function
    return () => {
      console.log('WebSocket: Removing scan update callback');
      this.callbacks.delete(callback);
    };
  }

  public disconnect() {
    console.log('WebSocket: Disconnecting');
    this.stopHeartbeat();
    
    if (this.connectionPromise) {
      this.connectionPromise = null;
    }
    
    if (this.ws) {
      this.ws.close(1000, 'Client disconnecting');
      this.ws = null;
    }
    this.callbacks.clear();
    this.isConnecting = false;
    this.reconnectAttempts = 0;
  }

  public isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  public getConnectionState(): string {
    if (!this.ws) return 'disconnected';
    
    switch (this.ws.readyState) {
      case WebSocket.CONNECTING: return 'connecting';
      case WebSocket.OPEN: return 'connected';
      case WebSocket.CLOSING: return 'closing';
      case WebSocket.CLOSED: return 'closed';
      default: return 'unknown';
    }
  }

  public getDebugInfo() {
    return {
      isConnected: this.isConnected(),
      connectionState: this.getConnectionState(),
      reconnectAttempts: this.reconnectAttempts,
      callbackCount: this.callbacks.size,
      lastMessageTime: this.lastMessageTime,
      timeSinceLastMessage: this.lastMessageTime ? Date.now() - this.lastMessageTime : 0
    };
  }
}

// Create singleton instance
const websocketService = new WebSocketService();

export default websocketService;
