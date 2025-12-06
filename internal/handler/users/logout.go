package users

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (u *Users) Logout() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "logout success",
		})
	})
}
