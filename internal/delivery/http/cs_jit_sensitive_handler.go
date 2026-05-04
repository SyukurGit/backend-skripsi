package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"support-backend/internal/domain"
	"support-backend/internal/middleware"
	"support-backend/pkg/response"
)

type jitRequest struct {
	Feature string `json:"feature" binding:"required"`
}

func (h *HTTPHandler) CSRequestJIT(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
	ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
	if err != nil {
		code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
		c.JSON(code, body)
		return
	}

	var req jitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		code, body := response.Error(http.StatusBadRequest, "request tidak valid")
		c.JSON(code, body)
		return
	}

	// Validasi feature berdasarkan spesifikasi.
	if req.Feature != domain.JITFeatureResetPassword && req.Feature != domain.JITFeatureUnblockAccount && req.Feature != domain.JITFeatureChangeEmail && req.Feature != domain.JITFeatureResetPIN && req.Feature != "VIEW_KYC" {
		code, body := response.Error(http.StatusBadRequest, "feature tidak valid")
		c.JSON(code, body)
		return
	}

	s, err := h.jitUC.Request(c.Request.Context(), csID, ticketID, req.Feature)
	if err != nil {
		code, body := response.Error(http.StatusForbidden, "jit request ditolak")
		c.JSON(code, body)
		return
	}
	code, body := response.Created("jit aktif", map[string]any{"expired_at": s.ExpiredAt, "feature": s.Feature})
	c.JSON(code, body)
}

type resetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

func (h *HTTPHandler) CSSensitiveResetPassword(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
	ticketID, _ := strconv.ParseUint(c.Param("ticket_id"), 10, 64)

	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		code, body := response.Error(http.StatusBadRequest, "request tidak valid")
		c.JSON(code, body)
		return
	}

	if err := h.csUC.ResetPasswordByTicket(c.Request.Context(), csID, ticketID, req.NewPassword); err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal reset password")
		c.JSON(code, body)
		return
	}
	// Audit sudah dicatat di usecase agar tidak dobel.
	code, body := response.OK("password direset", nil)
	c.JSON(code, body)
}

type changeEmailRequest struct {
	NewEmail string `json:"new_email" binding:"required,email"`
}

func (h *HTTPHandler) CSSensitiveChangeEmail(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
	ticketID, _ := strconv.ParseUint(c.Param("ticket_id"), 10, 64)

	var req changeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		code, body := response.Error(http.StatusBadRequest, "request tidak valid")
		c.JSON(code, body)
		return
	}

	if err := h.csUC.ChangeEmailByTicket(c.Request.Context(), csID, ticketID, req.NewEmail); err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal ganti email")
		c.JSON(code, body)
		return
	}
	// Audit sudah dicatat di usecase agar tidak dobel.
	code, body := response.OK("email diganti", nil)
	c.JSON(code, body)
}

type unblockAccountRequest struct {
	Reason string `json:"reason"`
}

func (h *HTTPHandler) CSSensitiveUnblockAccount(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
	ticketID, _ := strconv.ParseUint(c.Param("ticket_id"), 10, 64)

	_ = c.ShouldBindJSON(&unblockAccountRequest{})

	if err := h.csUC.UnblockAccountByTicket(c.Request.Context(), csID, ticketID); err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal unblock")
		c.JSON(code, body)
		return
	}
	// Audit sudah dicatat di usecase agar tidak dobel.
	code, body := response.OK("akun dibuka", nil)
	c.JSON(code, body)
}

type resetPINRequest struct {
	NewPIN string `json:"new_pin" binding:"required,min=4"`
}

func (h *HTTPHandler) CSSensitiveResetPIN(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
	ticketID, _ := strconv.ParseUint(c.Param("ticket_id"), 10, 64)

	var req resetPINRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		code, body := response.Error(http.StatusBadRequest, "request tidak valid")
		c.JSON(code, body)
		return
	}

	if err := h.csUC.ResetPINByTicket(c.Request.Context(), csID, ticketID, req.NewPIN); err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal reset pin")
		c.JSON(code, body)
		return
	}
	// Audit sudah dicatat di usecase agar tidak dobel.
	code, body := response.OK("pin direset", nil)
	c.JSON(code, body)
}
