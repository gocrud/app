package main

import "github.com/gocrud/app/di"

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
	println(c.Prefix + ": " + msg)
}

type MySQLDatabase struct {
	Host string
	Port int
}

func (m *MySQLDatabase) Connect() error {
	println("Connecting to MySQL at", m.Host, ":", m.Port)
	return nil
}

// 服务
type UserService struct {
	Logger Logger   `di:""`
	DB     Database `di:""`
}

type IInt interface {
	Value() int
}

type Int int

func (i Int) Value() int {
	return int(i)
}

func main() {
	// 创建容器实例
	container := di.NewContainer()

	// 使用 BindWith 绑定接口到实现
	di.BindWith[Logger](container, &ConsoleLogger{Prefix: "APP"})
	di.BindWith[Database](container, &MySQLDatabase{Host: "localhost", Port: 3306})
	di.BindWith[IInt](container, Int(42))

	// 注册服务
	container.Provide(&UserService{})

	// 构建容器
	if err := container.Build(); err != nil {
		panic(err)
	}

	// 使用 Inject 获取服务
	println("\n=== 使用 container.Inject 获取服务 ===")
	var svc *UserService
	container.Inject(&svc)
	svc.Logger.Log("UserService initialized")
	svc.DB.Connect()

	var intVal IInt
	container.Inject(&intVal)
	println("Injected int value:", intVal.Value())

	// 批量注入示例
	println("\n=== 批量注入示例 ===")
	var (
		logger Logger
		db     Database
	)
	container.Inject(&logger)
	container.Inject(&db)

	logger.Log("Logger injected")
	db.Connect()
}
