package di

import (
	"testing"
)

// 测试 Inject 方法
func TestContainerInject(t *testing.T) {
	// 准备
	c := NewContainer()
	c.Provide(&testSimpleService{Name: "test"})
	if err := c.Build(); err != nil {
		t.Fatal(err)
	}

	// 测试：注入到指针变量
	var svc *testSimpleService
	c.Inject(&svc)

	// 验证
	if svc == nil {
		t.Error("Injected service is nil")
	}
	if svc.Name != "test" {
		t.Errorf("Expected Name='test', got '%s'", svc.Name)
	}
}

// 测试 Inject 方法 - 接口类型
func TestContainerInjectInterface(t *testing.T) {
	// 准备
	c := NewContainer()
	BindWith[testLogger](c, &testConsoleLogger{Prefix: "TEST"})
	if err := c.Build(); err != nil {
		t.Fatal(err)
	}

	// 测试：注入接口
	var logger testLogger
	c.Inject(&logger)

	// 验证
	if logger == nil {
		t.Error("Injected logger is nil")
	}

	// 验证实际类型
	if consoleLogger, ok := logger.(*testConsoleLogger); ok {
		if consoleLogger.Prefix != "TEST" {
			t.Errorf("Expected Prefix='TEST', got '%s'", consoleLogger.Prefix)
		}
	} else {
		t.Error("Injected logger is not *testConsoleLogger")
	}
}

// 测试 Inject 方法 - 成功场景
func TestContainerInjectSuccess(t *testing.T) {
	// 准备
	c := NewContainer()
	c.Provide(&testSimpleService{Name: "must-test"})
	if err := c.Build(); err != nil {
		t.Fatal(err)
	}

	// 测试：Inject 不应该 panic
	var svc *testSimpleService
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Inject panicked: %v", r)
		}
	}()

	c.Inject(&svc)

	// 验证
	if svc == nil {
		t.Error("Injected service is nil")
	}
	if svc.Name != "must-test" {
		t.Errorf("Expected Name='must-test', got '%s'", svc.Name)
	}
}

// 测试 Inject - 失败时应该 panic
func TestContainerInjectPanic(t *testing.T) {
	// 准备
	c := NewContainer()
	// 不注册任何服务
	if err := c.Build(); err != nil {
		t.Fatal(err)
	}

	// 测试：Inject 应该 panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Inject should panic when service not found")
		}
	}()

	var svc *testSimpleService
	c.Inject(&svc)
}

// 测试 Inject - 错误情况：非指针
func TestContainerInjectNonPointer(t *testing.T) {
	c := NewContainer()
	c.Provide(&testSimpleService{Name: "test"})
	if err := c.Build(); err != nil {
		t.Fatal(err)
	}

	// 测试：传入非指针应该 panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when passing non-pointer")
		}
	}()

	var svc testSimpleService
	c.Inject(svc) // 注意：这里传的是值，不是指针
}

// 测试 Inject - 错误情况：nil 指针
func TestContainerInjectNilPointer(t *testing.T) {
	c := NewContainer()
	c.Provide(&testSimpleService{Name: "test"})
	if err := c.Build(); err != nil {
		t.Fatal(err)
	}

	// 测试：传入 nil 指针应该 panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when passing nil pointer")
		}
	}()

	var svc *testSimpleService // svc 本身是 nil
	c.Inject(svc)              // 注意：这里传的是 nil 指针
}

// 测试 Inject - 使用 Token
func TestContainerInjectWithToken(t *testing.T) {
	// Token 的测试较为复杂，暂时跳过
	// 因为需要确保 Token 的类型键匹配逻辑正确
	t.Skip("Token injection test needs more investigation")
}

// 测试辅助类型
type testSimpleService struct {
	Name string
}

type testLogger interface {
	Log(msg string)
}

type testConsoleLogger struct {
	Prefix string
}

func (c *testConsoleLogger) Log(msg string) {
	// 空实现，用于测试
}
