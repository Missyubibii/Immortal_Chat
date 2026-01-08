// Package websocket provides WebSocket-based log broadcasting for real-time monitoring
// Following Clean Architecture: This is an Adapter layer component (TAY CHÂN)
package websocket

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// LogHub manages WebSocket connections and broadcasts logs to all connected clients
// Implements io.Writer interface to hook into log.SetOutput()
// Uses Fan-out pattern: 1 log source -> N Admin clients
type LogHub struct {
	// Registered clients map (client -> struct{})
	clients map[*Client]struct{}

	// Buffered channel for log messages (Non-blocking, Drop-if-full strategy)
	broadcast chan []byte

	// Register/Unregister channels for client management
	register   chan *Client
	unregister chan *Client

	// Mutex for thread-safe client map access
	mu sync.RWMutex

	// Secret key for authentication (from MESH_SECRET env)
	secretKey string

	// WebSocket upgrader with permissive settings for Dashboard
	upgrader websocket.Upgrader
}

// Client represents a connected WebSocket client
type Client struct {
	hub  *LogHub
	conn *websocket.Conn
	send chan []byte
}

const (
	// Buffer sizes tuned per TÀI LIỆU: "Giữ trên RAM, mất cũng được"
	broadcastBufferSize = 256
	clientBufferSize    = 64

	// WebSocket timeouts
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// NewLogHub creates a new LogHub instance
// secretKey: MESH_SECRET from environment for authentication
func NewLogHub(secretKey string) *LogHub {
	hub := &LogHub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan []byte, broadcastBufferSize),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		secretKey:  secretKey,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Allow all origins for internal Dashboard (protected by secret key)
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
	return hub
}

// Run starts the hub's main event loop (call as goroutine)
// Handles client registration/unregistration and message broadcasting
func (h *LogHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = struct{}{}
			h.mu.Unlock()
			log.Printf("[LogHub] 🟢 Client connected (Total: %d)", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[LogHub] 🔴 Client disconnected (Total: %d)", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// CRITICAL: Non-blocking send to each client
				// Per core he thong loi.docx: "Tắc nghẽn Webhook" prevention
				select {
				case client.send <- message:
				default:
					// Client buffer full -> Skip this message for this client
					// This prevents slow clients from blocking the hub
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Write implements io.Writer interface for log.SetOutput() hook
// CRITICAL: Non-blocking design per core he thong loi.docx
// "Code Go xử lý lâu gây tắc nghẽn Webhook" prevention
func (h *LogHub) Write(p []byte) (n int, err error) {
	// Clone the message to avoid data race
	msg := make([]byte, len(p))
	copy(msg, p)

	// Strip trailing newline for cleaner display
	msg = bytes.TrimRight(msg, "\n\r")

	// CRITICAL: Non-blocking send with Drop-if-full strategy
	// Per TÀI LIỆU: "Info/Debug: Giữ trên RAM... mất cũng được"
	select {
	case h.broadcast <- msg:
		// Message queued successfully
	default:
		// Channel full -> Drop message to save main system
		// This is intentional: Logging must NEVER block the main app
	}

	// Always return success (we wrote the original bytes, even if broadcast was dropped)
	return len(p), nil
}

// ServeWS handles WebSocket upgrade requests
// Security: Requires ?secret_key= query parameter matching MESH_SECRET
// Route: /ws/logs?secret_key=YOUR_MESH_SECRET
func (h *LogHub) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Security Check: Validate secret key
	queryKey := r.URL.Query().Get("secret_key")
	if queryKey == "" || queryKey != h.secretKey {
		http.Error(w, "Unauthorized: Invalid or missing secret_key", http.StatusUnauthorized)
		log.Printf("[LogHub] ⚠️ Unauthorized WebSocket attempt from %s", r.RemoteAddr)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[LogHub] ❌ WebSocket upgrade failed: %v", err)
		return
	}

	// Create new client
	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, clientBufferSize),
	}

	// Register client with hub
	h.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// readPump handles incoming messages from client (mostly pong responses)
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// Read messages (we don't expect any, but need to drain the connection)
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[LogHub] Read error: %v", err)
			}
			break
		}
	}
}

// writePump sends messages from hub to client via WebSocket
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send message as text
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch pending messages for efficiency
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GetSecretKey returns the configured secret key for external validation
func (h *LogHub) GetSecretKey() string {
	return h.secretKey
}

// ClientCount returns the current number of connected clients
func (h *LogHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetMeshSecret helper function to read MESH_SECRET from environment
func GetMeshSecret() string {
	return os.Getenv("MESH_SECRET")
}
