package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/datatypes"

	"support-backend/internal/domain"
	"support-backend/pkg/password"
)

var ErrAdminManagedRoleInvalid = errors.New("managed role invalid")

type AdminUsecase struct {
	userRepo    domain.UserRepository
	profileRepo domain.UserProfileRepository
	ticketRepo  domain.TicketRepository
	auditRepo   domain.AuditLogRepository
	terminalPub domain.TerminalLogPublisher
}

func NewAdminUsecase(userRepo domain.UserRepository, profileRepo domain.UserProfileRepository, ticketRepo domain.TicketRepository, auditRepo domain.AuditLogRepository, terminalPub domain.TerminalLogPublisher) *AdminUsecase {
	return &AdminUsecase{userRepo: userRepo, profileRepo: profileRepo, ticketRepo: ticketRepo, auditRepo: auditRepo, terminalPub: terminalPub}
}

func (u *AdminUsecase) DashboardStats(ctx context.Context) (*domain.AdminDashboardStats, error) {
	totalUsers, err := u.userRepo.CountByRole(ctx, domain.RoleUser)
	if err != nil {
		return nil, err
	}
	totalCS, err := u.userRepo.CountByRole(ctx, domain.RoleCS)
	if err != nil {
		return nil, err
	}
	totalAdmins, err := u.userRepo.CountByRole(ctx, domain.RoleAdmin)
	if err != nil {
		return nil, err
	}
	inProcess, err := u.ticketRepo.CountByStatuses(ctx, domain.TicketStatusClaimed, domain.TicketStatusInProgress)
	if err != nil {
		return nil, err
	}
	unassigned, err := u.ticketRepo.CountByStatuses(ctx, domain.TicketStatusOpen)
	if err != nil {
		return nil, err
	}
	resolved, err := u.ticketRepo.CountByStatuses(ctx, domain.TicketStatusResolved)
	if err != nil {
		return nil, err
	}
	closed, err := u.ticketRepo.CountByStatuses(ctx, domain.TicketStatusClosed)
	if err != nil {
		return nil, err
	}
	audits, err := u.auditRepo.List(ctx, "", 500)
	if err != nil {
		return nil, err
	}

	stats := &domain.AdminDashboardStats{
		TotalUsers:        totalUsers,
		TotalCS:           totalCS,
		TotalAdmins:       totalAdmins,
		TicketsInProcess:  inProcess,
		TicketsUnassigned: unassigned,
		TicketsResolved:   resolved,
		TicketsClosed:     closed,
	}
	for _, item := range audits {
		switch item.Action {
		case "RESET_PASSWORD", "CHANGE_EMAIL", "UNBLOCK_ACCOUNT", "RESET_PIN":
			stats.SensitiveActions++
		case "JIT_REQUEST":
			stats.PendingJITRequests++
		}
	}
	return stats, nil
}

func (u *AdminUsecase) ListSessions(ctx context.Context) ([]domain.AdminSessionListItem, error) {
	tickets, err := u.ticketRepo.ListByStatuses(ctx, domain.TicketStatusOpen, domain.TicketStatusClaimed, domain.TicketStatusInProgress, domain.TicketStatusResolved, domain.TicketStatusClosed)
	if err != nil {
		return nil, err
	}
	items := make([]domain.AdminSessionListItem, 0, len(tickets))
	for _, ticket := range tickets {
		audits, err := u.auditRepo.ListByTicketID(ctx, ticket.ID, 200)
		if err != nil {
			return nil, err
		}
		var assignedCSEmail *string
		var claimedAt *time.Time
		var lastActivityAt *time.Time
		sensitiveActions := 0
		jitAttempts := 0
		if ticket.AssignedCSID != nil {
			cs, err := u.userRepo.GetByID(ctx, *ticket.AssignedCSID)
			if err != nil {
				return nil, err
			}
			if cs != nil {
				assignedCSEmail = &cs.Email
			}
		}
		for _, audit := range audits {
			t := audit.CreatedAt
			if lastActivityAt == nil || t.After(*lastActivityAt) {
				lastActivityAt = &t
			}
			switch audit.Action {
			case "TICKET_CLAIM":
				if claimedAt == nil {
					claimedAt = &t
				}
			case "JIT_REQUEST", "JIT_REQUEST_DENIED":
				jitAttempts++
			case "RESET_PASSWORD", "CHANGE_EMAIL", "UNBLOCK_ACCOUNT", "RESET_PIN", "VIEW_KYC":
				sensitiveActions++
			}
		}
		items = append(items, domain.AdminSessionListItem{
			TicketID:         ticket.ID,
			UserID:           ticket.UserID,
			TicketStatus:     ticket.Status,
			CreatedAt:        ticket.CreatedAt,
			AssignedCSID:     ticket.AssignedCSID,
			AssignedCSEmail:  assignedCSEmail,
			ClaimedAt:        claimedAt,
			LastActivityAt:   lastActivityAt,
			SensitiveActions: sensitiveActions,
			JITAttempts:      jitAttempts,
		})
	}
	return items, nil
}

func (u *AdminUsecase) SessionDetail(ctx context.Context, ticketID uint64) (*domain.AdminSessionDetail, error) {
	ticket, err := u.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, ErrTicketNotFound
	}
	audits, err := u.auditRepo.ListByTicketID(ctx, ticketID, 300)
	if err != nil {
		return nil, err
	}
	user, err := u.userRepo.GetByID(ctx, ticket.UserID)
	if err != nil {
		return nil, err
	}
	var userEmail *string
	if user != nil {
		userEmail = &user.Email
	}
	var assignedCSEmail *string
	if ticket.AssignedCSID != nil {
		cs, err := u.userRepo.GetByID(ctx, *ticket.AssignedCSID)
		if err != nil {
			return nil, err
		}
		if cs != nil {
			assignedCSEmail = &cs.Email
		}
	}
	var claimedAt *time.Time
	jitAttempts := make([]domain.AdminJITAttempt, 0)
	for _, audit := range audits {
		t := audit.CreatedAt
		if audit.Action == "TICKET_CLAIM" && claimedAt == nil {
			claimedAt = &t
		}
		if audit.Action == "JIT_REQUEST" || audit.Action == "JIT_REQUEST_DENIED" {
			feature := "-"
			reason := "Seluruh pemeriksaan backend dinyatakan lolos dan sesi JIT dibentuk sementara."
			if meta, ok := datatypesJSONToMap(audit.Metadata); ok {
				if v, ok := meta["feature"].(string); ok && v != "" {
					feature = v
				}
				if v, ok := meta["reason"].(string); ok && v != "" {
					reason = v
				}
			}
			jitAttempts = append(jitAttempts, domain.AdminJITAttempt{
				RequestedAt: audit.CreatedAt,
				Feature:     feature,
				Granted:     audit.Action == "JIT_REQUEST",
				Reason:      reason,
			})
		}
	}
	return &domain.AdminSessionDetail{
		TicketID:        ticket.ID,
		UserID:          ticket.UserID,
		UserEmail:       userEmail,
		TicketStatus:    ticket.Status,
		CreatedAt:       ticket.CreatedAt,
		AssignedCSID:    ticket.AssignedCSID,
		AssignedCSEmail: assignedCSEmail,
		ClaimedAt:       claimedAt,
		JITAttempts:     jitAttempts,
		Activities:      audits,
	}, nil
}

func (u *AdminUsecase) ListUsers(ctx context.Context) ([]domain.AdminManagedUser, error) {
	users, err := u.userRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]domain.AdminManagedUser, 0, len(users))
	for _, user := range users {
		items = append(items, domain.AdminManagedUser{ID: user.ID, Email: user.Email, Role: user.Role, CreatedAt: user.CreatedAt})
	}
	return items, nil
}

func (u *AdminUsecase) CreateManagedUser(ctx context.Context, email, plainPassword, role string) (*domain.AdminManagedUser, error) {
	if role != domain.RoleUser && role != domain.RoleCS {
		return nil, ErrAdminManagedRoleInvalid
	}
	existing, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email already exists")
	}
	hashed, err := password.Hash(plainPassword)
	if err != nil {
		return nil, err
	}
	user := &domain.User{Email: email, Password: hashed, Role: role, CreatedAt: time.Now()}
	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	kyc := datatypes.JSON([]byte(`{"kyc_status":"PENDING","is_blocked":false,"pin_hash":""}`))
	if role == domain.RoleCS {
		kyc = datatypes.JSON([]byte(`{"department":"support"}`))
	}
	profile := &domain.UserProfile{UserID: user.ID, Phone: "", Balance: 0, KYCData: kyc, CreatedAt: time.Now()}
	if err := u.profileRepo.Create(ctx, profile); err != nil {
		return nil, err
	}
	return &domain.AdminManagedUser{ID: user.ID, Email: user.Email, Role: user.Role, CreatedAt: user.CreatedAt}, nil
}

func (u *AdminUsecase) ListTerminalTickets(ctx context.Context) ([]domain.AdminSessionListItem, error) {
	tickets, err := u.ticketRepo.ListByStatuses(ctx, domain.TicketStatusClaimed, domain.TicketStatusInProgress)
	if err != nil {
		return nil, err
	}
	items := make([]domain.AdminSessionListItem, 0, len(tickets))
	for _, ticket := range tickets {
		var assignedCSEmail *string
		if ticket.AssignedCSID != nil {
			cs, err := u.userRepo.GetByID(ctx, *ticket.AssignedCSID)
			if err != nil {
				return nil, err
			}
			if cs != nil {
				assignedCSEmail = &cs.Email
			}
		}
		items = append(items, domain.AdminSessionListItem{
			TicketID:        ticket.ID,
			UserID:          ticket.UserID,
			TicketStatus:    ticket.Status,
			CreatedAt:       ticket.CreatedAt,
			AssignedCSID:    ticket.AssignedCSID,
			AssignedCSEmail: assignedCSEmail,
		})
	}
	return items, nil
}

func (u *AdminUsecase) ListTerminalLogs(ticketID uint64, limit int) []domain.TerminalLogEntry {
	if u.terminalPub == nil {
		return []domain.TerminalLogEntry{}
	}
	return u.terminalPub.ListTicketTerminal(ticketID, limit)
}

func datatypesJSONToMap(input datatypes.JSON) (map[string]any, bool) {
	if len(input) == 0 {
		return nil, false
	}
	var out map[string]any
	if err := json.Unmarshal(input, &out); err != nil {
		return nil, false
	}
	return out, true
}
