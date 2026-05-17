package http

import (
	"net/http"
	"slices"
	"strconv"

	"github.com/gin-gonic/gin"

	"support-backend/internal/domain"
	"support-backend/internal/middleware"
	"support-backend/pkg/response"
)

func (h *HTTPHandler) UserCreateTicket(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	t, err := h.ticketUC.CreateTicket(c.Request.Context(), userID)
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal buat ticket")
		c.JSON(code, body)
		return
	}
	code, body := response.Created("ticket dibuat", t)
	c.JSON(code, body)
}

func (h *HTTPHandler) UserListTickets(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	items, err := h.ticketUC.ListMyTickets(c.Request.Context(), userID)
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal list ticket")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}

func (h *HTTPHandler) UserCloseTicket(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	role := middleware.MustGetRole(c)
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}

	if err := h.ticketUC.SetStatus(c.Request.Context(), userID, role, ticketID, domain.TicketStatusClosed); err != nil {
		code, body := response.Error(http.StatusForbidden, "tidak bisa close ticket")
		c.JSON(code, body)
		return
	}

	code, body := response.OK("ticket ditutup", nil)
	c.JSON(code, body)
}

type sendMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

func (h *HTTPHandler) UserSendMessage(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	role := middleware.MustGetRole(c)
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		code, body := response.Error(http.StatusBadRequest, "request tidak valid")
		c.JSON(code, body)
		return
	}

	m, err := h.messageUC.SendMessage(c.Request.Context(), userID, role, ticketID, req.Message)
	if err != nil {
		code, body := response.Error(http.StatusForbidden, "tidak bisa kirim pesan")
		c.JSON(code, body)
		return
	}

	code, body := response.Created("pesan terkirim", m)
	c.JSON(code, body)
}

func (h *HTTPHandler) UserListMessages(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	role := middleware.MustGetRole(c)
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))

	items, err := h.messageUC.ListMessages(c.Request.Context(), userID, role, ticketID, limit)
	if err != nil {
		code, body := response.Error(http.StatusForbidden, "tidak bisa akses pesan")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}

func (h *HTTPHandler) UserTicketActivity(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}

	t, err := h.ticketUC.GetByID(c.Request.Context(), ticketID)
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal ambil ticket")
		c.JSON(code, body)
		return
	}
	if t == nil || t.UserID != userID {
		code, body := response.Error(http.StatusForbidden, "tidak bisa akses aktivitas ticket")
		c.JSON(code, body)
		return
	}

	items, err := h.auditUC.List(c.Request.Context(), "", 200)
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal ambil aktivitas ticket")
		c.JSON(code, body)
		return
	}

	allowed := []string{
		"TICKET_CLAIM",
		"TICKET_STATUS_UPDATE",
		"MESSAGE_SEND",
		"JIT_REQUEST",
		"VIEW_KYC",
		"RESET_PASSWORD",
		"CHANGE_EMAIL",
		"UNBLOCK_ACCOUNT",
		"RESET_PIN",
		"JIT_REVOKE_TICKET_CLOSED",
	}

	filtered := make([]domain.AuditLog, 0)
	for _, item := range items {
		if item.TicketID == nil || *item.TicketID != ticketID {
			continue
		}
		if !slices.Contains(allowed, item.Action) {
			continue
		}
		filtered = append(filtered, item)
	}

	code, body := response.OK("ok", filtered)
	c.JSON(code, body)
}
