package di_test

import (
	"testing"

	"github.com/gocrud/app/di"
)

type ContainerLogger interface {
	Log(msg string) string
}

type ContainerConsoleLogger struct {
	Prefix string
}

func (c *ContainerConsoleLogger) Log(msg string) string {
	return c.Prefix + ": " + msg
}

// 测试容器实例的 Provide 方法
func TestContainerProvide(t *testing.T) {
	container := di.NewContainer()

	logger := &ContainerConsoleLogger{Prefix: "TEST"}
	container.Provide(logger)

	if err := container.Build(); err != nil {
		t.Fatalf("container.Build failed: %v", err)
	}

	var result *ContainerConsoleLogger
	container.Inject(&result)
	if result.Prefix != "TEST" {
		t.Errorf("expected prefix 'TEST', got '%s'", result.Prefix)
	}
}

// 测试 BindWith 泛型函数
func TestBindWith(t *testing.T) {
	container := di.NewContainer()

	logger := &ContainerConsoleLogger{Prefix: "BINDWITH"}
	di.BindWith[ContainerLogger](container, logger)

	container.Build()

	var result ContainerLogger
	container.Inject(&result)
	msg := result.Log("test")
	expected := "BINDWITH: test"
	if msg != expected {
		t.Errorf("expected '%s', got '%s'", expected, msg)
	}
}

// 测试 Inject 方法
func TestInject(t *testing.T) {
	container := di.NewContainer()

	logger := &ContainerConsoleLogger{Prefix: "INJECT"}
	container.Provide(logger)
	container.Build()

	var result *ContainerConsoleLogger
	container.Inject(&result)
	if result.Prefix != "INJECT" {
		t.Errorf("expected prefix 'INJECT', got '%s'", result.Prefix)
	}
}

// 测试 Inject 带错误处理
func TestInjectWithError(t *testing.T) {
	container := di.NewContainer()

	logger := &ContainerConsoleLogger{Prefix: "TRY"}
	container.Provide(logger)
	container.Build()

	var result *ContainerConsoleLogger
	// Inject 不再返回错误，改用 panic 测试
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Inject panicked: %v", r)
		}
	}()
	container.Inject(&result)
	if result.Prefix != "TRY" {
		t.Errorf("expected prefix 'TRY', got '%s'", result.Prefix)
	}
}

// 测试 Inject 失败情况
func TestInjectNotFound(t *testing.T) {
	container := di.NewContainer()
	container.Build()

	var result *ContainerConsoleLogger
	// Inject 不再返回错误，应该 panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when injecting non-existent type")
		}
	}()
	container.Inject(&result)
}

// 移除 FromDefault 相关测试，已经不再支持
// 使用 Inject 带默认值的方式替代

// 测试多容器隔离
func TestMultipleContainerIsolation(t *testing.T) {
	container1 := di.NewContainer()
	container2 := di.NewContainer()

	logger1 := &ContainerConsoleLogger{Prefix: "CONTAINER1"}
	logger2 := &ContainerConsoleLogger{Prefix: "CONTAINER2"}

	di.BindWith[ContainerLogger](container1, logger1)
	di.BindWith[ContainerLogger](container2, logger2)

	container1.Build()
	container2.Build()

	var result1 ContainerLogger
	var result2 ContainerLogger
	container1.Inject(&result1)
	container2.Inject(&result2)

	msg1 := result1.Log("test")
	msg2 := result2.Log("test")

	if msg1 != "CONTAINER1: test" {
		t.Errorf("container1: expected 'CONTAINER1: test', got '%s'", msg1)
	}
	if msg2 != "CONTAINER2: test" {
		t.Errorf("container2: expected 'CONTAINER2: test', got '%s'", msg2)
	}
}

// 测试容器实例的 ProvideType 方法
func TestContainerProvideType(t *testing.T) {
	container := di.NewContainer()

	logger := &ContainerConsoleLogger{Prefix: "PROVIDETYPE"}
	container.ProvideType(di.TypeProvider{
		Provide: di.TypeOf[ContainerLogger](),
		UseType: logger,
	})

	container.Build()

	var result ContainerLogger
	container.Inject(&result)
	msg := result.Log("test")
	expected := "PROVIDETYPE: test"
	if msg != expected {
		t.Errorf("expected '%s', got '%s'", expected, msg)
	}
}
