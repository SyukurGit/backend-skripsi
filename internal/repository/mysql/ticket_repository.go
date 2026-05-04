package mysql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"support-backend/internal/domain"
)

type TicketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

func (r *TicketRepository) Create(ctx context.Context, t *domain.Ticket) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *TicketRepository) GetByID(ctx context.Context, id uint64) (*domain.Ticket, error) {
	var t domain.Ticket
	err := r.db.WithContext(ctx).First(&t, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TicketRepository) ListByUserID(ctx context.Context, userID uint64) ([]domain.Ticket, error) {
	var items []domain.Ticket
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&items).Error
	return items, err
}

func (r *TicketRepository) ListOpenUnassigned(ctx context.Context) ([]domain.Ticket, error) {
	var items []domain.Ticket
	err := r.db.WithContext(ctx).
		Where("status = ? AND assigned_cs_id IS NULL", domain.TicketStatusOpen).
		Order("id asc").
		Find(&items).Error
	return items, err
}

func (r *TicketRepository) ListByCSIDActive(ctx context.Context, csID uint64) ([]domain.Ticket, error) {
	var items []domain.Ticket
	err := r.db.WithContext(ctx).
		Where("assigned_cs_id = ? AND status IN (?, ?)", csID, domain.TicketStatusClaimed, domain.TicketStatusInProgress).
		Order("id desc").
		Find(&items).Error
	return items, err
}

func (r *TicketRepository) CountActiveByCSID(ctx context.Context, csID uint64) (int64, error) {
	var cnt int64
	err := r.db.WithContext(ctx).
		Model(&domain.Ticket{}).
		Where("assigned_cs_id = ? AND status IN (?, ?)", csID, domain.TicketStatusClaimed, domain.TicketStatusInProgress).
		Count(&cnt).Error
	return cnt, err
}

func (r *TicketRepository) AssignToCS(ctx context.Context, ticketID uint64, csID uint64) error {
	// Atomik: hanya assign jika status OPEN dan belum ada assigned CS.
	// Catatan: status langsung menjadi CLAIMED (state machine).
	res := r.db.WithContext(ctx).
		Model(&domain.Ticket{}).
		Where("id = ? AND status = ? AND assigned_cs_id IS NULL", ticketID, domain.TicketStatusOpen).
		Updates(map[string]any{"assigned_cs_id": csID, "status": domain.TicketStatusClaimed})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *TicketRepository) UpdateStatus(ctx context.Context, ticketID uint64, status string) error {
	return r.db.WithContext(ctx).Model(&domain.Ticket{}).Where("id = ?", ticketID).Update("status", status).Error
}
