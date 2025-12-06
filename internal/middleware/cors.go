package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 基础跨域中间件：允许指定源、常用方法和头
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 允许的源（生产环境建议改为具体域名，如 "https://your-frontend.com"）
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin) // 动态允许当前请求源（支持多源）
			// 若需固定允许多个源，可判断 origin 是否在白名单中：
			// allowedOrigins := []string{"https://a.com", "https://b.com"}
			// for _, o := range allowedOrigins {
			// 	if o == origin {
			// 		c.Header("Access-Control-Allow-Origin", origin)
			// 		break
			// 	}
			// }
		}

		// 2. 允许的 HTTP 方法
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")

		// 3. 允许的请求头（需包含前端实际发送的头，如 Authorization、Content-Type）
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization,X-Requested-With")

		// 4. 是否允许携带 Cookie（若前端需传 Cookie，设为 true；此时 Allow-Origin 不能为 *）
		c.Header("Access-Control-Allow-Credentials", "true")

		// 5. 预检请求（OPTIONS）的缓存时间（86400 秒 = 1 天）
		c.Header("Access-Control-Max-Age", "86400")

		// 6. 处理预检请求（OPTIONS 请求）：直接返回 200 状态码，无需进入后续 Handler
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		// 继续执行后续中间件和 Handler
		c.Next()
	}
}
