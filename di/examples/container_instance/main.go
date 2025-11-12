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
	container.Inject(&logger)
	logger.Log("Hello from container instance")

	// 注入数据库
	var db Database
	container.Inject(&db)
	db.Connect()

	// 注入服务
	var svc *UserService
	container.Inject(&svc)
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
	container1.Inject(&logger1)

	var logger2 Logger
	container2.Inject(&logger2)

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
	container.Inject(&logger)
	logger.Log("Injected using var + Inject pattern")

	// 方式2: 使用 var + Inject 注入结构体
	var svc *UserService
	container.Inject(&svc)
	svc.Logger.Log("UserService injected using var + Inject")
	svc.DB.Connect()

	// 方式3: 注入数据库
	var db Database
	container.Inject(&db)
	db.Connect()

	// 演示：注入方式示例
	fmt.Println("\n--- 注入方式示例 ---")

	// 使用 var + Inject
	var logger1 Logger
	container.Inject(&logger1)
	logger1.Log("方式: var svc T; container.Inject(&svc)")

	var logger2 Logger
	container.Inject(&logger2)
	logger2.Log("批量注入示例")
}
