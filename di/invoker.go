package di

import (
	"fmt"
	"reflect"
)

// Invoker 实例化调用器
// 封装了反射调用的细节，预先检查错误和返回值
type Invoker func(args []reflect.Value) (any, error)

// createConstructorInvoker 创建构造函数调用器
func createConstructorInvoker(info *providerInfo) Invoker {
	fn := reflect.ValueOf(info.value)
	
	return func(args []reflect.Value) (any, error) {
		results := fn.Call(args)
		if len(results) == 0 {
			return nil, fmt.Errorf("constructor returned no values")
		}

		// 检查 error
		if len(results) > 1 {
			lastResult := results[len(results)-1]
			if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				if !lastResult.IsNil() {
					return nil, fmt.Errorf("constructor failed: %w", lastResult.Interface().(error))
				}
			}
		}

		// 检查 nil
		firstResult := results[0]
		if firstResult.Kind() == reflect.Ptr || firstResult.Kind() == reflect.Interface {
			if firstResult.IsNil() {
				return nil, fmt.Errorf("constructor returned nil instance")
			}
		}

		return firstResult.Interface(), nil
	}
}

// createFactoryInvoker 创建工厂函数调用器
func createFactoryInvoker(info *providerInfo) Invoker {
	fn := reflect.ValueOf(info.value)
	
	return func(args []reflect.Value) (any, error) {
		results := fn.Call(args)
		if len(results) == 0 {
			return nil, fmt.Errorf("factory returned no values")
		}

		// 检查 error
		if len(results) > 1 {
			lastResult := results[len(results)-1]
			if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				if !lastResult.IsNil() {
					return nil, fmt.Errorf("factory failed: %w", lastResult.Interface().(error))
				}
			}
		}

		// 检查 nil
		firstResult := results[0]
		if firstResult.Kind() == reflect.Ptr || firstResult.Kind() == reflect.Interface {
			if firstResult.IsNil() {
				return nil, fmt.Errorf("factory returned nil instance")
			}
		}

		return firstResult.Interface(), nil
	}
}

