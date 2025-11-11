package main

import "github.com/gocrud/app/di"

// 定义接口
type Logger interface {
	Log(msg string)
}

type Cache interface {
	Get(key string) string
	Set(key, value string)
}

type Metrics interface {
	Inc(name string)
}

// 实现
type ConsoleLogger struct {
	Prefix string
}

func (l *ConsoleLogger) Log(msg string) {
	println(l.Prefix + ": " + msg)
}

type MemoryCache struct{}

func (c *MemoryCache) Get(key string) string { return "" }
func (c *MemoryCache) Set(key, value string) {}

type PrometheusMetrics struct{}

func (m *PrometheusMetrics) Inc(name string) {}

// 服务 - 演示可选依赖
type UserService struct {
	Logger  Logger  `di:""`  // 必需：日志是必须的
	Cache   Cache   `di:"?"` // 可选：缓存可选
	Metrics Metrics `di:"?"` // 可选：监控可选
}

func (s *UserService) GetUser(id string) {
	s.Logger.Log("Getting user: " + id)

	// 安全使用可选依赖
	if s.Cache != nil {
		s.Cache.Get(id)
		s.Logger.Log("Cache hit")
	} else {
		s.Logger.Log("Cache not available")
	}

	if s.Metrics != nil {
		s.Metrics.Inc("user.get")
	}
}

func main() {
	di.Reset()

	// 场景 1: 只注册必需的依赖
	println("=== 场景 1: 最小依赖 ===")
	di.Bind[Logger](&ConsoleLogger{Prefix: "APP"})
	di.Provide(&UserService{})
	di.MustBuild()

	svc := di.Inject[*UserService]()
	svc.GetUser("user123")

	// 场景 2: 使用 InjectOrDefault 提供后备方案
	println("\n=== 场景 2: InjectOrDefault ===")
	di.Reset()
	di.Bind[Logger](&ConsoleLogger{Prefix: "APP"})
	di.Provide(&UserService{})
	di.MustBuild()

	// 尝试获取 Cache，如果不存在则使用默认实现
	defaultCache := &MemoryCache{}
	cache := di.InjectOrDefault[Cache](defaultCache)
	println("Got cache:", cache != nil)

	// 场景 3: 完整配置（所有依赖都注册）
	println("\n=== 场景 3: 完整依赖 ===")
	di.Reset()
	di.Bind[Logger](&ConsoleLogger{Prefix: "APP"})
	di.Bind[Cache](&MemoryCache{})
	di.Bind[Metrics](&PrometheusMetrics{})
	di.Provide(&UserService{})
	di.MustBuild()

	svc2 := di.Inject[*UserService]()
	svc2.GetUser("user456")

	// 场景 4: TryInject 检查依赖是否存在
	println("\n=== 场景 4: TryInject ===")
	if metrics, err := di.TryInject[Metrics](); err == nil {
		println("Metrics available:", metrics != nil)
	} else {
		println("Metrics not available:", err.Error())
	}

	// 尝试获取未注册的类型
	type UnknownService any
	if _, err := di.TryInject[UnknownService](); err != nil {
		println("UnknownService not found (expected):", err != nil)
	}
}
