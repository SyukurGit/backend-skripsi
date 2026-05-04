package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"support-backend/pkg/response"
)

func RBAC(allowedRoles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, r := range allowedRoles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		role := MustGetRole(c)
		if !allowed[role] {
			code, body := response.Error(http.StatusForbidden, "role tidak diizinkan")
			c.JSON(code, body)
			c.Abort()
			return
		}
		c.Next()
	}
}
