package core

import "fmt"

// Extension 定义应用程序扩展的基础接口
// 扩展模块应该实现 ServiceConfigurator 或 AppConfigurator 接口（或两者都实现）
type Extension interface {
	// Name 返回扩展的名称，用于日志记录和调试
	Name() string
}

// ServiceConfigurator 负责注册依赖注入服务
// 对应应用程序启动的 ConfigureServices 阶段
type ServiceConfigurator interface {
	// ConfigureServices 在此方法中注册服务到 DI 容器
	ConfigureServices(services *ServiceCollection)
}

// AppConfigurator 负责配置应用程序构建上下文
// 对应应用程序启动的 Configure 阶段，用于设置 Options、HostedService 等
type AppConfigurator interface {
	// ConfigureBuilder 在此方法中配置构建上下文
	ConfigureBuilder(ctx *BuildContext)
}

// validateExtension 验证扩展是否实现了支持的接口
// 如果未实现任何支持的接口，将 panic
func validateExtension(ext Extension) {
	_, isServiceConfigurator := ext.(ServiceConfigurator)
	_, isAppConfigurator := ext.(AppConfigurator)

	if !isServiceConfigurator && !isAppConfigurator {
		panic(fmt.Sprintf("app: Extension '%s' does not implement any supported interfaces (ServiceConfigurator, AppConfigurator). \n"+
			"Check if your method signatures exactly match the interface definitions.", ext.Name()))
	}
}

