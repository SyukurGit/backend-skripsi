package websocket

import (
	"encoding/json"
	"sync"
)

type ChatHub struct {
	mu    sync.RWMutex
	rooms map[uint64]map[*WSConn]bool
}

func NewChatHub() *ChatHub {
	return &ChatHub{rooms: map[uint64]map[*WSConn]bool{}}
}

func (h *ChatHub) Run() {
	// Hub sederhana; tidak ada loop select karena kita broadcast via method.
}

func (h *ChatHub) Join(ticketID uint64, c *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[ticketID] == nil {
		h.rooms[ticketID] = map[*WSConn]bool{}
	}
	h.rooms[ticketID][c] = true
}

func (h *ChatHub) Leave(ticketID uint64, c *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[ticketID] != nil {
		delete(h.rooms[ticketID], c)
		if len(h.rooms[ticketID]) == 0 {
			delete(h.rooms, ticketID)
		}
	}
}

func (h *ChatHub) Broadcast(ticketID uint64, payload any) {
	b, _ := json.Marshal(map[string]any{"event": "ticket_message", "payload": payload})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[ticketID] {
		_ = c.Write(b)
	}
}
