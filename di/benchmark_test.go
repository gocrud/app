package di_test

import (
	"testing"

	"github.com/gocrud/app/di"
)

// Benchmarking Structures
type IService interface {
	Do()
}

type ServiceImpl struct{}

func (s *ServiceImpl) Do() {}

type LargeStruct struct {
	S1 *ServiceImpl `di:""`
	S2 *ServiceImpl `di:""`
	S3 *ServiceImpl `di:""`
	S4 *ServiceImpl `di:""`
	S5 *ServiceImpl `di:""`
}

func BenchmarkResolve_Singleton(b *testing.B) {
	c := di.NewContainer()
	di.Provide(c, &ServiceImpl{}) // Implicit Singleton
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = di.Get[*ServiceImpl](c)
	}
}

func BenchmarkResolve_Singleton_Parallel(b *testing.B) {
	c := di.NewContainer()
	di.ProvideService[*ServiceImpl](c)
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = di.Get[*ServiceImpl](c)
		}
	})
}

func BenchmarkResolve_Transient(b *testing.B) {
	c := di.NewContainer()
	di.ProvideService[*ServiceImpl](c, di.WithTransient())
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = di.Get[*ServiceImpl](c)
	}
}

func BenchmarkResolve_Transient_Parallel(b *testing.B) {
	c := di.NewContainer()
	di.ProvideService[*ServiceImpl](c, di.WithTransient())
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = di.Get[*ServiceImpl](c)
		}
	})
}

func BenchmarkResolve_Scoped(b *testing.B) {
	c := di.NewContainer()
	di.ProvideService[*ServiceImpl](c, di.WithScoped())
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	scope := c.CreateScope()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = di.Get[*ServiceImpl](scope)
	}
}

// BenchmarkResolve_Scoped_Parallel simulates multiple requests (goroutines)
// sharing the SAME scope. This tests the scope's internal locking.
func BenchmarkResolve_Scoped_Parallel_SharedScope(b *testing.B) {
	c := di.NewContainer()
	di.ProvideService[*ServiceImpl](c, di.WithScoped())
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	scope := c.CreateScope()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = di.Get[*ServiceImpl](scope)
		}
	})
}

// BenchmarkResolve_Scoped_Parallel_SeparateScopes simulates a real web server
// where each request (goroutine) has its OWN scope.
func BenchmarkResolve_Scoped_Parallel_SeparateScopes(b *testing.B) {
	c := di.NewContainer()
	di.ProvideService[*ServiceImpl](c, di.WithScoped())
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	b.RunParallel(func(pb *testing.PB) {
		scope := c.CreateScope()
		for pb.Next() {
			_, _ = di.Get[*ServiceImpl](scope)
		}
	})
}

func BenchmarkInjection_Field_Transient(b *testing.B) {
	c := di.NewContainer()
	di.ProvideService[*ServiceImpl](c, di.WithTransient())
	di.ProvideService[*LargeStruct](c, di.WithTransient())
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = di.Get[*LargeStruct](c)
	}
}

func BenchmarkResolve_Interface(b *testing.B) {
	c := di.NewContainer()
	di.ProvideService[IService](c, di.Use[*ServiceImpl]())
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = di.Get[IService](c)
	}
}

func TestConcurrency(t *testing.T) {
	c := di.NewContainer()
	di.ProvideService[*ServiceImpl](c, di.WithTransient())
	if err := c.Build(); err != nil {
		t.Fatal(err)
	}

	concurrency := 100
	done := make(chan bool)

	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < 1000; j++ {
				if _, err := di.Get[*ServiceImpl](c); err != nil {
					t.Errorf("Concurrent resolve failed: %v", err)
				}
			}
			done <- true
		}()
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}
}
