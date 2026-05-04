package usecase

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/datatypes"

	"support-backend/internal/domain"
)

type AuditUsecase struct {
	repo      domain.AuditLogRepository
	publisher domain.AuditEventPublisher
}

func NewAuditUsecase(repo domain.AuditLogRepository, publisher domain.AuditEventPublisher) *AuditUsecase {
	return &AuditUsecase{repo: repo, publisher: publisher}
}

func (u *AuditUsecase) Log(ctx context.Context, userID uint64, role, action string, ticketID *uint64, metadata any) error {
	var meta datatypes.JSON
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		meta = datatypes.JSON(b)
	}

	level := deriveAuditLevel(action)

	a := domain.AuditLog{
		UserID:    userID,
		Role:      role,
		Level:     level,
		Action:    action,
		TicketID:  ticketID,
		Metadata:  meta,
		CreatedAt: time.Now(),
	}
	if err := u.repo.Create(ctx, &a); err != nil {
		return err
	}

	// Kirim real-time ke admin (admin only) untuk level HIGH dan MEDIUM.
	if a.Level == domain.AuditLevelHigh || a.Level == domain.AuditLevelMedium {
		u.publisher.PublishAdminAudit(map[string]any{
			"id":         a.ID,
			"user_id":    a.UserID,
			"role":       a.Role,
			"level":      a.Level,
			"action":     a.Action,
			"ticket_id":  a.TicketID,
			"metadata":   json.RawMessage(a.Metadata),
			"created_at": a.CreatedAt,
		})
	}

	return nil
}

func (u *AuditUsecase) List(ctx context.Context, level string, limit int) ([]domain.AuditLog, error) {
	return u.repo.List(ctx, level, limit)
}

func deriveAuditLevel(action string) string {
	// Mapping sesuai spesifikasi.
	switch action {
	case "JIT_REQUEST", "RESET_PASSWORD", "UNBLOCK_ACCOUNT":
		return domain.AuditLevelHigh
	case "VIEW_KYC", "VIEW_TRANSACTION":
		return domain.AuditLevelMedium
	default:
		return domain.AuditLevelLow
	}
}
