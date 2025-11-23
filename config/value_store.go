package config

import (
	"sync/atomic"
)

// ValueStore 使用 atomic.Value 存储配置数据，实现无锁读取
type ValueStore struct {
	value atomic.Value // stores map[string]any
}

// NewValueStore 创建新的 ValueStore
func NewValueStore() *ValueStore {
	s := &ValueStore{}
	s.value.Store(make(map[string]any))
	return s
}

// Load 加载当前配置快照
func (s *ValueStore) Load() map[string]any {
	val := s.value.Load()
	if val == nil {
		return nil
	}
	return val.(map[string]any)
}

// Store 原子替换配置数据
func (s *ValueStore) Store(data map[string]any) {
	s.value.Store(data)
}
