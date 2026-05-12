package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"support-backend/internal/domain"
	"support-backend/internal/middleware"
	"support-backend/internal/usecase"
	"support-backend/pkg/response"
)

func (h *HTTPHandler) CSListOpenTickets(c *gin.Context) {
	items, err := h.ticketUC.ListOpenUnassigned(c.Request.Context())
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal list ticket")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}

func (h *HTTPHandler) CSListMyTickets(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
	items, err := h.ticketUC.ListMyActiveTicketsCS(c.Request.Context(), csID)
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal list ticket")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}

func (h *HTTPHandler) CSGetTicket(c *gin.Context) {
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}
	// LP sudah divalidasi oleh middleware.
	t, err := h.ticketUC.GetByID(c.Request.Context(), ticketID)
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal ambil ticket")
		c.JSON(code, body)
		return
	}
	if t == nil {
		code, body := response.Error(http.StatusNotFound, "ticket tidak ditemukan")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", t)
	c.JSON(code, body)
}

func (h *HTTPHandler) CSClaimTicket(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}

	if err := h.ticketUC.ClaimTicket(c.Request.Context(), csID, ticketID); err != nil {
		if err == usecase.ErrCSActiveTicketLimit {
			code, body := response.Error(http.StatusForbidden, "maksimal 2 ticket aktif")
			c.JSON(code, body)
			return
		}
		code, body := response.Error(http.StatusConflict, "ticket tidak bisa di-claim")
		c.JSON(code, body)
		return
	}

	code, body := response.OK("ticket di-claim", nil)
	c.JSON(code, body)
}

type updateTicketStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *HTTPHandler) CSUpdateTicketStatus(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
	role := middleware.MustGetRole(c)
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}

	var req updateTicketStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		code, body := response.Error(http.StatusBadRequest, "request tidak valid")
		c.JSON(code, body)
		return
	}

	// Batasi status yang boleh diset oleh CS (state machine).
	if req.Status != domain.TicketStatusInProgress && req.Status != domain.TicketStatusResolved && req.Status != domain.TicketStatusClosed {
		code, body := response.Error(http.StatusBadRequest, "status tidak valid")
		c.JSON(code, body)
		return
	}

	if err := h.ticketUC.SetStatus(c.Request.Context(), csID, role, ticketID, req.Status); err != nil {
		if err == usecase.ErrTicketInvalidStatus {
			code, body := response.Error(http.StatusBadRequest, "transisi status tidak valid")
			c.JSON(code, body)
			return
		}
		code, body := response.Error(http.StatusForbidden, "tidak bisa update status")
		c.JSON(code, body)
		return
	}

	code, body := response.OK("status diupdate", nil)
	c.JSON(code, body)
}

func (h *HTTPHandler) CSSendMessage(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
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

	m, err := h.messageUC.SendMessage(c.Request.Context(), csID, role, ticketID, req.Message)
	if err != nil {
		code, body := response.Error(http.StatusForbidden, "tidak bisa kirim pesan")
		c.JSON(code, body)
		return
	}
	code, body := response.Created("pesan terkirim", m)
	c.JSON(code, body)
}

func (h *HTTPHandler) CSListMessages(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
	role := middleware.MustGetRole(c)
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))

	items, err := h.messageUC.ListMessages(c.Request.Context(), csID, role, ticketID, limit)
	if err != nil {
		code, body := response.Error(http.StatusForbidden, "tidak bisa akses pesan")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}
