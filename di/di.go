package di

import (
	"fmt"
	"reflect"
)

// Provide registers a service automatically.
// It accepts a constructor function, a struct pointer, a reflect.Type, or any value (as Singleton).
//
// Supported targets:
// 1. func(...) (Service, error?) -> Registered as Factory. ServiceType is the first return value.
// 2. *Struct                      -> Registered as Value (Singleton). ServiceType is *Struct.
//   - If struct has fields with `di` tag, field injection is enabled.
//
// 3. reflect.Type                 -> Registered as Implementation (Struct injection). ServiceType is the Type.
// 4. Any value                    -> Registered as Value (Singleton). ServiceType is TypeOf(value).
func Provide(c Container, target any, opts ...Option) (reflect.Type, error) {
	targetVal := reflect.ValueOf(target)
	var def *ServiceDefinition
	var serviceType reflect.Type

	// 1. Handle reflect.Type (Type registration)
	if typeVal, ok := target.(reflect.Type); ok {
		serviceType = typeVal
		def = &ServiceDefinition{
			Type:     serviceType,
			Scope:    ScopeSingleton, // Default singleton
			ImplType: serviceType,
			// IsFactory/IsValue default to false (struct injection)
		}
	} else if targetVal.Kind() == reflect.Func {
		// 2. Handle Function (Constructor)
		fnType := targetVal.Type()
		if fnType.NumOut() == 0 {
			return nil, fmt.Errorf("di: constructor function must return at least one value")
		}

		// Infer service type from the first return value
		serviceType = fnType.Out(0)

		def = &ServiceDefinition{
			Type:      serviceType,
			Scope:     ScopeSingleton,
			Impl:      target,
			IsFactory: true,
		}
	} else if targetVal.Kind() == reflect.Ptr {
		// 3. Handle Pointer (Pre-initialized instance)
		// Must be a pointer to a struct
		serviceType = targetVal.Type()

		def = &ServiceDefinition{
			Type:    serviceType,
			Scope:   ScopeSingleton,
			Impl:    target,
			IsValue: true,
		}

		// Smart detection: if struct has fields with `di` tag, enable field injection
		if targetVal.Elem().Kind() == reflect.Struct {
			elemType := targetVal.Elem().Type()
			for i := 0; i < elemType.NumField(); i++ {
				if _, hasTag := elemType.Field(i).Tag.Lookup("di"); hasTag {
					def.InjectFields = true
					break
				}
			}
		}
	} else {
		// 4. Handle Value (Singleton) - Support for basic types (int, string) etc.
		serviceType = targetVal.Type()
		def = &ServiceDefinition{
			Type:    serviceType,
			Scope:   ScopeSingleton,
			Impl:    target,
			IsValue: true,
		}
	}

	// Apply options
	for _, opt := range opts {
		opt(def)
	}

	// Register
	if err := c.Add(def); err != nil {
		return nil, err
	}

	return serviceType, nil
}

// ProvideService registers a service of type T with the container.
// If T is an interface, you must use di.Use[Impl]() to specify the implementation.
func ProvideService[T any](c Container, opts ...Option) {
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
		panic(fmt.Sprintf("di: failed to provide %v: %v", typ, err))
	}
}

// Get resolves an instance of type T from the container or scope.
func Get[T any](c Container) (T, error) {
	return GetNamed[T](c, "")
}

// GetNamed resolves an instance of type T with a specific name from the container or scope.
func GetNamed[T any](c Container, name string) (T, error) {
	var zero T
	typ := reflect.TypeOf((*T)(nil)).Elem()

	val, err := c.GetNamed(typ, name)
	if err != nil {
		return zero, err
	}

	if val == nil {
		return zero, nil
	}

	// Type assertion
	if v, ok := val.(T); ok {
		return v, nil
	}

	return zero, fmt.Errorf("di: resolved value is %T, expected %v", val, typ)
}

// Invoke executes a function, injecting dependencies into its arguments.
// The function can return an error as its last return value.
func Invoke(c Container, fn any) error {
	fnVal := reflect.ValueOf(fn)
	if fnVal.Kind() != reflect.Func {
		return fmt.Errorf("di: invoke target must be a function")
	}
	fnType := fnVal.Type()

	// Prepare arguments
	args := make([]reflect.Value, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		argType := fnType.In(i)
		// Currently Invoke only supports type-based injection
		val, err := c.Get(argType)
		if err != nil {
			return fmt.Errorf("di: failed to resolve argument %d (%v): %w", i, argType, err)
		}
		args[i] = reflect.ValueOf(val)
	}

	// Call function
	results := fnVal.Call(args)

	// Check error return
	if len(results) > 0 {
		last := results[len(results)-1]
		if last.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !last.IsNil() {
				return last.Interface().(error)
			}
		}
	}

	return nil
}
