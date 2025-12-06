package router

import (
	"context"
	"health-probe/internal/svc"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Engine 路由引擎结构体
type Engine struct {
	ginEngine *gin.Engine  // 底层 Gin 引擎
	server    *http.Server // HTTP 服务器（用于优雅关闭）
	addr      string       // 监听地址（如 :8080）
	svcCtx    *svc.ServiceContext
}

// NewEngine 创建路由引擎实例
// addr: 监听地址（如 ":8080"）
// mode: Gin 运行模式（gin.DebugMode/gin.ReleaseMode/gin.TestMode）
func NewEngine(addr, mode string, svcCtx *svc.ServiceContext) *Engine {
	gin.SetMode(mode)
	engin := gin.New()
	engin.Use(gin.Recovery())
	return &Engine{
		ginEngine: engin, // 默认包含 Logger 和 Recovery 中间件
		addr:      addr,
		svcCtx:    svcCtx,
	}
}

// Use 注册全局中间件（对所有路由生效）
func (e *Engine) Use(middlewares ...gin.HandlerFunc) {
	e.ginEngine.Use(middlewares...)
}

// Group 创建路由组（支持路由前缀和组内中间件）
func (e *Engine) Group(relativePath string, middlewares ...gin.HandlerFunc) *gin.RouterGroup {
	return e.ginEngine.Group(relativePath, middlewares...)
}

// RegisterRoutes 注册业务路由（由外部实现，解耦核心逻辑）
func (e *Engine) RegisterRoutes(registerFunc func(engine *Engine)) {
	registerFunc(e)
}

// Run 启动服务器（阻塞）并支持优雅关闭
func (e *Engine) Run() {
	// 初始化 HTTP 服务器
	e.server = &http.Server{
		Addr:    e.addr,
		Handler: e.ginEngine,
	}

	// 启动服务器（非阻塞）
	go func() {
		log.Printf("The server has started successfully. Listen for the address: %s (Mode: %s)\n", e.addr, gin.Mode())
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("The server startup failed: %v", err)
		}
	}()

	// 监听关闭信号（SIGINT: Ctrl+C，SIGTERM: 容器停止信号）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // 阻塞等待信号
	log.Println("The server is shutting down gracefully...")

	// 创建 5 秒超时上下文（确保请求有足够时间处理）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭服务器（不再接收新请求，等待现有请求完成）
	if err := e.server.Shutdown(ctx); err != nil {
		log.Printf("The server shutdown failed: %v", err)
	} else {
		log.Println("The server has been shut down gracefully.")
	}
}
