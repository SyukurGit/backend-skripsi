package websocket

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"support-backend/config"
	"support-backend/internal/domain"
	"support-backend/internal/usecase"
	"support-backend/pkg/jwt"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ChatWSHandler:
// - Auth wajib (token di query atau header)
// - User/CS hanya boleh join ticket sesuai LP
func ChatWSHandler(cfg config.Config, messageUC *usecase.MessageUsecase, ticketRepo domain.TicketRepository, hub *ChatHub) gin.HandlerFunc {
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

		// VALIDASI: enforce least privilege di WebSocket.
		t, err := ticketRepo.GetByID(c.Request.Context(), ticketID)
		if err != nil || t == nil {
			c.Status(http.StatusNotFound)
			return
		}
		if claims.Role == domain.RoleUser && t.UserID != claims.UserID {
			c.Status(http.StatusForbidden)
			return
		}
		// VALIDASI: ticket harus aktif untuk chat.
		if t.Status != domain.TicketStatusOpen && t.Status != domain.TicketStatusClaimed && t.Status != domain.TicketStatusInProgress {
			c.Status(http.StatusForbidden)
			return
		}
		if claims.Role == domain.RoleCS {
			// VALIDASI: memastikan CS hanya bisa akses tiket miliknya dan status sesuai LP
			if t.AssignedCSID == nil || *t.AssignedCSID != claims.UserID {
				c.Status(http.StatusForbidden)
				return
			}
			if t.Status != domain.TicketStatusClaimed && t.Status != domain.TicketStatusInProgress {
				c.Status(http.StatusForbidden)
				return
			}
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
			// Client boleh mengirim pesan via WS.
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			text := strings.TrimSpace(string(msg))
			if text == "" {
				continue
			}
			_, _ = messageUC.SendMessage(c.Request.Context(), claims.UserID, claims.Role, ticketID, text)
		}
	}
}

func parseTokenFromWSRequest(cfg config.Config, c *gin.Context) (*jwt.Claims, bool) {
	// token bisa via query ?token=... atau header Authorization.
	tok := c.Query("token")
	if tok == "" {
		h := c.GetHeader("Authorization")
		if strings.HasPrefix(h, "Bearer ") {
			tok = strings.TrimPrefix(h, "Bearer ")
		}
	}
	if tok == "" {
		return nil, false
	}
	claims, err := jwt.ParseToken(cfg.JWTSecret, tok)
	if err != nil {
		return nil, false
	}
	return claims, true
}

// (tidak ada helper tambahan)
