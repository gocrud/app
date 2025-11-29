package core

import (
	"reflect"
	"sync"
)

// FeatureCollection 是一个类型安全的特性集合
// 用于存放 WebBuilder, DbBuilder 等构建时特性
type FeatureCollection struct {
	features sync.Map
}

// Set 注册一个特性
func (fc *FeatureCollection) Set(feature any) {
	typ := reflect.TypeOf(feature)
	fc.features.Store(typ, feature)
}

// Get 获取一个特性
func (fc *FeatureCollection) Get(typ reflect.Type) (any, bool) {
	return fc.features.Load(typ)
}

// GetFeature 泛型辅助函数，从 Runtime 获取特性
func GetFeature[T any](rt *Runtime) T {
	var zero T
	// 如果 T 是接口，reflect.TypeOf(zero) 会返回 nil (如果 zero 是 nil 接口)。
	// 正确的做法是用 reflect.TypeOf((*T)(nil)).Elem()

	targetType := reflect.TypeOf((*T)(nil)).Elem()

	if val, ok := rt.Features.Get(targetType); ok {
		return val.(T)
	}
	return zero
}
