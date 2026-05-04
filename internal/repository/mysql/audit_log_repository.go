package mysql

import (
	"context"

	"gorm.io/gorm"

	"support-backend/internal/domain"
)

type AuditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, a *domain.AuditLog) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *AuditLogRepository) List(ctx context.Context, level string, limit int) ([]domain.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	q := r.db.WithContext(ctx).Model(&domain.AuditLog{}).Order("id desc").Limit(limit)
	if level != "" {
		q = q.Where("level = ?", level)
	}
	var items []domain.AuditLog
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
