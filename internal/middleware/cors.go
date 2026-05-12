package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"support-backend/config"
)

// CORS enables browser access from the frontend dev server.
// Configure via CORS_ALLOWED_ORIGINS (comma-separated) or "*".
func CORS(cfg config.Config) gin.HandlerFunc {
	allowed := parseAllowedOrigins(cfg.CORSAllowedOrigins)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if allowed["*"] || allowed[origin] {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}

		c.Next()
	}
}

func parseAllowedOrigins(v string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(v, ",") {
		o := strings.TrimSpace(part)
		if o == "" {
			continue
		}
		out[o] = true
	}
	return out
}
