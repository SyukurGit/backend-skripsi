package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"support-backend/internal/domain"
	"support-backend/pkg/response"
)

// LeastPrivilegeTicketCS melakukan validasi bahwa CS hanya bisa akses tiket miliknya.
// Catatan: pembatasan status (mis. chat hanya untuk ticket aktif) divalidasi di usecase/handler terkait.
func LeastPrivilegeTicketCS(ticketRepo domain.TicketRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := MustGetRole(c)
		if role != domain.RoleCS {
			c.Next()
			return
		}

		ticketIDStr := c.Param("ticket_id")
		ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
		if err != nil {
			code, body := response.Error(http.StatusBadRequest, "ticket_id tidak valid")
			c.JSON(code, body)
			c.Abort()
			return
		}

		t, err := ticketRepo.GetByID(c.Request.Context(), ticketID)
		if err != nil {
			code, body := response.Error(http.StatusInternalServerError, "gagal ambil ticket")
			c.JSON(code, body)
			c.Abort()
			return
		}
		if t == nil {
			code, body := response.Error(http.StatusNotFound, "ticket tidak ditemukan")
			c.JSON(code, body)
			c.Abort()
			return
		}

		csID := MustGetUserID(c)
		// VALIDASI: memastikan CS hanya bisa akses tiket miliknya
		if t.AssignedCSID == nil || *t.AssignedCSID != csID {
			code, body := response.Error(http.StatusForbidden, "akses ditolak")
			c.JSON(code, body)
			c.Abort()
			return
		}

		c.Next()
	}
}
