package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func New() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"data":    "",
		"message": "ok",
	})
}
