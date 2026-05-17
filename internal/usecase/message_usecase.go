package usecase

import (
	"context"
	"errors"
	"time"

	"support-backend/internal/domain"
)

var (
	ErrMessageForbidden = errors.New("message forbidden")
)

type MessageUsecase struct {
	ticketRepo  domain.TicketRepository
	msgRepo     domain.MessageRepository
	auditUC     *AuditUsecase
	chatPub     domain.ChatEventPublisher
	terminalPub domain.TerminalLogPublisher
}

func NewMessageUsecase(ticketRepo domain.TicketRepository, msgRepo domain.MessageRepository, auditUC *AuditUsecase, chatPub domain.ChatEventPublisher, terminalPub domain.TerminalLogPublisher) *MessageUsecase {
	return &MessageUsecase{ticketRepo: ticketRepo, msgRepo: msgRepo, auditUC: auditUC, chatPub: chatPub, terminalPub: terminalPub}
}

func (u *MessageUsecase) SendMessage(ctx context.Context, senderID uint64, senderRole string, ticketID uint64, text string) (*domain.Message, error) {
	t, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTicketNotFound
	}

	// VALIDASI: user hanya boleh chat di ticket miliknya.
	if senderRole == domain.RoleUser && t.UserID != senderID {
		return nil, ErrMessageForbidden
	}
	if senderRole == domain.RoleUser {
		// VALIDASI: ticket harus aktif.
		if t.Status != domain.TicketStatusOpen && t.Status != domain.TicketStatusClaimed && t.Status != domain.TicketStatusInProgress {
			return nil, ErrMessageForbidden
		}
	}
	// VALIDASI: CS hanya boleh chat di ticket yang dia claim.
	if senderRole == domain.RoleCS {
		if t.AssignedCSID == nil || *t.AssignedCSID != senderID {
			return nil, ErrMessageForbidden
		}
		// VALIDASI: status ticket harus aktif.
		if t.Status != domain.TicketStatusClaimed && t.Status != domain.TicketStatusInProgress {
			return nil, ErrMessageForbidden
		}
	}

	m := domain.Message{TicketID: ticketID, SenderID: senderID, Message: text, CreatedAt: time.Now()}
	if err := u.msgRepo.Create(ctx, &m); err != nil {
		return nil, err
	}

	// Push real-time ke room ticket.
	u.chatPub.PublishTicketMessage(ticketID, map[string]any{
		"id":          m.ID,
		"ticket_id":   ticketID,
		"sender_id":   senderID,
		"sender_role": senderRole,
		"message":     text,
		"created_at":  m.CreatedAt,
	})

	_ = u.auditUC.Log(ctx, senderID, senderRole, "MESSAGE_SEND", &ticketID, nil)
	u.publishTerminal(ticketID, "INFO", "message_usecase", "chat message accepted by backend from role="+senderRole)
	return &m, nil
}

func (u *MessageUsecase) ListMessages(ctx context.Context, requesterID uint64, requesterRole string, ticketID uint64, limit int) ([]domain.Message, error) {
	t, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTicketNotFound
	}

	// VALIDASI: user hanya boleh lihat message di ticket miliknya.
	if requesterRole == domain.RoleUser && t.UserID != requesterID {
		return nil, ErrMessageForbidden
	}
	if requesterRole == domain.RoleUser {
		// VALIDASI: ticket harus aktif.
		if t.Status != domain.TicketStatusOpen && t.Status != domain.TicketStatusClaimed && t.Status != domain.TicketStatusInProgress {
			return nil, ErrMessageForbidden
		}
	}
	// VALIDASI: CS hanya boleh lihat message di ticket yang dia claim.
	if requesterRole == domain.RoleCS {
		if t.AssignedCSID == nil || *t.AssignedCSID != requesterID {
			return nil, ErrMessageForbidden
		}
		// VALIDASI: status ticket harus aktif.
		if t.Status != domain.TicketStatusClaimed && t.Status != domain.TicketStatusInProgress {
			return nil, ErrMessageForbidden
		}
	}

	return u.msgRepo.ListByTicketID(ctx, ticketID, limit)
}

func (u *MessageUsecase) publishTerminal(ticketID uint64, level, source, message string) {
	if u.terminalPub == nil {
		return
	}
	u.terminalPub.PublishTicketTerminal(ticketID, domain.TerminalLogEntry{TicketID: ticketID, Timestamp: time.Now(), Level: level, Source: source, Message: message})
}
