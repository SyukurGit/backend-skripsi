package http

import (
	"github.com/gin-gonic/gin"

	"support-backend/config"
	"support-backend/internal/usecase"
	"support-backend/pkg/response"
)

type HTTPHandler struct {
	cfg       config.Config
	authUC    *usecase.AuthUsecase
	ticketUC  *usecase.TicketUsecase
	messageUC *usecase.MessageUsecase
	jitUC     *usecase.JITUsecase
	auditUC   *usecase.AuditUsecase
	csUC      *usecase.CSUsecase
	adminUC   *usecase.AdminUsecase
}

func NewHTTPHandler(cfg config.Config, authUC *usecase.AuthUsecase, ticketUC *usecase.TicketUsecase, messageUC *usecase.MessageUsecase, jitUC *usecase.JITUsecase, auditUC *usecase.AuditUsecase, csUC *usecase.CSUsecase, adminUC *usecase.AdminUsecase) *HTTPHandler {
	return &HTTPHandler{cfg: cfg, authUC: authUC, ticketUC: ticketUC, messageUC: messageUC, jitUC: jitUC, auditUC: auditUC, csUC: csUC, adminUC: adminUC}
}

func (h *HTTPHandler) Health(c *gin.Context) {
	code, body := response.OK("ok", map[string]any{"service": "support-backend"})
	c.JSON(code, body)
}
