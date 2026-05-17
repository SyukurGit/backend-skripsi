package websocket

import (
	"encoding/json"
	"sync"

	"support-backend/internal/domain"
)

type TerminalHub struct {
	mu    sync.RWMutex
	rooms map[uint64]map[*WSConn]bool
	logs  map[uint64][]domain.TerminalLogEntry
	seq   map[uint64]uint64
}

func NewTerminalHub() *TerminalHub {
	return &TerminalHub{rooms: map[uint64]map[*WSConn]bool{}, logs: map[uint64][]domain.TerminalLogEntry{}, seq: map[uint64]uint64{}}
}

func (h *TerminalHub) Run() {}

func (h *TerminalHub) Join(ticketID uint64, c *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[ticketID] == nil {
		h.rooms[ticketID] = map[*WSConn]bool{}
	}
	h.rooms[ticketID][c] = true
}

func (h *TerminalHub) Leave(ticketID uint64, c *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[ticketID] != nil {
		delete(h.rooms[ticketID], c)
		if len(h.rooms[ticketID]) == 0 {
			delete(h.rooms, ticketID)
		}
	}
}

func (h *TerminalHub) Broadcast(ticketID uint64, payload any) {
	b, _ := json.Marshal(map[string]any{"event": "terminal_log", "payload": payload})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[ticketID] {
		_ = c.Write(b)
	}
}

func (h *TerminalHub) Append(ticketID uint64, entry domain.TerminalLogEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seq[ticketID]++
	entry.Sequence = h.seq[ticketID]
	h.logs[ticketID] = append(h.logs[ticketID], entry)
	if len(h.logs[ticketID]) > 300 {
		h.logs[ticketID] = h.logs[ticketID][len(h.logs[ticketID])-300:]
	}
}

func (h *TerminalHub) List(ticketID uint64, limit int) []domain.TerminalLogEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	items := h.logs[ticketID]
	if limit <= 0 || len(items) <= limit {
		out := make([]domain.TerminalLogEntry, len(items))
		copy(out, items)
		return out
	}
	out := make([]domain.TerminalLogEntry, limit)
	copy(out, items[len(items)-limit:])
	return out
}
