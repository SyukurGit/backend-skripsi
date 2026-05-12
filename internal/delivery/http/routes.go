package http

import (
	"github.com/gin-gonic/gin"

	"support-backend/config"
	"support-backend/internal/delivery/websocket"
	"support-backend/internal/domain"
	"support-backend/internal/middleware"
	"support-backend/internal/usecase"
)

func RegisterRoutes(
	r *gin.Engine,
	cfg config.Config,
	authUC *usecase.AuthUsecase,
	ticketUC *usecase.TicketUsecase,
	messageUC *usecase.MessageUsecase,
	jitUC *usecase.JITUsecase,
	auditUC *usecase.AuditUsecase,
	csUC *usecase.CSUsecase,
	ticketRepo domain.TicketRepository,
	chatHub *websocket.ChatHub,
	auditHub *websocket.AuditHub,
) {
	h := NewHTTPHandler(cfg, authUC, ticketUC, messageUC, jitUC, auditUC, csUC)

	r.GET("/health", h.Health)
	r.POST("/auth/login", h.Login)

	// WebSocket (auth dilakukan di handler, tidak boleh bypass).
	r.GET("/ws/chat/:ticket_id", websocket.ChatWSHandler(cfg, messageUC, ticketRepo, chatHub))
	r.GET("/ws/audit", websocket.AuditWSHandler(cfg, auditHub))

	auth := r.Group("/")
	auth.Use(middleware.JWTAuth(cfg))
	auth.POST("/auth/logout", h.Logout)

	user := auth.Group("/user")
	user.Use(middleware.RBAC("user"))
	user.POST("/tickets", h.UserCreateTicket)
	user.GET("/tickets", h.UserListTickets)
	user.POST("/tickets/:ticket_id/close", h.UserCloseTicket)
	user.GET("/tickets/:ticket_id/messages", h.UserListMessages)
	user.POST("/tickets/:ticket_id/messages", h.UserSendMessage)

	cs := auth.Group("/cs")
	cs.Use(middleware.RBAC("cs"))
	cs.GET("/tickets/open", h.CSListOpenTickets)
	cs.GET("/tickets/my", h.CSListMyTickets)
	cs.POST("/tickets/:ticket_id/claim", h.CSClaimTicket)

	// CS endpoints yang butuh LP.
	csTicket := cs.Group("/tickets/:ticket_id")
	csTicket.Use(middleware.LeastPrivilegeTicketCS(ticketRepo))
	csTicket.GET("", h.CSGetTicket)
	csTicket.POST("/status", h.CSUpdateTicketStatus)
	csTicket.GET("/messages", h.CSListMessages)
	csTicket.POST("/messages", h.CSSendMessage)
	csTicket.GET("/user/profile", h.CSViewTicketUserProfile)
	csTicket.POST("/jit/request", h.CSRequestJIT)

	// Sensitive features: butuh JIT + LP.
	csSensitive := csTicket.Group("/sensitive")
	csSensitive.POST("/reset-password", middleware.JITRequired(jitUC, "RESET_PASSWORD"), h.CSSensitiveResetPassword)
	csSensitive.POST("/unblock-account", middleware.JITRequired(jitUC, "UNBLOCK_ACCOUNT"), h.CSSensitiveUnblockAccount)
	csSensitive.POST("/change-email", middleware.JITRequired(jitUC, "CHANGE_EMAIL"), h.CSSensitiveChangeEmail)
	csSensitive.POST("/reset-pin", middleware.JITRequired(jitUC, "RESET_PIN"), h.CSSensitiveResetPIN)

	admin := auth.Group("/admin")
	admin.Use(middleware.RBAC("admin"))
	admin.GET("/audit-logs", h.AdminListAuditLogs)
}
