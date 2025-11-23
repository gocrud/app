package tests

import (
	"context"
	"testing"
	"time"

	"github.com/gocrud/app"
	"github.com/gocrud/app/config"
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
)

// IService 定义一个测试服务接口
type IService interface {
	SayHello() string
}

// ServiceImpl 实现测试服务接口
type ServiceImpl struct {
	// 使用 di 标签标记字段注入，? 表示可选
	Config config.Configuration `di:"?"`
	// 必须字段
	Logger logging.Logger `di:""`
}

// NewServiceImpl 是 ServiceImpl 的构造函数
func NewServiceImpl(logger logging.Logger) *ServiceImpl {
	return &ServiceImpl{Logger: logger}
}

func (s *ServiceImpl) SayHello() string {
	s.Logger.Info("Hello called")
	return "Hello"
}

// TestAppIntegration 测试整个应用框架的集成情况
// 包括：DI容器、配置读取、日志记录、应用生命周期
func TestAppIntegration(t *testing.T) {
	// 1. 创建应用构建器
	builder := app.NewApplicationBuilder()

	// 2. 配置 Configuration
	builder.ConfigureConfiguration(func(cb *config.ConfigurationBuilder) {
		// 使用内存配置源
		cb.AddInMemory(map[string]any{
			"app": map[string]any{
				"name":    "IntegrationTest",
				"version": 1,
			},
		})
	})

	// 3. 配置 Logging
	builder.ConfigureLogging(func(lb *logging.LoggingBuilder) {
		// 设置最低日志级别
		lb.SetMinimumLevel(logging.LogLevelDebug)
		// 添加控制台日志（内部使用了新实现的 AsyncWriter）
		lb.AddConsole()
	})

	// 4. 配置 Services (DI)
	builder.ConfigureServices(func(s *core.ServiceCollection) {
		// 注册单例服务 (构造函数方式)
		s.AddSingleton(NewServiceImpl)

		// 绑定接口到实现 (使用新的泛型语法糖)
		core.AddSingleton[IService](s, NewServiceImpl)
	})

	// 5. 构建应用
	application := builder.Build()

	// 6. 验证配置 (Config 优化验证)
	// 使用 Atomic 读取和 PathCache
	val := application.Configuration().Get("app:name")
	if val != "IntegrationTest" {
		t.Errorf("Expected app:name = IntegrationTest, got %s", val)
	}

	ver, err := application.Configuration().GetInt("app:version")
	if err != nil || ver != 1 {
		t.Errorf("Expected app:version = 1, got %d (err: %v)", ver, err)
	}

	// 7. 验证依赖注入 (DI 优化验证)
	container := application.Services()

	// 获取实现类实例
	svcImpl, err := container.GetByType(di.TypeOf[*ServiceImpl]())
	if err != nil {
		t.Fatalf("Failed to get *ServiceImpl: %v", err)
	}
	impl := svcImpl.(*ServiceImpl)
	if impl.SayHello() != "Hello" {
		t.Error("Service logic failed")
	}

	// 获取接口实例 (通过 BindWith 绑定)
	svcInterface, err := container.GetByType(di.TypeOf[IService]())
	if err != nil {
		t.Fatalf("Failed to get IService: %v", err)
	}
	if svcInterface.(IService).SayHello() != "Hello" {
		t.Error("Interface logic failed")
	}

	// 8. 验证应用运行生命周期 (Lifecycle)
	// 使用带超时的 Context 模拟运行
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// 异步运行
	errChan := make(chan error)
	go func() {
		// RunAsync 应该会运行直到 context 取消或 Stop 被调用
		errChan <- application.RunAsync(ctx)
	}()

	// 让应用运行一小段时间
	time.Sleep(100 * time.Millisecond)

	// 停止应用
	if err := application.Stop(context.Background()); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// 等待退出
	select {
	case err := <-errChan:
		// 正常退出或 context cancel 都是预期的
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Logf("RunAsync finished with error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Application did not stop in time")
	}
}
