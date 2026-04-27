package handler

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev
	},
}

type WebSocketHandler struct {
	clients   map[*websocket.Conn]bool
	broadcast chan []byte
	mutex     sync.Mutex
}

func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte),
	}
}

func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to set websocket upgrade: %v", err)
		return
	}
	defer ws.Close()

	h.mutex.Lock()
	h.clients[ws] = true
	h.mutex.Unlock()

	log.Println("New WebSocket client connected")

	// Read messages (ping/pong) to keep connection alive
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			h.mutex.Lock()
			delete(h.clients, ws)
			h.mutex.Unlock()
			log.Println("WebSocket client disconnected")
			break
		}
	}
}

// BroadcastMessages listens to the broadcast channel and sends messages to all connected clients
func (h *WebSocketHandler) BroadcastMessages() {
	for {
		msg := <-h.broadcast
		h.mutex.Lock()
		for client := range h.clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("Error sending message to client: %v", err)
				client.Close()
				delete(h.clients, client)
			}
		}
		h.mutex.Unlock()
	}
}

func (h *WebSocketHandler) SendMessage(msg []byte) {
	h.broadcast <- msg
}
