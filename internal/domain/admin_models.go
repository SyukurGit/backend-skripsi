package domain

import "time"

type AdminDashboardStats struct {
	TotalUsers         int64 `json:"total_users"`
	TotalCS            int64 `json:"total_cs"`
	TotalAdmins        int64 `json:"total_admins"`
	TicketsInProcess   int64 `json:"tickets_in_process"`
	TicketsUnassigned  int64 `json:"tickets_unassigned"`
	TicketsResolved    int64 `json:"tickets_resolved"`
	TicketsClosed      int64 `json:"tickets_closed"`
	SensitiveActions   int64 `json:"sensitive_actions"`
	PendingJITRequests int64 `json:"pending_jit_requests"`
}

type AdminSessionListItem struct {
	TicketID         uint64     `json:"ticket_id"`
	UserID           uint64     `json:"user_id"`
	TicketStatus     string     `json:"ticket_status"`
	CreatedAt        time.Time  `json:"created_at"`
	AssignedCSID     *uint64    `json:"assigned_cs_id,omitempty"`
	AssignedCSEmail  *string    `json:"assigned_cs_email,omitempty"`
	ClaimedAt        *time.Time `json:"claimed_at,omitempty"`
	LastActivityAt   *time.Time `json:"last_activity_at,omitempty"`
	SensitiveActions int        `json:"sensitive_actions"`
	JITAttempts      int        `json:"jit_attempts"`
}

type AdminJITAttempt struct {
	RequestedAt time.Time `json:"requested_at"`
	Feature     string    `json:"feature"`
	Granted     bool      `json:"granted"`
	Reason      string    `json:"reason"`
}

type AdminSessionDetail struct {
	TicketID        uint64            `json:"ticket_id"`
	UserID          uint64            `json:"user_id"`
	UserEmail       *string           `json:"user_email,omitempty"`
	TicketStatus    string            `json:"ticket_status"`
	CreatedAt       time.Time         `json:"created_at"`
	AssignedCSID    *uint64           `json:"assigned_cs_id,omitempty"`
	AssignedCSEmail *string           `json:"assigned_cs_email,omitempty"`
	ClaimedAt       *time.Time        `json:"claimed_at,omitempty"`
	JITAttempts     []AdminJITAttempt `json:"jit_attempts"`
	Activities      []AuditLog        `json:"activities"`
}

type AdminManagedUser struct {
	ID        uint64    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type TerminalLogEntry struct {
	TicketID  uint64    `json:"ticket_id"`
	Sequence  uint64    `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}
