package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

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
	msgRepo     domain.MessageRepository
	chatPub     domain.ChatEventPublisher
	jitUC       *JITUsecase
	auditUC     *AuditUsecase
	terminalPub domain.TerminalLogPublisher
}

func NewCSUsecase(userRepo domain.UserRepository, profileRepo domain.UserProfileRepository, ticketRepo domain.TicketRepository, msgRepo domain.MessageRepository, chatPub domain.ChatEventPublisher, jitUC *JITUsecase, auditUC *AuditUsecase, terminalPub domain.TerminalLogPublisher) *CSUsecase {
	return &CSUsecase{userRepo: userRepo, profileRepo: profileRepo, ticketRepo: ticketRepo, msgRepo: msgRepo, chatPub: chatPub, jitUC: jitUC, auditUC: auditUC, terminalPub: terminalPub}
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
	// Session JIT harus valid dan langsung dikonsumsi sebelum aksi sensitif dijalankan.
	if err := u.jitUC.Consume(ctx, csID, ticketID, domain.JITFeatureResetPassword); err != nil {
		return err
	}
	if err := u.userRepo.UpdatePassword(ctx, t.UserID, hashed); err != nil {
		return err
	}
	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "RESET_PASSWORD", &ticketID, map[string]any{"target_user_id": t.UserID})
	u.publishTerminal(ticketID, "INFO", "cs_usecase", "sensitive action executed; password reset completed")
	u.publishSystemNotice(ctx, ticketID, "Sistem: Customer Service menjalankan reset password melalui akses sementara yang telah disetujui.")
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
	if err := u.jitUC.Consume(ctx, csID, ticketID, domain.JITFeatureChangeEmail); err != nil {
		return err
	}
	if err := u.userRepo.UpdateEmail(ctx, t.UserID, newEmail); err != nil {
		return err
	}
	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "CHANGE_EMAIL", &ticketID, map[string]any{"target_user_id": t.UserID})
	u.publishTerminal(ticketID, "INFO", "cs_usecase", "sensitive action executed; email updated")
	u.publishSystemNotice(ctx, ticketID, "Sistem: Customer Service mengubah email akun melalui akses sementara yang telah disetujui.")
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
	if err := u.jitUC.Consume(ctx, csID, ticketID, domain.JITFeatureUnblockAccount); err != nil {
		return err
	}
	if err := u.updateProfileKYCField(ctx, t.UserID, "is_blocked", false); err != nil {
		return err
	}
	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "UNBLOCK_ACCOUNT", &ticketID, map[string]any{"target_user_id": t.UserID})
	u.publishTerminal(ticketID, "INFO", "cs_usecase", "sensitive action executed; account unblock completed")
	u.publishSystemNotice(ctx, ticketID, "Sistem: Customer Service membuka blokir akun melalui akses sementara yang telah disetujui.")
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
	if err := u.jitUC.Consume(ctx, csID, ticketID, domain.JITFeatureResetPIN); err != nil {
		return err
	}
	if err := u.updateProfileKYCField(ctx, t.UserID, "pin_hash", hashed); err != nil {
		return err
	}
	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "RESET_PIN", &ticketID, map[string]any{"target_user_id": t.UserID})
	u.publishTerminal(ticketID, "INFO", "cs_usecase", "sensitive action executed; pin reset completed")
	u.publishSystemNotice(ctx, ticketID, "Sistem: Customer Service menjalankan reset PIN melalui akses sementara yang telah disetujui.")
	return nil
}

func (u *CSUsecase) ViewUserProfileByTicket(ctx context.Context, csID uint64, ticketID uint64) (*domain.UserProfileMaskedView, error) {
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
		UserID:        p.UserID,
		Phone:         "",
		Balance:       0,
		KYCData:       hiddenKYC(),
		ExposureState: "LOCKED",
		PolicyNote:    "Data pengguna belum ditampilkan. Customer Service harus mengajukan akses sementara terlebih dahulu.",
	}

	// Full data hanya saat JIT aktif untuk VIEW_KYC.
	// CEK JIT: memastikan akses hanya berlaku dalam waktu terbatas
	if u.jitUC != nil {
		if err := u.jitUC.Consume(ctx, csID, ticketID, domain.JITFeatureViewKYC); err == nil {
			view.Phone = mask.Phone(p.Phone)
			view.Balance = p.Balance
			view.KYCData = revealedKYC(p.KYCData)
			view.ExposureState = "PARTIAL_AFTER_JIT"
			view.PolicyNote = "Akses sementara disetujui. Sistem hanya membuka sebagian data yang relevan dan tetap mempertahankan masking pada elemen sensitif tertentu."
			view.GrantedFeature = domain.JITFeatureViewKYC
			_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "VIEW_KYC", &ticketID, map[string]any{"target_user_id": t.UserID})
			u.publishTerminal(ticketID, "INFO", "cs_usecase", "sensitive data view opened; full kyc released within active jit session")
			u.publishSystemNotice(ctx, ticketID, "Sistem: Customer Service membuka data KYC pengguna melalui akses sementara yang telah disetujui.")
		}
	}

	return &view, nil
}

func (u *CSUsecase) publishTerminal(ticketID uint64, level, source, message string) {
	if u.terminalPub == nil {
		return
	}
	u.terminalPub.PublishTicketTerminal(ticketID, domain.TerminalLogEntry{TicketID: ticketID, Timestamp: time.Now(), Level: level, Source: source, Message: message})
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

func hiddenKYC() datatypes.JSON {
	return datatypes.JSON([]byte(`{"full_name":"TERSEMBUNYI","nik":"TERSEMBUNYI","birth_profile":"TERSEMBUNYI","address":"TERSEMBUNYI","occupation":"TERSEMBUNYI","risk_score":"TERSEMBUNYI","linked_bank":"TERSEMBUNYI"}`))
}

func revealedKYC(kyc datatypes.JSON) datatypes.JSON {
	if len(kyc) == 0 {
		return hiddenKYC()
	}
	var obj map[string]any
	if err := json.Unmarshal(kyc, &obj); err != nil {
		return hiddenKYC()
	}
	masked := map[string]any{
		"full_name":            maskMiddleString(asString(obj["full_name"])),
		"nik":                  maskLastDigits(asString(obj["nik"]), 4),
		"birth_profile":        asString(obj["place_of_birth"]) + ", " + asString(obj["birth_date"]),
		"address":              maskAddress(asString(obj["address"])),
		"occupation":           asString(obj["occupation"]),
		"monthly_income_range": asString(obj["monthly_income_range"]),
		"recent_device":        asString(obj["recent_device"]),
		"linked_bank":          asString(obj["linked_bank"]),
		"risk_score":           asString(obj["risk_score"]),
	}
	b, _ := json.Marshal(masked)
	return datatypes.JSON(b)
}

func (u *CSUsecase) publishSystemNotice(ctx context.Context, ticketID uint64, text string) {
	if u.msgRepo == nil || u.chatPub == nil {
		return
	}
	m := domain.Message{TicketID: ticketID, SenderID: 0, Message: text, CreatedAt: time.Now()}
	if err := u.msgRepo.Create(ctx, &m); err != nil {
		return
	}
	u.chatPub.PublishTicketMessage(ticketID, map[string]any{
		"id":          m.ID,
		"ticket_id":   ticketID,
		"sender_id":   0,
		"sender_role": "system",
		"message":     text,
		"created_at":  m.CreatedAt,
	})
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func maskMiddleString(s string) string {
	if len(s) <= 4 {
		return "***"
	}
	return s[:2] + "***" + s[len(s)-2:]
}

func maskLastDigits(s string, keep int) string {
	if len(s) <= keep {
		return "***"
	}
	return "********" + s[len(s)-keep:]
}

func maskAddress(s string) string {
	if len(s) <= 12 {
		return "Alamat tersaring"
	}
	return s[:12] + " ..."
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
