package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"support-backend/internal/usecase"
	"support-backend/pkg/response"
)

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *HTTPHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		code, body := response.Error(http.StatusBadRequest, "request tidak valid")
		c.JSON(code, body)
		return
	}

	token, user, err := h.authUC.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err == usecase.ErrInvalidCredentials {
			code, body := response.Error(http.StatusUnauthorized, "email/password salah")
			c.JSON(code, body)
			return
		}
		code, body := response.Error(http.StatusInternalServerError, "gagal login")
		c.JSON(code, body)
		return
	}

	code, body := response.OK("login berhasil", map[string]any{
		"token": token,
		"user":  map[string]any{"id": user.ID, "email": user.Email, "role": user.Role},
	})
	c.JSON(code, body)
}
