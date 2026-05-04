package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"support-backend/internal/domain"
)

type JITSessionRepository struct {
	db *gorm.DB
}

func NewJITSessionRepository(db *gorm.DB) *JITSessionRepository {
	return &JITSessionRepository{db: db}
}

func (r *JITSessionRepository) Create(ctx context.Context, s *domain.JITSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *JITSessionRepository) GetActive(ctx context.Context, csID uint64, ticketID uint64, feature string, now time.Time) (*domain.JITSession, error) {
	var s domain.JITSession
	err := r.db.WithContext(ctx).
		Where("cs_id = ? AND ticket_id = ? AND feature = ? AND is_active = 1 AND expired_at > ?", csID, ticketID, feature, now).
		Order("id desc").
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *JITSessionRepository) RevokeExpired(ctx context.Context, now time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.JITSession{}).
		Where("is_active = 1 AND expired_at <= ?", now).
		Update("is_active", false).Error
}

func (r *JITSessionRepository) RevokeByTicket(ctx context.Context, ticketID uint64) error {
	return r.db.WithContext(ctx).
		Model(&domain.JITSession{}).
		Where("ticket_id = ? AND is_active = 1", ticketID).
		Update("is_active", false).Error
}

func (r *JITSessionRepository) RevokeExisting(ctx context.Context, csID uint64, ticketID uint64, feature string) error {
	return r.db.WithContext(ctx).
		Model(&domain.JITSession{}).
		Where("cs_id = ? AND ticket_id = ? AND feature = ? AND is_active = 1", csID, ticketID, feature).
		Update("is_active", false).Error
}

func (r *JITSessionRepository) RevokeByCS(ctx context.Context, csID uint64) error {
	return r.db.WithContext(ctx).
		Model(&domain.JITSession{}).
		Where("cs_id = ? AND is_active = 1", csID).
		Update("is_active", false).Error
}
