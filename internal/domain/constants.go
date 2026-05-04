package domain

const (
	RoleAdmin = "admin"
	RoleCS    = "cs"
	RoleUser  = "user"
)

const (
	TicketStatusOpen       = "OPEN"
	TicketStatusClaimed    = "CLAIMED"
	TicketStatusInProgress = "IN_PROGRESS"
	TicketStatusResolved   = "RESOLVED"
	TicketStatusClosed     = "CLOSED"
)

const (
	JITFeatureResetPassword  = "RESET_PASSWORD"
	JITFeatureUnblockAccount = "UNBLOCK_ACCOUNT"
	JITFeatureChangeEmail    = "CHANGE_EMAIL"
	JITFeatureResetPIN       = "RESET_PIN"
)

const (
	AuditLevelHigh   = "HIGH"
	AuditLevelMedium = "MEDIUM"
	AuditLevelLow    = "LOW"
)
