package batch

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - in production, implement proper CORS
		return true
	},
}

// Client represents a WebSocket client
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	jobID  string
	mu     sync.Mutex
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients    map[string]map[*Client]bool // jobID -> set of clients
	broadcast  chan *models.WSProgressUpdate
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *logging.Logger
}

// NewHub creates a new WebSocket hub
func NewHub(logger *logging.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan *models.WSProgressUpdate, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.clients[client.jobID]; !ok {
				h.clients[client.jobID] = make(map[*Client]bool)
			}
			h.clients[client.jobID][client] = true
			h.mu.Unlock()
			h.logger.Info("WebSocket client registered", map[string]interface{}{
				"job_id":         client.jobID,
				"total_clients":  len(h.clients[client.jobID]),
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.jobID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.jobID)
					}
				}
			}
			h.mu.Unlock()
			h.logger.Info("WebSocket client unregistered", map[string]interface{}{
				"job_id": client.jobID,
			})

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[message.JobID]
			h.mu.RUnlock()

			data, err := json.Marshal(message)
			if err != nil {
				h.logger.Error("Failed to marshal WebSocket message", err)
				continue
			}

			for client := range clients {
				select {
				case client.send <- data:
				default:
					h.mu.Lock()
					close(client.send)
					delete(h.clients[message.JobID], client)
					h.mu.Unlock()
				}
			}
		}
	}
}

// BroadcastProgress broadcasts a progress update to all clients watching a job
func (h *Hub) BroadcastProgress(update *models.WSProgressUpdate) {
	h.broadcast <- update
}

// ServeWS handles WebSocket requests
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, jobID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection", err)
		return
	}

	client := &Client{
		hub:   h,
		conn:  conn,
		send:  make(chan []byte, 256),
		jobID: jobID,
	}

	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
