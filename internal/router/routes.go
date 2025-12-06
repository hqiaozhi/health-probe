package router

import (
	"health-probe/internal/handler/health"
	"health-probe/internal/handler/notify"
	"health-probe/internal/handler/task"
	"health-probe/internal/handler/users"
	"health-probe/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RegisterBusinessRoutes(engine *Engine) {
	engine.Use(middleware.Logger(engine.svcCtx), middleware.Cors())
	ctx := engine.svcCtx

	// metrics
	engine.ginEngine.GET("/metrics", func(gin *gin.Context) {
		promhttp.Handler().ServeHTTP(gin.Writer, gin.Request)
	})

	// 版本：/api/v1
	v1 := engine.Group("/api/v1")
	{
		// 公共路由组
		publicGroup := v1.Group("")
		{
			publicGroup.GET("/health", health.New().Health)
			publicGroup.POST("/users/login", users.New(ctx).Login())
		}

		// 需权限校验
		authGroup := v1.Group("")
		authGroup.Use(middleware.Auth(ctx))
		{
			// 用户
			usersGroup := authGroup.Group("/users")
			{
				usersGroup.POST("/logout", users.New(ctx).Logout())
			}
			// 通知
			notifyGroup := authGroup.Group("/notify")
			{
				notifyGroup.GET("", notify.New(ctx).Query)
				notifyGroup.DELETE("", notify.New(ctx).DeleteByIDList)
				notifyGroup.DELETE("/:id", notify.New(ctx).DeleteByID)
				notifyGroup.PUT("/:id", notify.New(ctx).UpdateReadStatus)
			}
			// 任务
			taskGroup := authGroup.Group("/tasks")
			{
				taskGroup.GET("", task.New(ctx).Query)
				taskGroup.POST("", task.New(ctx).Create)
				taskGroup.DELETE("", task.New(ctx).DeleteByIDList)
				taskGroup.DELETE("/:id", task.New(ctx).DeleteByID)
				taskGroup.PATCH("/:id", task.New(ctx).Update)
				taskGroup.POST("/control", task.New(ctx).Control)

			}
		}
	}
}
