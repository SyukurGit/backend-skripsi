package usecase

import (
	"context"
	"errors"
	"time"

	"support-backend/config"
	"support-backend/internal/domain"
	"support-backend/pkg/jwt"
	"support-backend/pkg/password"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthUsecase struct {
	userRepo    domain.UserRepository
	profileRepo domain.UserProfileRepository
	cfg         config.Config
}

func NewAuthUsecase(userRepo domain.UserRepository, profileRepo domain.UserProfileRepository, cfg config.Config) *AuthUsecase {
	return &AuthUsecase{userRepo: userRepo, profileRepo: profileRepo, cfg: cfg}
}

func (u *AuthUsecase) Login(ctx context.Context, email, plainPassword string) (string, *domain.User, error) {
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}
	if user == nil {
		return "", nil, ErrInvalidCredentials
	}
	if err := password.Compare(user.Password, plainPassword); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := jwt.GenerateToken(u.cfg.JWTSecret, user.ID, user.Role, user.Email, 24*time.Hour)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}
