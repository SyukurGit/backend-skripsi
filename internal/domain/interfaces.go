package domain

import (
	"context"
	"time"

	"gorm.io/datatypes"
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uint64) (*User, error)
	UpdateEmail(ctx context.Context, userID uint64, newEmail string) error
	UpdatePassword(ctx context.Context, userID uint64, newHashedPassword string) error
}

type UserProfileRepository interface {
	GetByUserID(ctx context.Context, userID uint64) (*UserProfile, error)
	UpdateKYCData(ctx context.Context, userID uint64, kyc datatypes.JSON) error
}

type TicketRepository interface {
	Create(ctx context.Context, t *Ticket) error
	GetByID(ctx context.Context, id uint64) (*Ticket, error)
	ListByUserID(ctx context.Context, userID uint64) ([]Ticket, error)
	ListOpenUnassigned(ctx context.Context) ([]Ticket, error)
	ListByCSIDActive(ctx context.Context, csID uint64) ([]Ticket, error)
	CountActiveByCSID(ctx context.Context, csID uint64) (int64, error)
	AssignToCS(ctx context.Context, ticketID uint64, csID uint64) error
	UpdateStatus(ctx context.Context, ticketID uint64, status string) error
}

type MessageRepository interface {
	Create(ctx context.Context, m *Message) error
	ListByTicketID(ctx context.Context, ticketID uint64, limit int) ([]Message, error)
}

type JITSessionRepository interface {
	Create(ctx context.Context, s *JITSession) error
	GetActive(ctx context.Context, csID uint64, ticketID uint64, feature string, now time.Time) (*JITSession, error)
	RevokeExpired(ctx context.Context, now time.Time) error
	RevokeByTicket(ctx context.Context, ticketID uint64) error
	RevokeExisting(ctx context.Context, csID uint64, ticketID uint64, feature string) error
	RevokeByCS(ctx context.Context, csID uint64) error
}

type AuditLogRepository interface {
	Create(ctx context.Context, a *AuditLog) error
	List(ctx context.Context, level string, limit int) ([]AuditLog, error)
}

type ChatEventPublisher interface {
	PublishTicketMessage(ticketID uint64, payload any)
}

type AuditEventPublisher interface {
	PublishAdminAudit(payload any)
}
