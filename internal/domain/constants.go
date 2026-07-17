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
	JITFeatureViewKYC        = "VIEW_KYC"
)

func IsValidJITFeature(feature string) bool {
	switch feature {
	case JITFeatureResetPassword,
		JITFeatureUnblockAccount,
		JITFeatureChangeEmail,
		JITFeatureResetPIN,
		JITFeatureViewKYC:
		return true
	default:
		return false
	}
}

const (
	AuditLevelHigh   = "HIGH"
	AuditLevelMedium = "MEDIUM"
	AuditLevelLow    = "LOW"
)
