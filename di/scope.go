package di

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// Scope 表示作用域生命周期上下文。
type Scope interface {
	Container
	// Dispose 释放与作用域关联的资源。
	Dispose()
}

type scopeEntry struct {
	val atomic.Value // 存储实例（如果尚未创建则为 nil）
	mu  sync.Mutex   // 用于创建此特定实例的锁
}

type scope struct {
	parent  *container
	entries []scopeEntry // 按 ServiceDefinition.ID 索引的数组
}

func newScope(parent *container) *scope {
	count := parent.serviceCount()
	return &scope{
		parent:  parent,
		entries: make([]scopeEntry, count),
	}
}

func (s *scope) Add(def *ServiceDefinition) error {
	return fmt.Errorf("di: 无法在作用域上注册服务")
}

func (s *scope) Build() error {
	return nil // 作用域已基于父容器构建
}

func (s *scope) CreateScope() Scope {
	return s.parent.CreateScope()
}

func (s *scope) Get(typ reflect.Type) (any, error) {
	// 1. 检查服务是否存在于父定义中
	def, ok := s.parent.definitions[typ]
	if !ok {
		return nil, fmt.Errorf("di: 未找到服务 %v", typ)
	}

	// 2. 处理不同作用域
	switch def.Scope {
	case ScopeSingleton:
		return s.parent.Get(typ)

	case ScopeTransient:
		// 使用此作用域作为容器创建新实例（用于依赖项）
		return s.parent.resolver.createInstance(s, def)

	case ScopeScoped:
		// 使用 ID 进行 O(1) 数组访问
		if def.ID < 0 || def.ID >= len(s.entries) {
			// 如果 ID 分配正确，这不应发生
			return nil, fmt.Errorf("di: 内部错误，无效的服务 ID %d", def.ID)
		}

		// 我们获取切片中条目的指针。
		// 由于切片大小在创建后是固定的，此指针是稳定的。
		entry := &s.entries[def.ID]

		// 快速路径：检查是否已创建
		if val := entry.val.Load(); val != nil {
			return val, nil
		}

		// 慢速路径：带锁创建
		entry.mu.Lock()
		defer entry.mu.Unlock()

		// 双重检查
		if val := entry.val.Load(); val != nil {
			return val, nil
		}

		// 创建实例
		instance, err := s.parent.resolver.createInstance(s, def)
		if err != nil {
			return nil, err
		}

		entry.val.Store(instance)
		return instance, nil
	}

	return nil, fmt.Errorf("di: 未知作用域 %v", def.Scope)
}

func (s *scope) Dispose() {
	// 释放引用以允许 GC
	// 我们不能在不锁定的情况下轻松地将切片调整为 0，
	// 但如果需要，我们可以清零条目。
	// 但是，由于 scope 通常是整体丢弃的，只需让它超出范围就足够了。
	// 如果需要显式清理（例如对实例调用 Close()），将在此处进行。
	// 目前，我们只清除切片以帮助 GC，如果 scope 对象本身保持活动状态（这很少见）。
	for i := range s.entries {
		s.entries[i].val.Store(nil)
	}
	s.entries = nil
}

// serviceCount 委托给父容器
func (s *scope) serviceCount() int {
	return s.parent.serviceCount()
}
