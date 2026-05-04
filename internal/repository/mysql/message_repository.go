package mysql

import (
	"context"

	"gorm.io/gorm"

	"support-backend/internal/domain"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, m *domain.Message) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *MessageRepository) ListByTicketID(ctx context.Context, ticketID uint64, limit int) ([]domain.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	var items []domain.Message
	err := r.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Order("id asc").Limit(limit).Find(&items).Error
	return items, err
}
