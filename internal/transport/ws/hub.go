package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/pkg/logger"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 4096
)

// Hub manages all active WebSocket connections and fans out events.
type Hub struct {
	mu       sync.RWMutex
	clients  map[string]map[*Client]struct{} // tenantID → clients
	eventBus domain.EventBus
	log      *logger.Logger
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client represents a single WebSocket connection.
type Client struct {
	tenantID string
	send     chan []byte
	hub      *Hub
	conn     wsConn
}

// wsConn is a minimal interface over the real WebSocket connection,
// making it easy to swap the underlying library (e.g. nhooyr/websocket,
// gorilla/websocket) without touching Hub logic.
type wsConn interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	SetReadLimit(limit int64)
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetPongHandler(h func(string) error)
	Close() error
}

// NewHub creates a hub and starts the event fan-out loop.
func NewHub(eventBus domain.EventBus, log *logger.Logger) *Hub {
	return &Hub{
		clients:  make(map[string]map[*Client]struct{}),
		eventBus: eventBus,
		log:      log,
	}
}

// ServeHTTP upgrades the HTTP connection to WebSocket.
// The caller must provide an upgraded wsConn (gorilla/websocket or similar).
// Example with gorilla:
//
//	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
//	conn, _ := upgrader.Upgrade(w, r, nil)
//	hub.ServeWS(conn, tenantID)
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithContext(r.Context()).Warn("websocket upgrade failed", map[string]interface{}{"err": err.Error()})
		return
	}
	h.ServeWS(conn, tenantID)
}

// ServeWS registers a new client and starts its read/write pumps.
func (h *Hub) ServeWS(conn wsConn, tenantID string) {
	client := &Client{
		tenantID: tenantID,
		send:     make(chan []byte, 64),
		hub:      h,
		conn:     conn,
	}

	h.register(client)

	// Subscribe to event bus for this tenant
	evtCh, unsub := h.eventBus.Subscribe(tenantID)

	go client.writePump()
	go client.eventForwarder(evtCh)

	client.readPump() // blocks until client disconnects

	unsub()
	h.unregister(client)
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.tenantID] == nil {
		h.clients[c.tenantID] = make(map[*Client]struct{})
	}
	h.clients[c.tenantID][c] = struct{}{}
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c.tenantID][c]; ok {
		delete(h.clients[c.tenantID], c)
		close(c.send)
	}
}

// ─── Client pumps ─────────────────────────────────────────────────────────────

// readPump drains incoming messages (ping/pong handling).
func (c *Client) readPump() {
	defer c.conn.Close()
	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writePump flushes outbound messages and sends periodic pings.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	const (
		textMessage = 1 // websocket.TextMessage
		pingMessage = 9 // websocket.PingMessage
	)

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(pingMessage, nil)
				return
			}
			c.conn.WriteMessage(textMessage, msg)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(pingMessage, nil); err != nil {
				return
			}
		}
	}
}

// eventForwarder reads from the event bus and pushes to the client's send channel.
func (c *Client) eventForwarder(ch <-chan domain.Event) {
	for evt := range ch {
		b, err := json.Marshal(evt)
		if err != nil {
			continue
		}
		select {
		case c.send <- b:
		default:
			// slow client — drop
		}
	}
}

// ─── Broadcast (optional direct use) ─────────────────────────────────────────

// Broadcast sends raw bytes to all clients of a given tenant.
func (h *Hub) Broadcast(ctx context.Context, tenantID string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients[tenantID] {
		select {
		case c.send <- data:
		default:
		}
	}
	_ = ctx
}
