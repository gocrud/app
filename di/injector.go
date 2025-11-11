package di

import (
	"fmt"
	"reflect"
)

// Inject 从容器中注入类型T的实例
// 在Build后调用，直接从预构建的实例缓存中获取，零反射开销
// 支持两种用法：
// 1. di.Inject[T]() - 按类型注入
// 2. di.Inject[T](token) - 按 Token 注入
func Inject[T any](tokenOrNil ...any) T {
	var tk typeKey

	if len(tokenOrNil) > 0 && tokenOrNil[0] != nil {
		// 使用 Token 注入
		if token, ok := tokenOrNil[0].(tokenInterface); ok {
			tk = typeKey{typ: token.Type(), token: token}
		} else {
			panic("di.Inject: invalid token parameter")
		}
	} else {
		// 按类型注入
		typ := reflect.TypeOf((*T)(nil)).Elem()
		tk = typeKey{typ: typ}
	}

	instance, err := defaultContainer.Get(tk)
	if err != nil {
		panic("di.Inject failed: " + err.Error())
	}

	return instance.(T)
}

// TryInject 从容器中注入实例，返回实例和错误
func TryInject[T any](tokenOrNil ...any) (T, error) {
	var zero T
	var tk typeKey

	if len(tokenOrNil) > 0 && tokenOrNil[0] != nil {
		// 使用 Token 注入
		if token, ok := tokenOrNil[0].(tokenInterface); ok {
			tk = typeKey{typ: token.Type(), token: token}
		} else {
			var zeroVal T
			return zeroVal, fmt.Errorf("invalid token parameter")
		}
	} else {
		// 按类型注入
		typ := reflect.TypeOf((*T)(nil)).Elem()
		tk = typeKey{typ: typ}
	}

	instance, err := defaultContainer.Get(tk)
	if err != nil {
		return zero, err
	}

	return instance.(T), nil
}

// InjectOrDefault 从容器中注入实例，如果不存在则返回默认值
func InjectOrDefault[T any](defaultValue T, tokenOrNil ...any) T {
	instance, err := TryInject[T](tokenOrNil...)
	if err != nil {
		return defaultValue
	}
	return instance
}
