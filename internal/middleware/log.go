package middleware

import (
	"health-probe/internal/svc"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 自定义中间件示例：日志中间件
func Logger(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		// 执行后续处理逻辑
		cost := time.Since(start)
		svcCtx.Logger.Info("", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path), zap.Duration("cost", cost), zap.Int("status", c.Writer.Status()))
	}
}
