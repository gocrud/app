package core

import (
	"reflect"

	"github.com/gocrud/app/di"
)

// AddSingleton registers a singleton service.
// If impl is provided, it can be an instance or a factory function.
//
// Examples:
//
//	core.AddSingleton[IService](services) // Auto-register struct pointer or interface if implementation inferred
//	core.AddSingleton[IService](services, di.Use[*ServiceImpl]())
//	core.AddSingleton[IService](services, di.WithFactory(NewService))
func AddSingleton[T any](s *ServiceCollection, opts ...di.Option) {
	finalOpts := append([]di.Option{di.WithSingleton()}, opts...)
	di.Register[T](s.container, finalOpts...)
}

// AddTransient registers a transient service.
func AddTransient[T any](s *ServiceCollection, opts ...di.Option) {
	finalOpts := append([]di.Option{di.WithTransient()}, opts...)
	di.Register[T](s.container, finalOpts...)
}

// AddScoped registers a scoped service.
func AddScoped[T any](s *ServiceCollection, opts ...di.Option) {
	finalOpts := append([]di.Option{di.WithScoped()}, opts...)
	di.Register[T](s.container, finalOpts...)
}

// Helper to convert legacy "impl any" to options (internal use if needed, but we prefer explicit options)
// For backward compatibility wrappers if we wanted them:
func convertImplToOptions(impl any) []di.Option {
	if impl == nil {
		return nil
	}
	val := reflect.ValueOf(impl)
	// If function -> Factory
	if val.Kind() == reflect.Func {
		return []di.Option{di.WithFactory(impl)}
	}
	// If value -> Value (if not a type) - but wait, Register[T] expects static value via WithValue
	// or implementation type via Use[T].
	// Since 'impl' is 'any', strictly speaking if it's an instance we use WithValue.
	// If it's just a type hint, we can't easily use Use[T] because Use[T] requires compile-time type.
	// So we rely on WithValue for instances.
	return []di.Option{di.WithValue(impl)}
}
