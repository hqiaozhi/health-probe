package cmd

import (
	"fmt"
	"health-probe/internal/router"
	"health-probe/internal/svc"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	vers bool
	mode string
)

// startCmd represents the start command
var RootCmd = &cobra.Command{
	Short: "app [flag] [args]",
	Run:   startRun,
}

func startRun(cmd *cobra.Command, args []string) {
	svcCtx := svc.NewSvcCtx(cmd.Context())
	if vers {
		fmt.Println(svcCtx.Conf.App.Version)
		os.Exit(0)
	}
	if mode == "" {
		mode = svcCtx.Conf.Gin.Mode
	}

	appcfg := svcCtx.Conf.App
	addr := appcfg.Host + ":" + strconv.Itoa(appcfg.Port)
	// 1. 创建路由引擎实例（监听 :8080，生产环境使用 ReleaseMode）

	engine := router.NewEngine(addr, mode, svcCtx)

	// 2. 注册业务路由（核心：解耦路由定义与引擎实现）
	engine.RegisterRoutes(router.RegisterBusinessRoutes)

	// 3. 启动服务器（阻塞，支持优雅关闭）
	engine.Run()
}

func init() {
	// 初始化根命令，这一步会自动添加 completion 命令
	RootCmd.CompletionOptions.DisableDefaultCmd = true

	RootCmd.Flags().StringVarP(&mode, "mode", "m", "", "debug or release")
	RootCmd.PersistentFlags().BoolVarP(&vers, "version", "v", false, "print version")
}
