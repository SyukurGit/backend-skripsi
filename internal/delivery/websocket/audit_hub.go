package websocket

import (
	"encoding/json"
	"sync"
)

type AuditHub struct {
	mu      sync.RWMutex
	clients map[*WSConn]bool
}

func NewAuditHub() *AuditHub {
	return &AuditHub{clients: map[*WSConn]bool{}}
}

func (h *AuditHub) Run() {
	// Hub sederhana; broadcast via method.
}

func (h *AuditHub) Register(c *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = true
}

func (h *AuditHub) Unregister(c *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}

func (h *AuditHub) Broadcast(payload any) {
	b, _ := json.Marshal(map[string]any{"event": "audit_log", "payload": payload})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		_ = c.Write(b)
	}
}
