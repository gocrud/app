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
	di.Reset()

	// 使用 TypeOf 统一处理接口和具体类型
	di.ProvideType(di.TypeProvider{
		Provide: di.TypeOf[Logger](), // 自动检查是否为接口
		UseType: &ConsoleLogger{Prefix: "APP"},
	})

	di.ProvideType(di.TypeProvider{
		Provide: di.TypeOf[Database](), // 自动检查是否为接口
		UseType: &MySQLDatabase{Host: "localhost", Port: 3306},
	})

	di.ProvideType(di.TypeProvider{
		Provide: di.TypeOf[IInt](),
		UseType: Int(42),
	})

	// 注册服务
	di.Provide(&UserService{})

	// 构建容器
	di.MustBuild()

	// 方式1: 泛型 Inject
	println("\n=== 方式1: 泛型 Inject ===")
	svc := di.Inject[*UserService]()
	svc.Logger.Log("UserService initialized")
	svc.DB.Connect()

	intVal := di.Inject[IInt]()
	println("Injected int value:", intVal.Value())

	// 方式2: var + MustInject (推荐用于容器实例)
	println("\n=== 方式2: var + container.MustInject ===")
	container := di.GetContainer()

	var svc2 *UserService
	container.MustInject(&svc2)
	svc2.Logger.Log("UserService injected via MustInject")

	var logger2 Logger
	container.MustInject(&logger2)
	logger2.Log("Logger injected via MustInject")
}
