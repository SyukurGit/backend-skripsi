package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"support-backend/internal/domain"
	"support-backend/internal/usecase"
	"support-backend/pkg/response"
)

// JITRequired memastikan CS punya session JIT aktif untuk feature tertentu.
func JITRequired(jitUC *usecase.JITUsecase, feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := MustGetRole(c)
		if role != domain.RoleCS {
			code, body := response.Error(http.StatusForbidden, "hanya CS")
			c.JSON(code, body)
			c.Abort()
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
		csID := MustGetUserID(c)

		// CEK JIT: memastikan akses hanya berlaku dalam waktu terbatas
		if err := jitUC.EnsureValid(c.Request.Context(), csID, ticketID, feature); err != nil {
			code, body := response.Error(http.StatusForbidden, "jit tidak aktif atau expired")
			c.JSON(code, body)
			c.Abort()
			return
		}

		c.Next()
	}
}
