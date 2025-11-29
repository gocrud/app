package di

import (
	"fmt"
	"reflect"
)

// RegisterAuto 智能注册服务。
// 它可以接受构造函数、结构体指针或类型，并自动推断服务类型和注册方式。
//
// 支持的输入 target 类型:
// 1. func(...) (Service, error?) -> 注册为 Factory，ServiceType 为第一个返回值。
// 2. *Struct                      -> 注册为 Value (Singleton)，ServiceType 为 *Struct。
//   - 如果结构体包含带有 `di` 标签的字段，会自动启用 WithFields。
//
// 3. reflect.Type                 -> 注册为 Implementation (Struct注入)，ServiceType 为该 Type。
func RegisterAuto(c Container, target any, opts ...Option) (reflect.Type, error) {
	targetVal := reflect.ValueOf(target)
	var def *ServiceDefinition
	var serviceType reflect.Type

	// 1. 处理 reflect.Type (类型注册)
	if typeVal, ok := target.(reflect.Type); ok {
		serviceType = typeVal
		def = &ServiceDefinition{
			Type:     serviceType,
			Scope:    ScopeSingleton, // 默认单例
			ImplType: serviceType,
			// IsFactory/IsValue 默认为 false，即 struct 注入模式
		}
	} else if targetVal.Kind() == reflect.Func {
		// 2. 处理 Function (构造函数)
		fnType := targetVal.Type()
		if fnType.NumOut() == 0 {
			return nil, fmt.Errorf("di: constructor function must return at least one value")
		}

		// 推断服务类型为第一个返回值
		serviceType = fnType.Out(0)

		def = &ServiceDefinition{
			Type:      serviceType,
			Scope:     ScopeSingleton,
			Impl:      target,
			IsFactory: true,
		}
	} else if targetVal.Kind() == reflect.Ptr {
		// 3. 处理 Pointer (预初始化实例)
		// 必须是指向结构体的指针，或者是接口（但在运行时 any->interface 会解包，这里通常是指针）
		serviceType = targetVal.Type()

		def = &ServiceDefinition{
			Type:    serviceType,
			Scope:   ScopeSingleton,
			Impl:    target,
			IsValue: true,
		}

		// 智能检测：如果结构体有字段带 di 标签，则自动开启注入
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
		return nil, fmt.Errorf("di: unsupported auto-registration target type: %T", target)
	}

	// 应用选项
	for _, opt := range opts {
		opt(def)
	}

	// 注册
	// 注意：Add 方法如果发现 key 已存在会报错。
	// 对于自动注册，有时可能需要忽略已存在的情况，但标准行为应与 Register 一致，即报错。
	if err := c.Add(def); err != nil {
		return nil, err
	}

	return serviceType, nil
}

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
