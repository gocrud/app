package di

import (
	"fmt"
	"reflect"
)

// Token 表示一个依赖注入的令牌，用于区分相同类型的不同依赖
//
// 使用场景：
//   - 需要注册多个相同类型但用途不同的实例（如多个数据库连接）
//   - 配置值（如字符串、整数等基本类型）
//
// 示例：
//
//	// 定义 Token
//	var DBConnectionString = di.NewToken[string]("db-connection")
//	var CacheConnectionString = di.NewToken[string]("cache-connection")
//
//	// 注册
//	di.ProvideValue(di.ValueProvider{
//		Provide: DBConnectionString,
//		Value: "postgres://...",
//	})
//
//	// 获取
//	conn, _ := di.Get(DBConnectionString)
type Token[T any] struct {
	name string
	typ  reflect.Type
}

// NewToken 创建一个新的 Token
//
// 参数 name 用于标识此 Token，应该是唯一的描述性名称。
func NewToken[T any](name string) *Token[T] {
	return &Token[T]{
		name: name,
		typ:  reflect.TypeOf((*T)(nil)).Elem(),
	}
}

// Name 返回 Token 的名称
func (t *Token[T]) Name() string {
	return t.name
}

// Type 返回 Token 的类型
func (t *Token[T]) Type() reflect.Type {
	return t.typ
}

// String 返回 Token 的字符串表示
func (t *Token[T]) String() string {
	return fmt.Sprintf("Token[%s](%s)", t.typ, t.name)
}

// TypeOf 获取类型 T 的 reflect.Type（泛型辅助函数）
//
// 这是一个便捷函数，用于在泛型代码中获取类型信息。
//
// 示例：
//
//	userServiceType := di.TypeOf[UserService]()
//	instance, _ := container.GetByType(userServiceType)
func TypeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}
