package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"support-backend/config"
	"support-backend/pkg/jwt"
	"support-backend/pkg/response"
)

func JWTAuth(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			code, body := response.Error(http.StatusUnauthorized, "token tidak ada")
			c.JSON(code, body)
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(h, "Bearer ")
		claims, err := jwt.ParseToken(cfg.JWTSecret, tokenString)
		if err != nil {
			code, body := response.Error(http.StatusUnauthorized, "token tidak valid")
			c.JSON(code, body)
			c.Abort()
			return
		}

		c.Set(string(ContextUserID), claims.UserID)
		c.Set(string(ContextRole), claims.Role)
		c.Set(string(ContextEmail), claims.Email)
		c.Next()
	}
}

func MustGetUserID(c *gin.Context) uint64 {
	v, _ := c.Get(string(ContextUserID))
	id, _ := v.(uint64)
	return id
}

func MustGetRole(c *gin.Context) string {
	v, _ := c.Get(string(ContextRole))
	role, _ := v.(string)
	return role
}
