package http

import (
	"github.com/gin-gonic/gin"

	"support-backend/internal/domain"
	"support-backend/internal/middleware"
	"support-backend/pkg/response"
)

// Logout: untuk memenuhi requirement revoke JIT saat logout.
func (h *HTTPHandler) Logout(c *gin.Context) {
	userID := middleware.MustGetUserID(c)
	role := middleware.MustGetRole(c)

	if role == domain.RoleCS {
		_ = h.jitUC.RevokeOnLogout(c.Request.Context(), userID)
	}

	code, body := response.OK("logout berhasil", nil)
	c.JSON(code, body)
}
