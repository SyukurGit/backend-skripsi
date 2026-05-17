package usecase

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"support-backend/internal/domain"
)

var (
	ErrTicketNotFound        = errors.New("ticket not found")
	ErrTicketForbidden       = errors.New("ticket forbidden")
	ErrTicketInvalidStatus   = errors.New("ticket invalid status")
	ErrCSActiveTicketLimit   = errors.New("cs active ticket limit")
	ErrTicketAlreadyAssigned = errors.New("ticket already assigned")
)

type TicketUsecase struct {
	ticketRepo  domain.TicketRepository
	jitRepo     domain.JITSessionRepository
	auditUC     *AuditUsecase
	terminalPub domain.TerminalLogPublisher
}

func NewTicketUsecase(ticketRepo domain.TicketRepository, jitRepo domain.JITSessionRepository, auditUC *AuditUsecase, terminalPub domain.TerminalLogPublisher) *TicketUsecase {
	return &TicketUsecase{ticketRepo: ticketRepo, jitRepo: jitRepo, auditUC: auditUC, terminalPub: terminalPub}
}

func (u *TicketUsecase) CreateTicket(ctx context.Context, userID uint64) (*domain.Ticket, error) {
	t := domain.Ticket{UserID: userID, Status: domain.TicketStatusOpen, CreatedAt: time.Now()}
	if err := u.ticketRepo.Create(ctx, &t); err != nil {
		return nil, err
	}
	_ = u.auditUC.Log(ctx, userID, domain.RoleUser, "TICKET_CREATE", &t.ID, map[string]any{"status": t.Status})
	u.publishTerminal(t.ID, "INFO", "ticket_usecase", "ticket created by end user; status=OPEN")
	return &t, nil
}

func (u *TicketUsecase) ListMyTickets(ctx context.Context, userID uint64) ([]domain.Ticket, error) {
	return u.ticketRepo.ListByUserID(ctx, userID)
}

func (u *TicketUsecase) ListOpenUnassigned(ctx context.Context) ([]domain.Ticket, error) {
	return u.ticketRepo.ListOpenUnassigned(ctx)
}

func (u *TicketUsecase) GetByID(ctx context.Context, ticketID uint64) (*domain.Ticket, error) {
	return u.ticketRepo.GetByID(ctx, ticketID)
}

func (u *TicketUsecase) ListMyActiveTicketsCS(ctx context.Context, csID uint64) ([]domain.Ticket, error) {
	return u.ticketRepo.ListByCSIDActive(ctx, csID)
}

func (u *TicketUsecase) ClaimTicket(ctx context.Context, csID uint64, ticketID uint64) error {
	// VALIDASI: customer support maksimal 2 tiket aktif (CLAIMED / IN_PROGRESS)
	cnt, err := u.ticketRepo.CountActiveByCSID(ctx, csID)
	if err != nil {
		return err
	}
	if cnt >= 2 {
		return ErrCSActiveTicketLimit
	}

	if err := u.ticketRepo.AssignToCS(ctx, ticketID, csID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Bisa berarti ticket tidak OPEN atau sudah di-assign.
			return ErrTicketAlreadyAssigned
		}
		return err
	}

	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "TICKET_CLAIM", &ticketID, nil)
	u.publishTerminal(ticketID, "INFO", "ticket_usecase", "ticket claimed by customer service; least privilege context is now bound to assigned CS")
	return nil
}

func (u *TicketUsecase) SetStatus(ctx context.Context, actorID uint64, actorRole string, ticketID uint64, newStatus string) error {
	t, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTicketNotFound
	}

	// VALIDASI: akses ticket dibatasi di layer middleware LP juga, tapi tetap kita jaga di usecase.
	if actorRole == domain.RoleUser && t.UserID != actorID {
		return ErrTicketForbidden
	}
	if actorRole == domain.RoleCS {
		if t.AssignedCSID == nil || *t.AssignedCSID != actorID {
			return ErrTicketForbidden
		}
	}
	// User hanya boleh menutup ticket setelah RESOLVED.
	if actorRole == domain.RoleUser {
		if newStatus != domain.TicketStatusClosed {
			return ErrTicketInvalidStatus
		}
		if t.Status != domain.TicketStatusResolved {
			return ErrTicketInvalidStatus
		}
	}

	// VALIDASI: state machine ticket.
	if !isValidTicketTransition(t.Status, newStatus) {
		return ErrTicketInvalidStatus
	}

	if err := u.ticketRepo.UpdateStatus(ctx, ticketID, newStatus); err != nil {
		return err
	}

	_ = u.auditUC.Log(ctx, actorID, actorRole, "TICKET_STATUS_UPDATE", &ticketID, map[string]any{"status": newStatus})
	u.publishTerminal(ticketID, "INFO", "ticket_usecase", "ticket status updated; transition accepted by backend state machine -> "+newStatus)

	// VALIDASI: auto revoke JIT bila ticket di-close.
	if newStatus == domain.TicketStatusClosed {
		_ = u.jitRepo.RevokeByTicket(ctx, ticketID)
		_ = u.auditUC.Log(ctx, actorID, actorRole, "JIT_REVOKE_TICKET_CLOSED", &ticketID, nil)
		u.publishTerminal(ticketID, "WARN", "ticket_usecase", "ticket closed; all active JIT sessions revoked automatically")
	}

	return nil
}

func (u *TicketUsecase) publishTerminal(ticketID uint64, level, source, message string) {
	if u.terminalPub == nil {
		return
	}
	u.terminalPub.PublishTicketTerminal(ticketID, domain.TerminalLogEntry{TicketID: ticketID, Timestamp: time.Now(), Level: level, Source: source, Message: message})
}

func isValidTicketTransition(from, to string) bool {
	switch from {
	case domain.TicketStatusOpen:
		return to == domain.TicketStatusClaimed
	case domain.TicketStatusClaimed:
		return to == domain.TicketStatusInProgress
	case domain.TicketStatusInProgress:
		return to == domain.TicketStatusResolved
	case domain.TicketStatusResolved:
		return to == domain.TicketStatusClosed
	case domain.TicketStatusClosed:
		return false
	default:
		return false
	}
}
