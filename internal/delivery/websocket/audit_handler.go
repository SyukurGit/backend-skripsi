package websocket

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"support-backend/config"
	"support-backend/internal/domain"
	"support-backend/pkg/jwt"
)

// AuditWSHandler: hanya admin yang boleh connect.
func AuditWSHandler(cfg config.Config, hub *AuditHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := c.Query("token")
		if tok == "" {
			h := c.GetHeader("Authorization")
			if strings.HasPrefix(h, "Bearer ") {
				tok = strings.TrimPrefix(h, "Bearer ")
			}
		}
		if tok == "" {
			c.Status(http.StatusUnauthorized)
			return
		}

		claims, err := jwt.ParseToken(cfg.JWTSecret, tok)
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}
		if claims.Role != domain.RoleAdmin {
			c.Status(http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		wsc := NewWSConn(conn)
		hub.Register(wsc)
		defer func() {
			hub.Unregister(wsc)
			_ = wsc.Close()
		}()

		for {
			// Admin client tidak perlu kirim pesan; hanya drain.
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}
}
