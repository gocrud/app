package main

import (
	"fmt"

	"github.com/gocrud/app/di"
)

// 定义接口
type Logger interface {
	Log(msg string)
}

type Database interface {
	Connect() error
}

// 实现
type ConsoleLogger struct {
	Prefix string
}

func (c *ConsoleLogger) Log(msg string) {
	fmt.Printf("%s: %s\n", c.Prefix, msg)
}

type MySQLDatabase struct {
	Host string
	Port int
}

func (m *MySQLDatabase) Connect() error {
	fmt.Printf("Connecting to MySQL at %s:%d\n", m.Host, m.Port)
	return nil
}

// 服务
type UserService struct {
	Logger Logger   `di:""`
	DB     Database `di:""`
}

func main() {
	fmt.Println("=== 示例1: 独立容器实例 - 使用 From ===")
	example1()

	fmt.Println("\n=== 示例2: 多容器隔离 ===")
	example2()

	fmt.Println("\n=== 示例3: 使用 Inject 指针方式 ===")
	example3()
}

// 示例1: 独立容器实例
func example1() {
	// 创建独立容器实例
	container := di.NewContainer()

	// 使用容器实例的方法注册
	container.Provide(&ConsoleLogger{Prefix: "INSTANCE"})
	di.BindWith[Logger](container, &ConsoleLogger{Prefix: "INSTANCE-LOGGER"})
	di.BindWith[Database](container, &MySQLDatabase{Host: "localhost", Port: 3306})
	container.Provide(&UserService{})

	// 构建容器
	if err := container.Build(); err != nil {
		panic(err)
	}

	// 从容器实例注入（使用 Inject）
	var logger Logger
	container.MustInject(&logger)
	logger.Log("Hello from container instance")

	// 使用 Inject 带错误处理
	var db Database
	if err := container.Inject(&db); err != nil {
		panic(err)
	}
	db.Connect()

	// 注入服务
	var svc *UserService
	container.MustInject(&svc)
	svc.Logger.Log("UserService initialized")
	svc.DB.Connect()
}

// 示例2: 多容器隔离
func example2() {
	// 创建两个独立的容器
	container1 := di.NewContainer()
	container2 := di.NewContainer()

	// 在container1中注册
	di.BindWith[Logger](container1, &ConsoleLogger{Prefix: "CONTAINER1"})
	container1.Build()

	// 在container2中注册
	di.BindWith[Logger](container2, &ConsoleLogger{Prefix: "CONTAINER2"})
	container2.Build()

	// 从不同容器获取实例
	var logger1 Logger
	container1.MustInject(&logger1)

	var logger2 Logger
	container2.MustInject(&logger2)

	logger1.Log("From container 1")
	logger2.Log("From container 2")
}

// 示例3: 使用 Inject 指针方式
func example3() {
	// 创建独立容器实例
	container := di.NewContainer()

	// 注册服务
	di.BindWith[Logger](container, &ConsoleLogger{Prefix: "INJECT"})
	di.BindWith[Database](container, &MySQLDatabase{Host: "localhost", Port: 3306})
	container.Provide(&UserService{})

	// 构建容器
	if err := container.Build(); err != nil {
		panic(err)
	}

	// 方式1: 使用 var + Inject 注入接口
	var logger Logger
	if err := container.Inject(&logger); err != nil {
		panic(err)
	}
	logger.Log("Injected using var + Inject pattern")

	// 方式2: 使用 var + Inject 注入结构体
	var svc *UserService
	if err := container.Inject(&svc); err != nil {
		panic(err)
	}
	svc.Logger.Log("UserService injected using var + Inject")
	svc.DB.Connect()

	// 方式3: 使用 MustInject（失败时 panic）
	var db Database
	container.MustInject(&db)
	db.Connect()

	// 演示：两种注入方式对比
	fmt.Println("\n--- 两种注入方式对比 ---")

	// 方式A: var + Inject（带错误处理）
	var logger1 Logger
	if err := container.Inject(&logger1); err != nil {
		panic(err)
	}
	logger1.Log("方式A: var svc T; container.Inject(&svc)")

	// 方式B: var + MustInject（简洁）
	var logger2 Logger
	container.MustInject(&logger2)
	logger2.Log("方式B: var svc T; container.MustInject(&svc)")
}
