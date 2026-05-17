package websocket

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"support-backend/config"
	"support-backend/internal/domain"
	"support-backend/pkg/jwt"
)

func TerminalWSHandler(cfg config.Config, hub *TerminalHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketID, err := strconv.ParseUint(c.Param("ticket_id"), 10, 64)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		claims, ok := parseTokenFromWSRequest(cfg, c)
		if !ok {
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
		hub.Join(ticketID, wsc)
		defer func() {
			hub.Leave(ticketID, wsc)
			_ = wsc.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}
}

var _ = jwt.Claims{}
