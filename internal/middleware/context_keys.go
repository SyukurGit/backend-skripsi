package middleware

type contextKey string

const (
	ContextUserID contextKey = "user_id"
	ContextRole   contextKey = "role"
	ContextEmail  contextKey = "email"
)
