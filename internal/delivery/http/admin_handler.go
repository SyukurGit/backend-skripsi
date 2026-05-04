package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"support-backend/pkg/response"
)

func (h *HTTPHandler) AdminListAuditLogs(c *gin.Context) {
	level := c.Query("level")
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.auditUC.List(c.Request.Context(), level, limit)
	if err != nil {
		code, body := response.Error(http.StatusInternalServerError, "gagal list audit")
		c.JSON(code, body)
		return
	}
	code, body := response.OK("ok", items)
	c.JSON(code, body)
}
