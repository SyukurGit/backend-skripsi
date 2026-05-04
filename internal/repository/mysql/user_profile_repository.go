package mysql

import (
	"context"
	"errors"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"support-backend/internal/domain"
)

type UserProfileRepository struct {
	db *gorm.DB
}

func NewUserProfileRepository(db *gorm.DB) *UserProfileRepository {
	return &UserProfileRepository{db: db}
}

func (r *UserProfileRepository) GetByUserID(ctx context.Context, userID uint64) (*domain.UserProfile, error) {
	var p domain.UserProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *UserProfileRepository) UpdateKYCData(ctx context.Context, userID uint64, kyc datatypes.JSON) error {
	return r.db.WithContext(ctx).Model(&domain.UserProfile{}).Where("user_id = ?", userID).Update("kyc_data", kyc).Error
}
