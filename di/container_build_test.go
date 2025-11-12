package di

import (
	"sync"
	"testing"
)

// TestBuildIdempotent 测试 Build() 方法的幂等性
func TestBuildIdempotent(t *testing.T) {
	container := NewContainer()

	// 注册一个简单的服务
	type TestService struct {
		Value int
	}

	container.Provide(&TestService{Value: 42})

	// 第一次构建
	err := container.Build()
	if err != nil {
		t.Fatalf("First Build() failed: %v", err)
	}

	// 第二次构建应该成功（幂等性）
	err = container.Build()
	if err != nil {
		t.Errorf("Second Build() should succeed (idempotent), but got error: %v", err)
	}

	// 第三次构建也应该成功
	err = container.Build()
	if err != nil {
		t.Errorf("Third Build() should succeed (idempotent), but got error: %v", err)
	}

	// 验证服务仍然可以正常获取
	instance, err := container.GetByType(TypeOf[*TestService]())
	if err != nil {
		t.Fatalf("GetByType failed: %v", err)
	}

	svc := instance.(*TestService)
	if svc.Value != 42 {
		t.Errorf("Expected Value=42, got %d", svc.Value)
	}
}

// TestBuildConcurrent 测试 Build() 方法的并发安全性
func TestBuildConcurrent(t *testing.T) {
	container := NewContainer()

	// 注册一个服务
	type TestService struct {
		Value string
	}

	container.Provide(&TestService{Value: "concurrent"})

	// 并发调用 Build()
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if err := container.Build(); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Errorf("Concurrent Build() failed: %v", err)
	}

	// 验证服务可以正常获取
	instance, err := container.GetByType(TypeOf[*TestService]())
	if err != nil {
		t.Fatalf("GetByType failed after concurrent builds: %v", err)
	}

	svc := instance.(*TestService)
	if svc.Value != "concurrent" {
		t.Errorf("Expected Value='concurrent', got %s", svc.Value)
	}
}

// TestBuildAfterProvideError 测试在 Build() 后无法再注册服务
func TestBuildAfterProvideError(t *testing.T) {
	container := NewContainer()

	// 注册第一个服务
	type Service1 struct{}
	container.Provide(&Service1{})

	// 构建容器
	err := container.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	// 尝试在 Build() 后注册新服务（应该 panic）
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when Provide() is called after Build(), but no panic occurred")
		}
	}()

	type Service2 struct{}
	container.Provide(&Service2{})
}

// TestBuildWithSingletonDependencies 测试构建包含依赖关系的单例服务
func TestBuildWithSingletonDependencies(t *testing.T) {
	container := NewContainer()

	type Logger struct {
		Name string
	}

	type Service struct {
		Logger *Logger
	}

	// 注册依赖
	container.Provide(&Logger{Name: "AppLogger"})

	// 注册服务（构造函数依赖 Logger）
	container.Provide(func(logger *Logger) *Service {
		return &Service{Logger: logger}
	})

	// 第一次构建
	err := container.Build()
	if err != nil {
		t.Fatalf("First Build() failed: %v", err)
	}

	// 第二次构建应该成功（幂等）
	err = container.Build()
	if err != nil {
		t.Errorf("Second Build() should be idempotent: %v", err)
	}

	// 验证服务正确构建
	instance, err := container.GetByType(TypeOf[*Service]())
	if err != nil {
		t.Fatalf("GetByType failed: %v", err)
	}

	svc := instance.(*Service)
	if svc.Logger == nil {
		t.Error("Service.Logger should not be nil")
	}
	if svc.Logger.Name != "AppLogger" {
		t.Errorf("Expected Logger.Name='AppLogger', got %s", svc.Logger.Name)
	}
}

// TestBuildEmptyContainer 测试构建空容器
func TestBuildEmptyContainer(t *testing.T) {
	container := NewContainer()

	// 空容器也应该可以成功构建
	err := container.Build()
	if err != nil {
		t.Errorf("Build() on empty container should succeed: %v", err)
	}

	// 多次构建应该都成功
	err = container.Build()
	if err != nil {
		t.Errorf("Second Build() on empty container should succeed: %v", err)
	}
}
