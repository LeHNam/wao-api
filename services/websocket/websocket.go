package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

type WebSocketService struct {
	upgrader    websocket.Upgrader
	connections map[*websocket.Conn]bool
	mu          sync.Mutex
}

func NewWebSocketService() *WebSocketService {
	return &WebSocketService{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for simplicity; customize as needed
				return true
			},
		},
		connections: make(map[*websocket.Conn]bool),
		mu:          sync.Mutex{},
	}
}

func (ws *WebSocketService) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return ws.upgrader.Upgrade(w, r, nil)
}

func (ws *WebSocketService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the connection to WebSocket
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Ensure the connections map is initialized
	if ws.connections == nil {
		ws.mu.Lock()
		ws.connections = make(map[*websocket.Conn]bool)
		ws.mu.Unlock()
	}

	// Add connection to the list
	ws.mu.Lock()
	ws.connections[conn] = true
	ws.mu.Unlock()

	// Remove connection when done
	defer func() {
		ws.mu.Lock()
		delete(ws.connections, conn)
		ws.mu.Unlock()
	}()

	// Example: Keep the connection open
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (ws *WebSocketService) Broadcast(message interface{}) error {
	fmt.Print("ws.Broadcast called with message: ", ws)
	fmt.Println(ws)
	ws.mu.Lock()
	defer ws.mu.Unlock()

	for conn := range ws.connections {
		err := conn.WriteJSON(message)
		if err != nil {
			// Remove the connection if sending fails
			conn.Close()
			delete(ws.connections, conn)
		}
	}
	return nil
}
