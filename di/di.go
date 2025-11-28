package di

import (
	"fmt"
	"reflect"
)

// Register registers a service of type T with the container.
// If T is an interface, you must use di.Use[Impl]() to specify the implementation.
func Register[T any](c Container, opts ...Option) {
	typ := reflect.TypeOf((*T)(nil)).Elem()

	def := &ServiceDefinition{
		Type:     typ,
		Scope:    ScopeSingleton, // Default scope
		ImplType: typ,            // Default implementation is the type itself
	}

	for _, opt := range opts {
		opt(def)
	}

	if err := c.Add(def); err != nil {
		panic(fmt.Sprintf("di: failed to register %v: %v", typ, err))
	}
}

// Resolve resolves an instance of type T from the container or scope.
func Resolve[T any](c Container) (T, error) {
	return ResolveNamed[T](c, "")
}

// ResolveNamed resolves an instance of type T with a specific name from the container or scope.
func ResolveNamed[T any](c Container, name string) (T, error) {
	var zero T
	typ := reflect.TypeOf((*T)(nil)).Elem()

	val, err := c.GetNamed(typ, name)
	if err != nil {
		return zero, err
	}

	if val == nil {
		// If the value is nil but no error, it might be a valid nil for pointers/interfaces,
		// but usually we expect a value.
		// However, for interface T, val should be convertible to T.
		return zero, nil
	}

	// Type assertion
	if v, ok := val.(T); ok {
		return v, nil
	}

	return zero, fmt.Errorf("di: resolved value is %T, expected %v", val, typ)
}
