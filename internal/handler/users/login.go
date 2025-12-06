package users

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (u *Users) Login() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 判断请求内容是否为空
		if c.Request.ContentLength == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求内容不能为空"})
			return
		}

		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 校验用户名密码（直接从配置文件获取进行简单校验）
		if req.Username != u.svcCtx.Conf.Login.AdminUser || req.Password != u.svcCtx.Conf.Login.AdminPassword {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
			return
		}
		// 生成 JWT 令牌
		uID := uuid.New()
		token, err := u.jwtService.GenerateAccessToken(uID.String(), req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
			return
		}
		data := map[string]string{
			"token": token,
		}
		u.svcCtx.Resp.RESP_DATA(c, data)
	})

}
