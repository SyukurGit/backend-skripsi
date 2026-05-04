package websocket

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
