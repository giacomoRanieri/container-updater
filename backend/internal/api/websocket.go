package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"container-updater/backend/internal/logger"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connection from cross-origin frontend
		return true
	},
}

type WSClient struct {
	conn *websocket.Conn
	send chan []byte
}

type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.Mutex
}

var GlobalHub *WSHub

func InitHub() {
	GlobalHub = &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
	}
	go GlobalHub.run()
}

func (h *WSHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logger.Log.Debug("WebSocket client connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			logger.Log.Debug("WebSocket client disconnected")

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func BroadcastEvent(event string, data any) {
	if GlobalHub == nil {
		return
	}

	payload := map[string]any{
		"event": event,
		"data":  data,
	}

	message, err := json.Marshal(payload)
	if err != nil {
		logger.Log.Error("failed to serialize websocket broadcast message", "error", err)
		return
	}

	logger.Log.Debug("broadcasting websocket event", "event", event)
	GlobalHub.broadcast <- message
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Error("failed to upgrade HTTP connection to WebSocket", "error", err)
		return
	}

	client := &WSClient{
		conn: conn,
		send: make(chan []byte, 256),
	}

	GlobalHub.register <- client

	// Start read/write pumps
	go client.writePump()
	go client.readPump()
}

func (c *WSClient) readPump() {
	defer func() {
		GlobalHub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.Warn("websocket read error", "error", err)
			}
			break
		}
		// We only expect one-way communication from server to client,
		// ignore client messages for now.
	}
}

func (c *WSClient) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current packet
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}
