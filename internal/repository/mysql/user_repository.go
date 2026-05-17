package mysql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"support-backend/internal/domain"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uint64) (*domain.User, error) {
	var u domain.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	var items []domain.User
	err := r.db.WithContext(ctx).Order("id asc").Find(&items).Error
	return items, err
}

func (r *UserRepository) CountByRole(ctx context.Context, role string) (int64, error) {
	var cnt int64
	err := r.db.WithContext(ctx).Model(&domain.User{}).Where("role = ?", role).Count(&cnt).Error
	return cnt, err
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *UserRepository) UpdateEmail(ctx context.Context, userID uint64, newEmail string) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Update("email", newEmail).Error
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uint64, newHashedPassword string) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Update("password", newHashedPassword).Error
}
