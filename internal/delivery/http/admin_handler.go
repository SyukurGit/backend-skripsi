package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"support-backend/internal/usecase"
	"support-backend/pkg/response"
)

type adminCreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required"`
}

func (h *HTTPHandler) AdminDashboardStats(c *gin.Context) {
	stats, err := h.adminUC.DashboardStats(c.Request.Context())
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal ambil statistik admin")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", stats)
	c.JSON(code, body)
}

func (h *HTTPHandler) AdminListAuditLogs(c *gin.Context) {
	level := c.Query("level")
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.auditUC.List(c.Request.Context(), level, limit)
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal list audit")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}

func (h *HTTPHandler) AdminListSessions(c *gin.Context) {
	items, err := h.adminUC.ListSessions(c.Request.Context())
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal ambil sesi bantuan")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}

func (h *HTTPHandler) AdminSessionDetail(c *gin.Context) {
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}
	item, err := h.adminUC.SessionDetail(c.Request.Context(), ticketID)
	if err != nil {
		if errors.Is(err, usecase.ErrTicketNotFound) {
			code, body := response.Error(http.StatusNotFound, "sesi ticket tidak ditemukan")
			c.JSON(code, body)
			return
		}
		code, body := response.Error(http.StatusInternalServerError, "gagal ambil detail sesi")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", item)
	c.JSON(code, body)
}

func (h *HTTPHandler) AdminListUsers(c *gin.Context) {
	items, err := h.adminUC.ListUsers(c.Request.Context())
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal ambil daftar pengguna")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}

func (h *HTTPHandler) AdminListTerminalTickets(c *gin.Context) {
	items, err := h.adminUC.ListTerminalTickets(c.Request.Context())
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal ambil daftar terminal log")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}

func (h *HTTPHandler) AdminListTerminalLogs(c *gin.Context) {
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	items := h.adminUC.ListTerminalLogs(ticketID, limit)
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}

func (h *HTTPHandler) AdminCreateUser(c *gin.Context) {
	var req adminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		code, body := response.Error(http.StatusBadRequest, "request tidak valid")
		c.JSON(code, body)
		return
	}
	item, err := h.adminUC.CreateManagedUser(c.Request.Context(), req.Email, req.Password, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrAdminManagedRoleInvalid):
			code, body := response.Error(http.StatusBadRequest, "role hanya boleh user atau cs")
			c.JSON(code, body)
			return
		case err.Error() == "email already exists":
			code, body := response.Error(http.StatusConflict, "email sudah digunakan")
			c.JSON(code, body)
			return
		default:
			code, body := response.Error(http.StatusInternalServerError, "gagal menambah pengguna")
			c.JSON(code, body)
			return
		}
	}
	code, body := response.Created("pengguna berhasil dibuat", item)
	c.JSON(code, body)
}
