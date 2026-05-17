package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"support-backend/internal/middleware"
	"support-backend/pkg/response"
)

func (h *HTTPHandler) CSViewTicketUserProfile(c *gin.Context) {
	csID := middleware.MustGetUserID(c)
	ticketID, _ := strconv.ParseUint(c.Param("ticket_id"), 10, 64)

	p, err := h.csUC.ViewUserProfileByTicket(c.Request.Context(), csID, ticketID)
	if err != nil || p == nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal ambil profile")
		c.JSON(code, body)
		return
	}

	code, body := response.OK("ok", p)
	c.JSON(code, body)
}
