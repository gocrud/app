package di

import (
	"reflect"
	"sync"
	"testing"
)

// ===== 基准测试用的类型定义 =====

type BenchLogger interface {
	Log(msg string)
}

type BenchConsoleLogger struct {
	ID int
}

func (l *BenchConsoleLogger) Log(msg string) {}

var benchLoggerCounter int
var benchLoggerMu sync.Mutex

func NewBenchLogger() BenchLogger {
	benchLoggerMu.Lock()
	defer benchLoggerMu.Unlock()
	benchLoggerCounter++
	return &BenchConsoleLogger{ID: benchLoggerCounter}
}

type BenchService struct {
	Logger BenchLogger `di:""`
	Name   string
}

func NewBenchService(logger BenchLogger) *BenchService {
	return &BenchService{
		Logger: logger,
		Name:   "BenchService",
	}
}

type BenchRepository struct {
	Logger BenchLogger `di:""`
}

func NewBenchRepository(logger BenchLogger) *BenchRepository {
	return &BenchRepository{Logger: logger}
}

// ===== Singleton 作用域压测 =====

func BenchmarkSingletonGet(b *testing.B) {
	benchLoggerCounter = 0
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeSingleton},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = container.GetByType(reflect.TypeOf((*BenchLogger)(nil)).Elem())
	}
}

func BenchmarkSingletonGetParallel(b *testing.B) {
	benchLoggerCounter = 0
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeSingleton},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = container.GetByType(reflect.TypeOf((*BenchLogger)(nil)).Elem())
		}
	})
}

// ===== Transient 作用域压测 =====

func BenchmarkTransientGet(b *testing.B) {
	benchLoggerCounter = 0
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeTransient},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = container.GetByType(reflect.TypeOf((*BenchLogger)(nil)).Elem())
	}
}

func BenchmarkTransientGetParallel(b *testing.B) {
	benchLoggerCounter = 0
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeTransient},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = container.GetByType(reflect.TypeOf((*BenchLogger)(nil)).Elem())
		}
	})
}

// ===== Scoped 作用域压测 =====

func BenchmarkScopedGet(b *testing.B) {
	benchLoggerCounter = 0
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeScoped},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	scope := container.CreateScope()
	defer scope.Dispose()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scope.GetByType(reflect.TypeOf((*BenchLogger)(nil)).Elem())
	}
}

func BenchmarkScopedGetParallel(b *testing.B) {
	benchLoggerCounter = 0
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeScoped},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	scope := container.CreateScope()
	defer scope.Dispose()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = scope.GetByType(reflect.TypeOf((*BenchLogger)(nil)).Elem())
		}
	})
}

// ===== 多作用域场景压测 =====

func BenchmarkMultipleScopesSequential(b *testing.B) {
	benchLoggerCounter = 0
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeScoped},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scope := container.CreateScope()
		_, _ = scope.GetByType(reflect.TypeOf((*BenchLogger)(nil)).Elem())
		scope.Dispose()
	}
}

func BenchmarkMultipleScopesParallel(b *testing.B) {
	benchLoggerCounter = 0
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeScoped},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			scope := container.CreateScope()
			_, _ = scope.GetByType(reflect.TypeOf((*BenchLogger)(nil)).Elem())
			scope.Dispose()
		}
	})
}

// ===== 复杂依赖场景压测 =====

func BenchmarkComplexDependencySingleton(b *testing.B) {
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeSingleton},
	})

	// 使用 Provide 直接注册
	container.Provide(NewBenchRepository)

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = container.GetByType(reflect.TypeOf((*BenchRepository)(nil)))
	}
}

func BenchmarkComplexDependencyTransient(b *testing.B) {
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeSingleton},
	})

	container.ProvideFactory(FactoryProvider{
		Provide: reflect.TypeOf((*BenchService)(nil)),
		Factory: NewBenchService,
		Options: ProviderOptions{Scope: ScopeTransient},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = container.GetByType(reflect.TypeOf((*BenchService)(nil)))
	}
}

func BenchmarkComplexDependencyScoped(b *testing.B) {
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeSingleton},
	})

	// 使用 ProvideFactory 并设置为 Scoped
	container.ProvideFactory(FactoryProvider{
		Provide: reflect.TypeOf((*BenchRepository)(nil)),
		Factory: NewBenchRepository,
		Options: ProviderOptions{Scope: ScopeScoped},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	scope := container.CreateScope()
	defer scope.Dispose()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scope.GetByType(reflect.TypeOf((*BenchRepository)(nil)))
	}
}

// ===== HTTP 请求模拟场景 =====

func BenchmarkHTTPRequestSimulation(b *testing.B) {
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeSingleton},
	})

	container.ProvideFactory(FactoryProvider{
		Provide: reflect.TypeOf((*BenchRepository)(nil)),
		Factory: NewBenchRepository,
		Options: ProviderOptions{Scope: ScopeScoped},
	})

	container.ProvideFactory(FactoryProvider{
		Provide: reflect.TypeOf((*BenchService)(nil)),
		Factory: NewBenchService,
		Options: ProviderOptions{Scope: ScopeScoped},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scope := container.CreateScope()
		_, _ = scope.GetByType(reflect.TypeOf((*BenchService)(nil)))
		_, _ = scope.GetByType(reflect.TypeOf((*BenchRepository)(nil)))
		scope.Dispose()
	}
}

func BenchmarkHTTPRequestSimulationParallel(b *testing.B) {
	container := NewContainer()

	container.ProvideType(TypeProvider{
		Provide: reflect.TypeOf((*BenchLogger)(nil)).Elem(),
		UseType: NewBenchLogger,
		Options: ProviderOptions{Scope: ScopeSingleton},
	})

	container.ProvideFactory(FactoryProvider{
		Provide: reflect.TypeOf((*BenchRepository)(nil)),
		Factory: NewBenchRepository,
		Options: ProviderOptions{Scope: ScopeScoped},
	})

	container.ProvideFactory(FactoryProvider{
		Provide: reflect.TypeOf((*BenchService)(nil)),
		Factory: NewBenchService,
		Options: ProviderOptions{Scope: ScopeScoped},
	})

	if err := container.Build(); err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			scope := container.CreateScope()
			_, _ = scope.GetByType(reflect.TypeOf((*BenchService)(nil)))
			_, _ = scope.GetByType(reflect.TypeOf((*BenchRepository)(nil)))
			scope.Dispose()
		}
	})
}
