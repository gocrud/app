package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gocrud/app/core"
)

// Run 启动应用程序
// 这是基于微内核架构的唯一入口
func Run(opts ...core.Option) error {
	rt := core.NewRuntime()

	// 1. Bootstrap (应用所有选项)
	// 这一步会配置 Feature、注册服务、添加生命周期钩子等
	for _, opt := range opts {
		if err := opt(rt); err != nil {
			return err
		}
	}

	// 2. Build DI Container (构建依赖注入容器)
	if err := rt.Container.Build(); err != nil {
		return err
	}

	// 3. Start Lifecycle (启动生命周期)
	// 创建根上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := rt.Lifecycle.Start(ctx, rt.Container); err != nil {
		return err
	}

	// 4. 阻塞并监听退出信号
	// 支持 OS 信号 (Ctrl+C, kill) 和 Runtime 内部触发的退出 (rt.Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-quit:
		// 收到系统信号
	case <-rt.Done():
		// 运行时内部请求退出 (例如关键服务崩溃)
	}

	// 5. Graceful Shutdown (优雅关闭)
	// 给定 5 秒超时时间用于清理
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	return rt.Lifecycle.Stop(shutdownCtx)
}
