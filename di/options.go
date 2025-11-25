package di

import "reflect"

// Option configures a ServiceDefinition.
type Option func(*ServiceDefinition)

// Use specifies the implementation type.
// T is the concrete type that implements the service.
func Use[T any]() Option {
	return func(d *ServiceDefinition) {
		d.ImplType = reflect.TypeOf((*T)(nil)).Elem()
	}
}

// WithValue specifies a static value as the implementation.
func WithValue(v any) Option {
	return func(d *ServiceDefinition) {
		d.Impl = v
		d.IsValue = true
		d.Scope = ScopeSingleton // Values are implicitly singletons
	}
}

// WithFactory specifies a factory function to create the instance.
// The factory function must return the service type (or a pointer to it) and optionally an error.
func WithFactory(fn any) Option {
	return func(d *ServiceDefinition) {
		d.Impl = fn
		d.IsFactory = true
	}
}

// WithSingleton sets the scope to Singleton.
func WithSingleton() Option {
	return func(d *ServiceDefinition) {
		d.Scope = ScopeSingleton
	}
}

// WithTransient sets the scope to Transient.
func WithTransient() Option {
	return func(d *ServiceDefinition) {
		d.Scope = ScopeTransient
	}
}

// WithScoped sets the scope to Scoped.
func WithScoped() Option {
	return func(d *ServiceDefinition) {
		d.Scope = ScopeScoped
	}
}
