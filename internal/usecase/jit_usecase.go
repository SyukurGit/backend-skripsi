package usecase

import (
	"context"
	"errors"
	"time"

	"support-backend/internal/domain"
)

var (
	ErrJITForbidden     = errors.New("jit forbidden")
	ErrJITNotAvailable  = errors.New("jit not available")
	ErrJITInvalidTicket = errors.New("jit invalid ticket")
)

type JITUsecase struct {
	ticketRepo domain.TicketRepository
	jitRepo    domain.JITSessionRepository
	auditUC    *AuditUsecase
}

func NewJITUsecase(ticketRepo domain.TicketRepository, jitRepo domain.JITSessionRepository, auditUC *AuditUsecase) *JITUsecase {
	return &JITUsecase{ticketRepo: ticketRepo, jitRepo: jitRepo, auditUC: auditUC}
}

func (u *JITUsecase) Request(ctx context.Context, csID uint64, ticketID uint64, feature string) (*domain.JITSession, error) {
	t, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTicketNotFound
	}

	// VALIDASI: memastikan CS hanya bisa request JIT untuk tiket miliknya.
	if t.AssignedCSID == nil || *t.AssignedCSID != csID {
		return nil, ErrJITForbidden
	}
	// VALIDASI: status ticket harus CLAIMED atau IN_PROGRESS.
	if t.Status != domain.TicketStatusClaimed && t.Status != domain.TicketStatusInProgress {
		return nil, ErrJITInvalidTicket
	}

	now := time.Now()
	_ = u.jitRepo.RevokeExisting(ctx, csID, ticketID, feature)

	s := domain.JITSession{
		CSID:      csID,
		TicketID:  ticketID,
		Feature:   feature,
		ExpiredAt: now.Add(15 * time.Minute),
		IsActive:  true,
	}
	if err := u.jitRepo.Create(ctx, &s); err != nil {
		return nil, err
	}

	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "JIT_REQUEST", &ticketID, map[string]any{"feature": feature, "expired_at": s.ExpiredAt})
	return &s, nil
}

func (u *JITUsecase) EnsureValid(ctx context.Context, csID uint64, ticketID uint64, feature string) error {
	now := time.Now()
	// CEK JIT: memastikan akses hanya berlaku dalam waktu terbatas
	// CEK JIT: revoke otomatis session yang sudah expired.
	_ = u.jitRepo.RevokeExpired(ctx, now)

	s, err := u.jitRepo.GetActive(ctx, csID, ticketID, feature, now)
	if err != nil {
		return err
	}
	if s == nil {
		return ErrJITNotAvailable
	}
	return nil
}

func (u *JITUsecase) Consume(ctx context.Context, csID uint64, ticketID uint64, feature string) error {
	// CEK JIT: memastikan akses hanya berlaku dalam waktu terbatas
	if err := u.EnsureValid(ctx, csID, ticketID, feature); err != nil {
		return err
	}
	// Setelah aksi sensitif sukses, session harus direvoke.
	return u.jitRepo.RevokeExisting(ctx, csID, ticketID, feature)
}

func (u *JITUsecase) RevokeOnLogout(ctx context.Context, csID uint64) error {
	// CEK JIT: memastikan akses hanya berlaku dalam waktu terbatas
	// Saat logout, revoke semua JIT session aktif untuk CS tersebut.
	return u.jitRepo.RevokeByCS(ctx, csID)
}
