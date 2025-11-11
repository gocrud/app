package di

import (
	"fmt"
	"reflect"
)

// Inject 通过指针注入实例到目标变量
// 用法示例：
//
//	var svc *UserService
//	c.Inject(&svc)
//
// 支持 Token 注入：
//
//	var svc *UserService
//	c.Inject(&svc, token)
func (c *Container) Inject(target any, tokenOrNil ...any) error {
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Pointer {
		return fmt.Errorf("Inject: target must be a pointer, got %v", targetVal.Kind())
	}

	if targetVal.IsNil() {
		return fmt.Errorf("Inject: target pointer is nil")
	}

	// 获取指针指向的元素类型
	elemVal := targetVal.Elem()
	elemType := elemVal.Type()

	var tk typeKey

	if len(tokenOrNil) > 0 && tokenOrNil[0] != nil {
		// 使用 Token 注入
		if token, ok := tokenOrNil[0].(tokenInterface); ok {
			tk = typeKey{typ: token.Type(), token: token}
		} else {
			return fmt.Errorf("Inject: invalid token parameter")
		}
	} else {
		// 按类型注入
		tk = typeKey{typ: elemType}
	}

	instance, err := c.Get(tk)
	if err != nil {
		return fmt.Errorf("Inject failed: %w", err)
	}

	// 设置值
	elemVal.Set(reflect.ValueOf(instance))
	return nil
}

// MustInject 通过指针注入实例，失败时 panic
func (c *Container) MustInject(target any, tokenOrNil ...any) {
	if err := c.Inject(target, tokenOrNil...); err != nil {
		panic(err)
	}
}
