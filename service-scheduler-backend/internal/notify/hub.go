package notify

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Hub maintains websocket clients grouped by dealership id
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*websocket.Conn]bool
}

var DefaultHub = NewHub()

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*websocket.Conn]bool)}
}

func (h *Hub) AddClient(dealership string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	m, ok := h.clients[dealership]
	if !ok {
		m = make(map[*websocket.Conn]bool)
		h.clients[dealership] = m
	}
	m[conn] = true
}

func (h *Hub) RemoveClient(dealership string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.clients[dealership]; ok {
		delete(m, conn)
		if len(m) == 0 {
			delete(h.clients, dealership)
		}
	}
}

func (h *Hub) Broadcast(dealership string, msg interface{}) {
	h.mu.RLock()
	conns := h.clients[dealership]
	h.mu.RUnlock()
	if conns == nil {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// send to each connection; on error, remove client
	for conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			// remove connection
			h.RemoveClient(dealership, conn)
			conn.Close()
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS upgrades the request and registers websocket client to dealership topic
func ServeWS(c *gin.Context) {
	dealership := c.Query("dealership_id")
	if dealership == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing dealership_id"})
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	DefaultHub.AddClient(dealership, ws)

	// listen for close or ping messages; we don't expect incoming messages
	go func() {
		defer func() {
			DefaultHub.RemoveClient(dealership, ws)
			ws.Close()
		}()
		for {
			if _, _, err := ws.NextReader(); err != nil {
				return
			}
		}
	}()
}
