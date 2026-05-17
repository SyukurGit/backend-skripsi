package websocket

import "support-backend/internal/domain"

type AuditPublisher struct {
	hub *AuditHub
}

func NewAuditPublisher(hub *AuditHub) *AuditPublisher {
	return &AuditPublisher{hub: hub}
}

func (p *AuditPublisher) PublishAdminAudit(payload any) {
	p.hub.Broadcast(payload)
}

type ChatPublisher struct {
	hub *ChatHub
}

func NewChatPublisher(hub *ChatHub) *ChatPublisher {
	return &ChatPublisher{hub: hub}
}

func (p *ChatPublisher) PublishTicketMessage(ticketID uint64, payload any) {
	p.hub.Broadcast(ticketID, payload)
}

type TerminalPublisher struct {
	hub *TerminalHub
}

func NewTerminalPublisher(hub *TerminalHub) *TerminalPublisher {
	return &TerminalPublisher{hub: hub}
}

func (p *TerminalPublisher) PublishTicketTerminal(ticketID uint64, payload any) {
	if entry, ok := payload.(domain.TerminalLogEntry); ok {
		p.hub.Append(ticketID, entry)
	}
	p.hub.Broadcast(ticketID, payload)
}

func (p *TerminalPublisher) ListTicketTerminal(ticketID uint64, limit int) []domain.TerminalLogEntry {
	return p.hub.List(ticketID, limit)
}
