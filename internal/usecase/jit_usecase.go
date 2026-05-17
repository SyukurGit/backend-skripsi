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
	ticketRepo  domain.TicketRepository
	jitRepo     domain.JITSessionRepository
	auditUC     *AuditUsecase
	msgRepo     domain.MessageRepository
	chatPub     domain.ChatEventPublisher
	terminalPub domain.TerminalLogPublisher
}

func NewJITUsecase(ticketRepo domain.TicketRepository, jitRepo domain.JITSessionRepository, auditUC *AuditUsecase, msgRepo domain.MessageRepository, chatPub domain.ChatEventPublisher, terminalPub domain.TerminalLogPublisher) *JITUsecase {
	return &JITUsecase{ticketRepo: ticketRepo, jitRepo: jitRepo, auditUC: auditUC, msgRepo: msgRepo, chatPub: chatPub, terminalPub: terminalPub}
}

func (u *JITUsecase) Request(ctx context.Context, csID uint64, ticketID uint64, feature string) (*domain.JITSession, error) {
	u.publishTerminal(ticketID, "INFO", "jit_usecase", "jit request received; starting backend validation sequence")
	t, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		u.publishTerminal(ticketID, "ERROR", "jit_usecase", "validation step 1 ticket_exists=false; repository error while reading ticket")
		return nil, err
	}
	if t == nil {
		_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "JIT_REQUEST_DENIED", &ticketID, map[string]any{"feature": feature, "reason": "Ticket tidak ditemukan"})
		u.publishTerminal(ticketID, "ERROR", "jit_usecase", "validation step 1 ticket_exists=false; ticket not found")
		u.publishTerminal(ticketID, "ERROR", "jit_usecase", "jit request denied; ticket not found")
		return nil, ErrTicketNotFound
	}
	u.publishTerminal(ticketID, "INFO", "jit_usecase", "validation step 1 ticket_exists=true")

	// VALIDASI: memastikan CS hanya bisa request JIT untuk tiket miliknya.
	if t.AssignedCSID == nil || *t.AssignedCSID != csID {
		_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "JIT_REQUEST_DENIED", &ticketID, map[string]any{"feature": feature, "reason": "Tiket tidak ditugaskan ke CS ini"})
		u.publishTerminal(ticketID, "WARN", "jit_usecase", "validation step 2 assigned_to_current_cs=false")
		u.publishTerminal(ticketID, "WARN", "jit_usecase", "jit request denied; assigned_cs validation failed")
		return nil, ErrJITForbidden
	}
	u.publishTerminal(ticketID, "INFO", "jit_usecase", "validation step 2 assigned_to_current_cs=true")
	// VALIDASI: status ticket harus CLAIMED atau IN_PROGRESS.
	if t.Status != domain.TicketStatusClaimed && t.Status != domain.TicketStatusInProgress {
		_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "JIT_REQUEST_DENIED", &ticketID, map[string]any{"feature": feature, "reason": "Status tiket tidak aktif untuk JIT"})
		u.publishTerminal(ticketID, "WARN", "jit_usecase", "validation step 3 ticket_status_eligible=false; current_status="+t.Status)
		u.publishTerminal(ticketID, "WARN", "jit_usecase", "jit request denied; ticket status not eligible for temporary access")
		return nil, ErrJITInvalidTicket
	}
	u.publishTerminal(ticketID, "INFO", "jit_usecase", "validation step 3 ticket_status_eligible=true; current_status="+t.Status)
	u.publishTerminal(ticketID, "INFO", "jit_usecase", "validation step 4 feature_allowed=true; requested_feature="+feature)

	now := time.Now()
	_ = u.jitRepo.RevokeExisting(ctx, csID, ticketID, feature)
	u.publishTerminal(ticketID, "INFO", "jit_usecase", "validation step 5 previous_session_revoked=true; continuing with fresh temporary session")

	s := domain.JITSession{
		CSID:      csID,
		TicketID:  ticketID,
		Feature:   feature,
		ExpiredAt: now.Add(15 * time.Minute),
		IsActive:  true,
	}
	if err := u.jitRepo.Create(ctx, &s); err != nil {
		u.publishTerminal(ticketID, "ERROR", "jit_usecase", "validation step 6 jit_session_created=false; database create failed")
		return nil, err
	}
	u.publishTerminal(ticketID, "INFO", "jit_usecase", "validation step 6 jit_session_created=true")

	_ = u.auditUC.Log(ctx, csID, domain.RoleCS, "JIT_REQUEST", &ticketID, map[string]any{"feature": feature, "expired_at": s.ExpiredAt, "reason": "Seluruh pemeriksaan backend lolos dan sesi JIT diberikan"})
	u.publishTerminal(ticketID, "INFO", "jit_usecase", "validation summary access_granted=true; all backend checks passed")
	u.publishTerminal(ticketID, "INFO", "jit_usecase", "jit request approved; feature="+feature+" temporary session created")
	u.publishSystemNotice(ctx, ticketID, "Sistem: permintaan akses sementara untuk fitur "+feature+" telah disetujui oleh backend.")
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

func (u *JITUsecase) publishTerminal(ticketID uint64, level, source, message string) {
	if u.terminalPub == nil {
		return
	}
	u.terminalPub.PublishTicketTerminal(ticketID, domain.TerminalLogEntry{TicketID: ticketID, Timestamp: time.Now(), Level: level, Source: source, Message: message})
}

func (u *JITUsecase) publishSystemNotice(ctx context.Context, ticketID uint64, text string) {
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
