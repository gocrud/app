package di

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

// 测试用接口和实现
type TestLogger interface {
	Log(msg string)
}

type ConsoleLogger struct {
	ID int
}

func (l *ConsoleLogger) Log(msg string) {
	fmt.Printf("[Logger %d] %s\n", l.ID, msg)
}

var loggerCounter int
var loggerMu sync.Mutex

func NewLogger() *ConsoleLogger {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	loggerCounter++
	return &ConsoleLogger{ID: loggerCounter}
}

type TestService struct {
	Logger TestLogger `di:""`
	Name   string
}

func NewTestService(logger TestLogger) *TestService {
	return &TestService{
		Logger: logger,
		Name:   "TestService",
	}
}

// Test Singleton scope - 应该只创建一次
func TestScopeSingleton(t *testing.T) {
	loggerCounter = 0
	container := NewContainer()

	// 注册为 Singleton（默认）
	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*TestLogger)(nil)).Elem(),
		UseType: NewLogger,
		Options: ProviderOptions{
			Scope: ScopeSingleton,
		},
	})

	if err := container.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 多次获取应该返回同一实例
	logger1, err := container.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger1: %v", err)
	}

	logger2, err := container.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger2: %v", err)
	}

	logger3, err := container.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger3: %v", err)
	}

	// 验证是同一实例
	if logger1.(*ConsoleLogger).ID != logger2.(*ConsoleLogger).ID {
		t.Errorf("Expected same instance, got different IDs: %d vs %d",
			logger1.(*ConsoleLogger).ID, logger2.(*ConsoleLogger).ID)
	}

	if logger2.(*ConsoleLogger).ID != logger3.(*ConsoleLogger).ID {
		t.Errorf("Expected same instance, got different IDs: %d vs %d",
			logger2.(*ConsoleLogger).ID, logger3.(*ConsoleLogger).ID)
	}

	// 验证只创建了一次
	if loggerCounter != 1 {
		t.Errorf("Expected logger to be created once, but created %d times", loggerCounter)
	}
}

// Test Transient scope - 每次都应该创建新实例
func TestScopeTransient(t *testing.T) {
	loggerCounter = 0
	container := NewContainer()

	// 注册为 Transient
	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*TestLogger)(nil)).Elem(),
		UseType: NewLogger,
		Options: ProviderOptions{
			Scope: ScopeTransient,
		},
	})

	if err := container.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 多次获取应该返回不同实例
	logger1, err := container.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger1: %v", err)
	}

	logger2, err := container.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger2: %v", err)
	}

	logger3, err := container.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger3: %v", err)
	}

	// 验证是不同实例
	if logger1.(*ConsoleLogger).ID == logger2.(*ConsoleLogger).ID {
		t.Errorf("Expected different instances, got same ID: %d", logger1.(*ConsoleLogger).ID)
	}

	if logger2.(*ConsoleLogger).ID == logger3.(*ConsoleLogger).ID {
		t.Errorf("Expected different instances, got same ID: %d", logger2.(*ConsoleLogger).ID)
	}

	// 验证创建了三次
	if loggerCounter != 3 {
		t.Errorf("Expected logger to be created 3 times, but created %d times", loggerCounter)
	}
}

// Test Scoped scope - 在同一作用域内应该是单例，不同作用域应该不同
func TestScopeScoped(t *testing.T) {
	loggerCounter = 0
	container := NewContainer()

	// 注册为 Scoped
	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*TestLogger)(nil)).Elem(),
		UseType: NewLogger,
		Options: ProviderOptions{
			Scope: ScopeScoped,
		},
	})

	if err := container.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 作用域 1
	scope1 := container.CreateScope()
	container.SetCurrentScope(scope1)

	logger1a, err := scope1.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger1a: %v", err)
	}

	logger1b, err := scope1.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger1b: %v", err)
	}

	// 在同一作用域内应该是同一实例
	if logger1a.(*ConsoleLogger).ID != logger1b.(*ConsoleLogger).ID {
		t.Errorf("Expected same instance in scope1, got different IDs: %d vs %d",
			logger1a.(*ConsoleLogger).ID, logger1b.(*ConsoleLogger).ID)
	}

	// 作用域 2
	scope2 := container.CreateScope()
	container.SetCurrentScope(scope2)

	logger2a, err := scope2.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger2a: %v", err)
	}

	logger2b, err := scope2.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger2b: %v", err)
	}

	// 在同一作用域内应该是同一实例
	if logger2a.(*ConsoleLogger).ID != logger2b.(*ConsoleLogger).ID {
		t.Errorf("Expected same instance in scope2, got different IDs: %d vs %d",
			logger2a.(*ConsoleLogger).ID, logger2b.(*ConsoleLogger).ID)
	}

	// 不同作用域应该是不同实例
	if logger1a.(*ConsoleLogger).ID == logger2a.(*ConsoleLogger).ID {
		t.Errorf("Expected different instances between scopes, got same ID: %d",
			logger1a.(*ConsoleLogger).ID)
	}

	// 验证创建了两次（每个作用域一次）
	if loggerCounter != 2 {
		t.Errorf("Expected logger to be created 2 times, but created %d times", loggerCounter)
	}

	// 清理
	scope1.Dispose()
	scope2.Dispose()
}

// Test Singleton 不能依赖 Transient
func TestSingletonCannotDependOnTransient(t *testing.T) {
	container := NewContainer()

	// 注册 Transient logger
	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*TestLogger)(nil)).Elem(),
		UseType: NewLogger,
		Options: ProviderOptions{
			Scope: ScopeTransient,
		},
	})

	// 注册 Singleton service 依赖 Transient logger（使用 Provide 直接注册）
	container.Provide(NewTestService)

	// Build 应该失败
	err := container.Build()
	if err == nil {
		t.Fatal("Expected Build to fail when Singleton depends on Transient, but it succeeded")
	}

	t.Logf("Got expected error: %v", err)
}

// Test Singleton 不能依赖 Scoped
func TestSingletonCannotDependOnScoped(t *testing.T) {
	container := NewContainer()

	// 注册 Scoped logger
	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*TestLogger)(nil)).Elem(),
		UseType: NewLogger,
		Options: ProviderOptions{
			Scope: ScopeScoped,
		},
	})

	// 注册 Singleton service 依赖 Scoped logger（使用 Provide 直接注册）
	container.Provide(NewTestService)

	// Build 应该失败
	err := container.Build()
	if err == nil {
		t.Fatal("Expected Build to fail when Singleton depends on Scoped, but it succeeded")
	}

	t.Logf("Got expected error: %v", err)
}

// Test Transient 可以依赖 Singleton
func TestTransientCanDependOnSingleton(t *testing.T) {
	loggerCounter = 0
	container := NewContainer()

	// 注册 Singleton logger
	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*TestLogger)(nil)).Elem(),
		UseType: NewLogger,
		Options: ProviderOptions{
			Scope: ScopeSingleton,
		},
	})

	// 注册 Transient service 依赖 Singleton logger
	container.ProvideFactory(FactoryProvider{
		Provide: reflect.TypeOf((*TestService)(nil)),
		Factory: NewTestService,
		Options: ProviderOptions{
			Scope: ScopeTransient,
		},
	})

	if err := container.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 获取两个 service 实例
	service1, err := container.GetByType(reflect.TypeOf((*TestService)(nil)))
	if err != nil {
		t.Fatalf("Failed to get service1: %v", err)
	}

	service2, err := container.GetByType(reflect.TypeOf((*TestService)(nil)))
	if err != nil {
		t.Fatalf("Failed to get service2: %v", err)
	}

	// Service 应该是不同实例（Transient）
	if service1.(*TestService) == service2.(*TestService) {
		t.Error("Expected different service instances (Transient)")
	}

	// 但它们的 Logger 应该是同一实例（Singleton）
	if service1.(*TestService).Logger.(*ConsoleLogger).ID != service2.(*TestService).Logger.(*ConsoleLogger).ID {
		t.Error("Expected same logger instance in different transient services")
	}

	// Logger 应该只创建一次
	if loggerCounter != 1 {
		t.Errorf("Expected logger to be created once, but created %d times", loggerCounter)
	}
}

// Test Scope Dispose
func TestScopeDispose(t *testing.T) {
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*TestLogger)(nil)).Elem(),
		UseType: NewLogger,
		Options: ProviderOptions{
			Scope: ScopeScoped,
		},
	})

	if err := container.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	scope := container.CreateScope()

	// 获取实例
	_, err := scope.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to get logger: %v", err)
	}

	// 释放作用域
	scope.Dispose()

	// 再次获取应该失败
	_, err = scope.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err == nil {
		t.Fatal("Expected Get to fail after Dispose, but it succeeded")
	}

	if err.Error() != "scope has been disposed" {
		t.Errorf("Expected 'scope has been disposed' error, got: %v", err)
	}
}

// Test 并发访问 Transient
func TestTransientConcurrency(t *testing.T) {
	loggerCounter = 0
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*TestLogger)(nil)).Elem(),
		UseType: NewLogger,
		Options: ProviderOptions{
			Scope: ScopeTransient,
		},
	})

	if err := container.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	ids := make([]int, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			logger, err := container.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
			if err != nil {
				errors[index] = err
				return
			}
			ids[index] = logger.(*ConsoleLogger).ID
		}(i)
	}

	wg.Wait()

	// 检查错误
	for i, err := range errors {
		if err != nil {
			t.Errorf("Goroutine %d failed: %v", i, err)
		}
	}

	// 验证所有 ID 都不相同
	idMap := make(map[int]bool)
	for _, id := range ids {
		if idMap[id] {
			t.Errorf("Duplicate ID found: %d", id)
		}
		idMap[id] = true
	}

	// 验证创建了正确的数量
	if loggerCounter != numGoroutines {
		t.Errorf("Expected %d instances, got %d", numGoroutines, loggerCounter)
	}
}

// Test 并发访问 Scoped
func TestScopedConcurrency(t *testing.T) {
	loggerCounter = 0
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*TestLogger)(nil)).Elem(),
		UseType: NewLogger,
		Options: ProviderOptions{
			Scope: ScopeScoped,
		},
	})

	if err := container.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	scope := container.CreateScope()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	ids := make([]int, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			logger, err := scope.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
			if err != nil {
				errors[index] = err
				return
			}
			ids[index] = logger.(*ConsoleLogger).ID
		}(i)
	}

	wg.Wait()

	// 检查错误
	for i, err := range errors {
		if err != nil {
			t.Errorf("Goroutine %d failed: %v", i, err)
		}
	}

	// 验证所有 ID 都相同（同一作用域内）
	firstID := ids[0]
	for _, id := range ids {
		if id != firstID {
			t.Errorf("Expected all IDs to be %d, got %d", firstID, id)
		}
	}

	// 验证只创建了一次
	if loggerCounter != 1 {
		t.Errorf("Expected 1 instance, got %d", loggerCounter)
	}

	scope.Dispose()
}

// Test 访问 Scoped 但没有设置当前作用域
func TestScopedWithoutCurrentScope(t *testing.T) {
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*TestLogger)(nil)).Elem(),
		UseType: NewLogger,
		Options: ProviderOptions{
			Scope: ScopeScoped,
		},
	})

	if err := container.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 不设置当前作用域，直接从容器获取
	_, err := container.GetByType(reflect.TypeOf((*TestLogger)(nil)).Elem())
	if err == nil {
		t.Fatal("Expected Get to fail without current scope, but it succeeded")
	}

	t.Logf("Got expected error: %v", err)
}
