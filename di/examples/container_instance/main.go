package main

import (
	"fmt"

	"github.com/gocrud/app/di"
)

// Defines interfaces
type Logger interface {
	Log(msg string)
}

type Database interface {
	Connect() error
}

// Implementations
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

// Service
type UserService struct {
	Logger Logger   `di:""`
	DB     Database `di:""`
}

func main() {
	fmt.Println("=== Example 1: Independent Container Instances ===")
	example1()

	fmt.Println("\n=== Example 2: Multiple Containers Isolation ===")
	example2()
}

// Example 1: Independent Container
func example1() {
	container := di.NewContainer()

	// Register using new generic API
	// Bind interface Logger to ConsoleLogger
	di.Register[Logger](container, di.Use[*ConsoleLogger](), di.WithFactory(func() *ConsoleLogger {
		return &ConsoleLogger{Prefix: "INSTANCE-LOGGER"}
	}))

	// Register Database
	di.Register[Database](container, di.Use[*MySQLDatabase](), di.WithFactory(func() *MySQLDatabase {
		return &MySQLDatabase{Host: "localhost", Port: 3306}
	}))

	// Register UserService
	di.Register[*UserService](container)

	// Build
	if err := container.Build(); err != nil {
		panic(err)
	}

	// Resolve Logger
	logger, _ := di.Resolve[Logger](container)
	logger.Log("Hello from container instance")

	// Resolve Database
	db, _ := di.Resolve[Database](container)
	db.Connect()

	// Resolve Service
	svc, _ := di.Resolve[*UserService](container)
	svc.Logger.Log("UserService initialized")
	svc.DB.Connect()
}

// Example 2: Multiple Containers
func example2() {
	container1 := di.NewContainer()
	container2 := di.NewContainer()

	// Register in container1
	di.Register[Logger](container1, di.Use[*ConsoleLogger](), di.WithFactory(func() *ConsoleLogger {
		return &ConsoleLogger{Prefix: "CONTAINER1"}
	}))
	container1.Build()

	// Register in container2
	di.Register[Logger](container2, di.Use[*ConsoleLogger](), di.WithFactory(func() *ConsoleLogger {
		return &ConsoleLogger{Prefix: "CONTAINER2"}
	}))
	container2.Build()

	// Resolve
	logger1, _ := di.Resolve[Logger](container1)
	logger2, _ := di.Resolve[Logger](container2)

	logger1.Log("From container 1")
	logger2.Log("From container 2")
}
