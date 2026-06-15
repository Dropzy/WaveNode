package websocket

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"music-server/auth"

	"github.com/gorilla/websocket"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// WebSocketManager manages WebSocket connections
type WebSocketManager struct {
	clients     map[*websocket.Conn]bool
	register    chan *websocket.Conn
	unregister  chan *websocket.Conn
	broadcast   chan WebSocketMessage
	mutex       sync.RWMutex
	authHandler *auth.AuthHandler
	upgrader    websocket.Upgrader
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(authHandler *auth.AuthHandler) *WebSocketManager {
	return &WebSocketManager{
		clients:     make(map[*websocket.Conn]bool),
		register:    make(chan *websocket.Conn),
		unregister:  make(chan *websocket.Conn),
		broadcast:   make(chan WebSocketMessage),
		authHandler: authHandler,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}
}

// Start starts the WebSocket manager
func (m *WebSocketManager) Start() {
	go func() {
		for {
			select {
			case client := <-m.register:
				m.mutex.Lock()
				m.clients[client] = true
				m.mutex.Unlock()
				log.Printf("WebSocket client connected. Total clients: %d", len(m.clients))

			case client := <-m.unregister:
				m.mutex.Lock()
				if _, ok := m.clients[client]; ok {
					delete(m.clients, client)
					client.Close()
				}
				m.mutex.Unlock()
				log.Printf("WebSocket client disconnected. Total clients: %d", len(m.clients))

			case message := <-m.broadcast:
				m.mutex.RLock()
				for client := range m.clients {
					err := client.WriteJSON(message)
					if err != nil {
						log.Printf("Error writing to WebSocket: %v", err)
						client.Close()
						delete(m.clients, client)
					}
				}
				m.mutex.RUnlock()
			}
		}
	}()
}

// HandleWebSocket handles WebSocket connections
func (m *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract token from query parameter for WebSocket
	token := r.URL.Query().Get("token")
	if token == "" {
		// Try to get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) == 2 && tokenParts[0] == "Bearer" {
				token = tokenParts[1]
			}
		}
	}

	// Validate token
	_, err := m.authHandler.ValidateJWTForWebSocket(token)
	if err != nil {
		log.Printf("WebSocket authentication failed: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Register new client
	m.register <- conn

	// Start reading messages from this client
	go m.handleClient(conn)
}

// handleClient handles messages from a WebSocket client
func (m *WebSocketManager) handleClient(conn *websocket.Conn) {
	defer func() {
		m.unregister <- conn
	}()

	for {
		var message WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle different message types
		switch message.Type {
		case "ping":
			// Respond with pong
			response := WebSocketMessage{
				Type: "pong",
				Data: map[string]string{"message": "pong"},
			}
			conn.WriteJSON(response)

		case "play":
			// Broadcast play command to all clients
			m.broadcast <- WebSocketMessage{
				Type: "play",
				Data: message.Data,
			}

		case "pause":
			// Broadcast pause command to all clients
			m.broadcast <- WebSocketMessage{
				Type: "pause",
				Data: message.Data,
			}

		case "seek":
			// Broadcast seek command to all clients
			m.broadcast <- WebSocketMessage{
				Type: "seek",
				Data: message.Data,
			}

		case "volume":
			// Broadcast volume change to all clients
			m.broadcast <- WebSocketMessage{
				Type: "volume",
				Data: message.Data,
			}

		default:
			log.Printf("Unknown WebSocket message type: %s", message.Type)
		}
	}
}

// BroadcastMessage broadcasts a message to all connected clients
func (m *WebSocketManager) BroadcastMessage(messageType string, data interface{}) {
	message := WebSocketMessage{
		Type: messageType,
		Data: data,
	}
	m.broadcast <- message
}

// BroadcastScanUpdate broadcasts scan progress updates to all connected clients
func (m *WebSocketManager) BroadcastScanUpdate(scan interface{}) {
	message := WebSocketMessage{
		Type: "scan_update",
		Data: scan,
	}
	m.broadcast <- message
}

// GetClientCount returns number of connected clients
func (m *WebSocketManager) GetClientCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.clients)
}

// Global WebSocket manager instance for scan updates
var globalWebSocketManager *WebSocketManager

// SetGlobalWebSocketManager sets the global WebSocket manager instance
func SetGlobalWebSocketManager(wsManager *WebSocketManager) {
	globalWebSocketManager = wsManager
}

// BroadcastScanUpdateGlobal broadcasts scan updates using the global WebSocket manager
func BroadcastScanUpdate(scan interface{}) {
	if globalWebSocketManager != nil {
		globalWebSocketManager.BroadcastScanUpdate(scan)
	} else {
		log.Printf("Scan update: %+v", scan)
	}
}
