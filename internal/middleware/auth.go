package middleware

import (
	"health-probe/internal/svc"
	"strings"

	"github.com/gin-gonic/gin"
)

// 自定义中间件示例：权限校验中间件
// Auth JWT 认证中间件（解析 Bearer Token）
func Auth(ctx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 Authorization 头获取 Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			ctx.Resp.RESP_UNAUTHORIZED(c, "authorization failed")
			c.Abort()
			return
		}

		// 2. 解析 Bearer 前缀（必须是 "Bearer " + Token，注意空格）
		const bearerPrefix = "Bearer "
		if len(authHeader) < len(bearerPrefix) || !strings.HasPrefix(authHeader, bearerPrefix) {
			ctx.Resp.RESP_UNAUTHORIZED(c, "The token format is incorrect (need: Bearer <token>).")
			c.Abort()
			return
		}

		// 3. 提取 Token 字符串（去掉前缀）
		tokenStr := authHeader[len(bearerPrefix):]

		// 4. 验证 Token
		claims, err := ctx.JWT.ValidateAccessToken(tokenStr)
		if err != nil {
			ctx.Resp.RESP_UNAUTHORIZED(c, err.Error())
			c.Abort()
			return
		}

		// 5. 存储用户信息到上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
