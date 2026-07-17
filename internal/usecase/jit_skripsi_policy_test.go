package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/datatypes"

	"support-backend/internal/domain"
)

func TestJITRequestRequiresInProgress(t *testing.T) {
	ctx := context.Background()
	csID := uint64(10)
	ticketID := uint64(99)
	ticketRepo := newFakeTicketRepo(&domain.Ticket{
		ID:           ticketID,
		UserID:       20,
		AssignedCSID: &csID,
		Status:       domain.TicketStatusClaimed,
	})
	jitRepo := newFakeJITRepo()
	uc := NewJITUsecase(ticketRepo, jitRepo, newFakeAuditUsecase(), nil, nil, nil)

	_, err := uc.Request(ctx, csID, ticketID, domain.JITFeatureChangeEmail)
	if !errors.Is(err, ErrJITInvalidTicket) {
		t.Fatalf("expected ErrJITInvalidTicket for CLAIMED ticket, got %v", err)
	}
	if len(jitRepo.sessions) != 0 {
		t.Fatalf("expected no JIT session to be created, got %d", len(jitRepo.sessions))
	}

	ticketRepo.ticket.Status = domain.TicketStatusInProgress
	session, err := uc.Request(ctx, csID, ticketID, domain.JITFeatureChangeEmail)
	if err != nil {
		t.Fatalf("expected IN_PROGRESS ticket to receive JIT session, got %v", err)
	}
	if session == nil || session.Feature != domain.JITFeatureChangeEmail {
		t.Fatalf("unexpected JIT session: %#v", session)
	}
}

func TestJITRequestRejectsWrongAssignment(t *testing.T) {
	ctx := context.Background()
	assignedCSID := uint64(10)
	requesterCSID := uint64(11)
	ticketID := uint64(99)
	uc := NewJITUsecase(
		newFakeTicketRepo(&domain.Ticket{
			ID:           ticketID,
			UserID:       20,
			AssignedCSID: &assignedCSID,
			Status:       domain.TicketStatusInProgress,
		}),
		newFakeJITRepo(),
		newFakeAuditUsecase(),
		nil,
		nil,
		nil,
	)

	_, err := uc.Request(ctx, requesterCSID, ticketID, domain.JITFeatureChangeEmail)
	if !errors.Is(err, ErrJITForbidden) {
		t.Fatalf("expected ErrJITForbidden for wrong CS assignment, got %v", err)
	}
}

func TestJITFeatureWhitelistIncludesViewKYC(t *testing.T) {
	valid := []string{
		domain.JITFeatureResetPassword,
		domain.JITFeatureUnblockAccount,
		domain.JITFeatureChangeEmail,
		domain.JITFeatureResetPIN,
		domain.JITFeatureViewKYC,
	}
	for _, feature := range valid {
		if !domain.IsValidJITFeature(feature) {
			t.Fatalf("expected feature %s to be valid", feature)
		}
	}
	if domain.IsValidJITFeature("EXPORT_ALL_USERS") {
		t.Fatal("expected unknown feature to be invalid")
	}
}

func TestResetPasswordRequiresAndConsumesSingleUseJIT(t *testing.T) {
	ctx := context.Background()
	csID := uint64(10)
	userID := uint64(20)
	ticketID := uint64(99)
	ticketRepo := newFakeTicketRepo(&domain.Ticket{
		ID:           ticketID,
		UserID:       userID,
		AssignedCSID: &csID,
		Status:       domain.TicketStatusInProgress,
	})
	jitRepo := newFakeJITRepo()
	userRepo := &fakeUserRepo{users: map[uint64]*domain.User{
		userID: {ID: userID, Email: "user@example.com", Password: "old", Role: domain.RoleUser},
	}}
	jitUC := NewJITUsecase(ticketRepo, jitRepo, newFakeAuditUsecase(), nil, nil, nil)
	csUC := NewCSUsecase(userRepo, &fakeProfileRepo{}, ticketRepo, nil, nil, jitUC, newFakeAuditUsecase(), nil)

	if err := csUC.ResetPasswordByTicket(ctx, csID, ticketID, "newpassword123"); !errors.Is(err, ErrJITNotAvailable) {
		t.Fatalf("expected reset password without JIT to fail with ErrJITNotAvailable, got %v", err)
	}
	if userRepo.updatePasswordCount != 0 {
		t.Fatalf("expected password update not to run without JIT, got %d updates", userRepo.updatePasswordCount)
	}

	jitRepo.addActive(csID, ticketID, domain.JITFeatureResetPassword)
	if err := csUC.ResetPasswordByTicket(ctx, csID, ticketID, "newpassword123"); err != nil {
		t.Fatalf("expected reset password with active JIT to succeed, got %v", err)
	}
	if userRepo.updatePasswordCount != 1 {
		t.Fatalf("expected one password update, got %d", userRepo.updatePasswordCount)
	}
	if jitRepo.hasActive(csID, ticketID, domain.JITFeatureResetPassword) {
		t.Fatal("expected JIT session to be consumed after successful reset password")
	}

	if err := csUC.ResetPasswordByTicket(ctx, csID, ticketID, "anotherpassword123"); !errors.Is(err, ErrJITNotAvailable) {
		t.Fatalf("expected second reset password to fail after single-use JIT consumption, got %v", err)
	}
	if userRepo.updatePasswordCount != 1 {
		t.Fatalf("expected second attempt not to update password, got %d updates", userRepo.updatePasswordCount)
	}
}

func TestViewKYCConsumesSingleUseJIT(t *testing.T) {
	ctx := context.Background()
	csID := uint64(10)
	userID := uint64(20)
	ticketID := uint64(99)
	ticketRepo := newFakeTicketRepo(&domain.Ticket{
		ID:           ticketID,
		UserID:       userID,
		AssignedCSID: &csID,
		Status:       domain.TicketStatusInProgress,
	})
	jitRepo := newFakeJITRepo()
	profileRepo := &fakeProfileRepo{profiles: map[uint64]*domain.UserProfile{
		userID: {
			UserID:  userID,
			Phone:   "081234567890",
			Balance: 50000,
			KYCData: datatypes.JSON([]byte(`{"full_name":"Muhammad Syukur","nik":"3175091209990001","place_of_birth":"Jakarta","birth_date":"1999-09-12","address":"Jl. Merdeka Selatan No. 18","occupation":"Designer","monthly_income_range":"10-15 juta","recent_device":"iPhone","linked_bank":"Bank **** 1188","risk_score":"LOW"}`)),
		},
	}}
	jitUC := NewJITUsecase(ticketRepo, jitRepo, newFakeAuditUsecase(), nil, nil, nil)
	csUC := NewCSUsecase(&fakeUserRepo{}, profileRepo, ticketRepo, nil, nil, jitUC, newFakeAuditUsecase(), nil)

	locked, err := csUC.ViewUserProfileByTicket(ctx, csID, ticketID)
	if err != nil {
		t.Fatalf("expected locked profile view without JIT, got error %v", err)
	}
	if locked.ExposureState != "LOCKED" {
		t.Fatalf("expected profile to remain locked without JIT, got %s", locked.ExposureState)
	}

	jitRepo.addActive(csID, ticketID, domain.JITFeatureViewKYC)
	revealed, err := csUC.ViewUserProfileByTicket(ctx, csID, ticketID)
	if err != nil {
		t.Fatalf("expected profile view with VIEW_KYC JIT to succeed, got %v", err)
	}
	if revealed.ExposureState != "PARTIAL_AFTER_JIT" {
		t.Fatalf("expected partial profile after VIEW_KYC JIT, got %s", revealed.ExposureState)
	}
	if revealed.GrantedFeature != domain.JITFeatureViewKYC {
		t.Fatalf("expected granted feature VIEW_KYC, got %s", revealed.GrantedFeature)
	}
	if jitRepo.hasActive(csID, ticketID, domain.JITFeatureViewKYC) {
		t.Fatal("expected VIEW_KYC JIT session to be consumed")
	}

	lockedAgain, err := csUC.ViewUserProfileByTicket(ctx, csID, ticketID)
	if err != nil {
		t.Fatalf("expected second profile view after consume to return locked data, got %v", err)
	}
	if lockedAgain.ExposureState != "LOCKED" {
		t.Fatalf("expected second view to be locked after single-use consume, got %s", lockedAgain.ExposureState)
	}
}

func newFakeAuditUsecase() *AuditUsecase {
	return NewAuditUsecase(&fakeAuditRepo{}, fakeAuditPublisher{})
}

type fakeTicketRepo struct {
	ticket *domain.Ticket
}

func newFakeTicketRepo(ticket *domain.Ticket) *fakeTicketRepo {
	return &fakeTicketRepo{ticket: ticket}
}

func (r *fakeTicketRepo) Create(context.Context, *domain.Ticket) error { return nil }
func (r *fakeTicketRepo) GetByID(_ context.Context, id uint64) (*domain.Ticket, error) {
	if r.ticket == nil || r.ticket.ID != id {
		return nil, nil
	}
	return r.ticket, nil
}
func (r *fakeTicketRepo) ListByUserID(context.Context, uint64) ([]domain.Ticket, error) {
	return nil, nil
}
func (r *fakeTicketRepo) ListOpenUnassigned(context.Context) ([]domain.Ticket, error) {
	return nil, nil
}
func (r *fakeTicketRepo) ListByStatuses(context.Context, ...string) ([]domain.Ticket, error) {
	return nil, nil
}
func (r *fakeTicketRepo) ListByCSIDActive(context.Context, uint64) ([]domain.Ticket, error) {
	return nil, nil
}
func (r *fakeTicketRepo) CountActiveByCSID(context.Context, uint64) (int64, error) { return 0, nil }
func (r *fakeTicketRepo) CountByStatuses(context.Context, ...string) (int64, error) {
	return 0, nil
}
func (r *fakeTicketRepo) AssignToCS(context.Context, uint64, uint64) error { return nil }
func (r *fakeTicketRepo) UpdateStatus(context.Context, uint64, string) error {
	return nil
}

type fakeJITRepo struct {
	sessions []domain.JITSession
	nextID   uint64
}

func newFakeJITRepo() *fakeJITRepo {
	return &fakeJITRepo{nextID: 1}
}

func (r *fakeJITRepo) Create(_ context.Context, s *domain.JITSession) error {
	s.ID = r.nextID
	r.nextID++
	r.sessions = append(r.sessions, *s)
	return nil
}

func (r *fakeJITRepo) GetActive(_ context.Context, csID uint64, ticketID uint64, feature string, now time.Time) (*domain.JITSession, error) {
	for i := len(r.sessions) - 1; i >= 0; i-- {
		s := r.sessions[i]
		if s.CSID == csID && s.TicketID == ticketID && s.Feature == feature && s.IsActive && s.ExpiredAt.After(now) {
			return &r.sessions[i], nil
		}
	}
	return nil, nil
}

func (r *fakeJITRepo) RevokeExpired(_ context.Context, now time.Time) error {
	for i := range r.sessions {
		if r.sessions[i].IsActive && !r.sessions[i].ExpiredAt.After(now) {
			r.sessions[i].IsActive = false
		}
	}
	return nil
}

func (r *fakeJITRepo) RevokeByTicket(_ context.Context, ticketID uint64) error {
	for i := range r.sessions {
		if r.sessions[i].TicketID == ticketID {
			r.sessions[i].IsActive = false
		}
	}
	return nil
}

func (r *fakeJITRepo) RevokeExisting(_ context.Context, csID uint64, ticketID uint64, feature string) error {
	for i := range r.sessions {
		if r.sessions[i].CSID == csID && r.sessions[i].TicketID == ticketID && r.sessions[i].Feature == feature {
			r.sessions[i].IsActive = false
		}
	}
	return nil
}

func (r *fakeJITRepo) RevokeByCS(_ context.Context, csID uint64) error {
	for i := range r.sessions {
		if r.sessions[i].CSID == csID {
			r.sessions[i].IsActive = false
		}
	}
	return nil
}

func (r *fakeJITRepo) addActive(csID uint64, ticketID uint64, feature string) {
	_ = r.Create(context.Background(), &domain.JITSession{
		CSID:      csID,
		TicketID:  ticketID,
		Feature:   feature,
		ExpiredAt: time.Now().Add(15 * time.Minute),
		IsActive:  true,
	})
}

func (r *fakeJITRepo) hasActive(csID uint64, ticketID uint64, feature string) bool {
	s, _ := r.GetActive(context.Background(), csID, ticketID, feature, time.Now())
	return s != nil
}

type fakeUserRepo struct {
	users               map[uint64]*domain.User
	updatePasswordCount int
	updateEmailCount    int
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}
func (r *fakeUserRepo) GetByID(_ context.Context, id uint64) (*domain.User, error) {
	return r.users[id], nil
}
func (r *fakeUserRepo) List(context.Context) ([]domain.User, error) { return nil, nil }
func (r *fakeUserRepo) CountByRole(context.Context, string) (int64, error) {
	return 0, nil
}
func (r *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	if r.users == nil {
		r.users = map[uint64]*domain.User{}
	}
	r.users[u.ID] = u
	return nil
}
func (r *fakeUserRepo) UpdateEmail(_ context.Context, userID uint64, newEmail string) error {
	r.updateEmailCount++
	if r.users[userID] != nil {
		r.users[userID].Email = newEmail
	}
	return nil
}
func (r *fakeUserRepo) UpdatePassword(_ context.Context, userID uint64, newHashedPassword string) error {
	r.updatePasswordCount++
	if r.users[userID] != nil {
		r.users[userID].Password = newHashedPassword
	}
	return nil
}

type fakeProfileRepo struct {
	profiles map[uint64]*domain.UserProfile
}

func (r *fakeProfileRepo) GetByUserID(_ context.Context, userID uint64) (*domain.UserProfile, error) {
	return r.profiles[userID], nil
}
func (r *fakeProfileRepo) Create(_ context.Context, p *domain.UserProfile) error {
	if r.profiles == nil {
		r.profiles = map[uint64]*domain.UserProfile{}
	}
	r.profiles[p.UserID] = p
	return nil
}
func (r *fakeProfileRepo) UpdateKYCData(_ context.Context, userID uint64, kyc datatypes.JSON) error {
	if r.profiles[userID] == nil {
		return errors.New("profile not found")
	}
	r.profiles[userID].KYCData = kyc
	return nil
}

type fakeAuditRepo struct{}

func (r *fakeAuditRepo) Create(context.Context, *domain.AuditLog) error { return nil }
func (r *fakeAuditRepo) List(context.Context, string, int) ([]domain.AuditLog, error) {
	return nil, nil
}
func (r *fakeAuditRepo) ListByTicketID(context.Context, uint64, int) ([]domain.AuditLog, error) {
	return nil, nil
}

type fakeAuditPublisher struct{}

func (fakeAuditPublisher) PublishAdminAudit(any) {}
