package core

import "context"

// HostedService 定义了一个具有启动和停止生命周期的托管服务
// 这是框架中所有后台服务的标准接口。
type HostedService interface {
	// Start 启动服务
	// 框架会在独立的 Goroutine 中调用此方法，因此【允许阻塞】。
	// 通常在此方法中运行服务的主循环 (select loop)。
	// 如果方法返回 error，App 将记录错误并触发优雅关闭流程。
	Start(ctx context.Context) error

	// Stop 停止服务
	// 在应用关闭时调用。应执行优雅关闭逻辑（如等待请求处理完成）。
	// 必须支持通过 ctx 进行超时控制。
	Stop(ctx context.Context) error
}
