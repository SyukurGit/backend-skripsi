package usecase

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/datatypes"

	"support-backend/internal/domain"
	"support-backend/pkg/mask"
	"support-backend/pkg/password"
)

var (
	ErrCSLeastPrivilegeViolation = errors.New("least privilege violation")
)

type CSUsecase struct {
	userRepo    domain.UserRepository
	profileRepo domain.UserProfileRepository
	ticketRepo  domain.TicketRepository
	jitUC       *JITUsecase
	auditUC     *AuditUsecase
}

func NewCSUsecase(userRepo domain.UserRepository, profileRepo domain.UserProfileRepository, ticketRepo domain.TicketRepository, jitUC *JITUsecase, auditUC *AuditUsecase) *CSUsecase {
	return &CSUsecase{userRepo: userRepo, profileRepo: profileRepo, ticketRepo: ticketRepo, jitUC: jitUC, auditUC: auditUC}
}

func (u *CSUsecase) ResetPasswordByTicket(ctx context.Context, csID uint64, ticketID uint64, newPassword string) error {
	// VALIDASI: memastikan CS hanya bisa akses user dari ticket yang dia claim.
	t, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return err
	}
	if t == nil || t.AssignedCSID == nil || *t.AssignedCSID != csID {
		return ErrCSLeastPrivilegeViolation
	}

	hashed, err := password.Hash(newPassword)
	if err != nil {
		return err
	}
	if err := u.userRepo.UpdatePassword(ctx, t.UserID, hashed); err != nil {
		return err
	}
	// Revoke JIT setelah eksekusi sukses.
	_ = u.jitUC.Consume(ctx, csID, ticketID, domain.JITFeatureResetPassword)
	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "RESET_PASSWORD", &ticketID, map[string]any{"target_user_id": t.UserID})
	return nil
}

func (u *CSUsecase) ChangeEmailByTicket(ctx context.Context, csID uint64, ticketID uint64, newEmail string) error {
	// VALIDASI: memastikan CS hanya bisa akses user dari ticket yang dia claim.
	t, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return err
	}
	if t == nil || t.AssignedCSID == nil || *t.AssignedCSID != csID {
		return ErrCSLeastPrivilegeViolation
	}
	if err := u.userRepo.UpdateEmail(ctx, t.UserID, newEmail); err != nil {
		return err
	}
	_ = u.jitUC.Consume(ctx, csID, ticketID, domain.JITFeatureChangeEmail)
	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "CHANGE_EMAIL", &ticketID, map[string]any{"target_user_id": t.UserID})
	return nil
}

func (u *CSUsecase) UnblockAccountByTicket(ctx context.Context, csID uint64, ticketID uint64) error {
	// VALIDASI: memastikan CS hanya bisa akses user dari ticket yang dia claim.
	t, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return err
	}
	if t == nil || t.AssignedCSID == nil || *t.AssignedCSID != csID {
		return ErrCSLeastPrivilegeViolation
	}
	if err := u.updateProfileKYCField(ctx, t.UserID, "is_blocked", false); err != nil {
		return err
	}
	_ = u.jitUC.Consume(ctx, csID, ticketID, domain.JITFeatureUnblockAccount)
	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "UNBLOCK_ACCOUNT", &ticketID, map[string]any{"target_user_id": t.UserID})
	return nil
}

func (u *CSUsecase) ResetPINByTicket(ctx context.Context, csID uint64, ticketID uint64, newPIN string) error {
	// VALIDASI: memastikan CS hanya bisa akses user dari ticket yang dia claim.
	t, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return err
	}
	if t == nil || t.AssignedCSID == nil || *t.AssignedCSID != csID {
		return ErrCSLeastPrivilegeViolation
	}

	hashed, err := password.Hash(newPIN)
	if err != nil {
		return err
	}
	if err := u.updateProfileKYCField(ctx, t.UserID, "pin_hash", hashed); err != nil {
		return err
	}
	_ = u.jitUC.Consume(ctx, csID, ticketID, domain.JITFeatureResetPIN)
	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "RESET_PIN", &ticketID, map[string]any{"target_user_id": t.UserID})
	return nil
}

func (u *CSUsecase) ViewUserProfileByTicket(ctx context.Context, csID uint64, ticketID uint64) (*domain.UserProfile, error) {
	// VALIDASI: memastikan CS hanya bisa akses user dari ticket yang dia claim.
	t, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil || t.AssignedCSID == nil || *t.AssignedCSID != csID {
		return nil, ErrCSLeastPrivilegeViolation
	}
	// VALIDASI: masking data untuk mencegah kebocoran informasi sensitif
	p, err := u.profileRepo.GetByUserID(ctx, t.UserID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}

	// Default: phone dimasking dan KYC hanya partial.
	view := domain.UserProfileMaskedView{
		UserID:  p.UserID,
		Phone:   mask.Phone(p.Phone),
		Balance: p.Balance,
		KYCData: partialKYC(p.KYCData),
	}

	// Full data hanya saat JIT aktif untuk VIEW_KYC.
	// CEK JIT: memastikan akses hanya berlaku dalam waktu terbatas
	if u.jitUC != nil {
		if err := u.jitUC.EnsureValid(ctx, csID, ticketID, "VIEW_KYC"); err == nil {
			view.Phone = p.Phone
			view.KYCData = p.KYCData
			// Revoke JIT setelah full view berhasil diberikan.
			_ = u.jitUC.Consume(ctx, csID, ticketID, "VIEW_KYC")
		}
	}

	// Karena signature sudah terlanjur, kita map ke domain.UserProfile minimal.
	return &domain.UserProfile{UserID: view.UserID, Phone: view.Phone, Balance: view.Balance, KYCData: view.KYCData}, nil
}

func partialKYC(kyc datatypes.JSON) datatypes.JSON {
	// VALIDASI: masking data untuk mencegah kebocoran informasi sensitif
	if len(kyc) == 0 {
		return kyc
	}
	var obj map[string]any
	if err := json.Unmarshal(kyc, &obj); err != nil {
		return datatypes.JSON([]byte(`{}`))
	}
	allowed := map[string]bool{
		"kyc_status": true,
		"is_blocked": true,
		"tier":       true,
		"department": true,
	}
	res := map[string]any{}
	for k, v := range obj {
		if allowed[k] {
			res[k] = v
		}
	}
	b, _ := json.Marshal(res)
	return datatypes.JSON(b)
}

func (u *CSUsecase) updateProfileKYCField(ctx context.Context, userID uint64, key string, value any) error {
	p, err := u.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("profile not found")
	}

	var obj map[string]any
	if len(p.KYCData) > 0 {
		_ = json.Unmarshal(p.KYCData, &obj)
	}
	if obj == nil {
		obj = map[string]any{}
	}
	obj[key] = value
	b, _ := json.Marshal(obj)
	return u.profileRepo.UpdateKYCData(ctx, userID, datatypes.JSON(b))
}
