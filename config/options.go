package config

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Option 静态配置选项（应用生命周期内不变）
// 在应用启动时加载一次，之后不再更新
type Option[T any] interface {
	Value() T
}

// OptionSnapshot 快照配置选项（作用域内不变）
// 在每个作用域创建时获取配置快照，同一作用域内保持不变
type OptionSnapshot[T any] interface {
	Value() T
}

// OptionMonitor 监听配置选项（实时更新，框架自动处理）
// 总是返回最新的配置值，框架会自动监听配置变更并更新
type OptionMonitor[T any] interface {
	Value() T
}

// OptionsCache 配置缓存，用于存储和自动更新配置
type OptionsCache[T any] struct {
	config  Configuration
	section string
	current T
	mu      sync.RWMutex
}

// NewOptionsCache 创建配置缓存
func NewOptionsCache[T any](config Configuration, section string) *OptionsCache[T] {
	cache := &OptionsCache[T]{
		config:  config,
		section: section,
	}

	// 初始加载配置
	if err := cache.reload(); err != nil {
		// 如果配置不存在，使用零值
		var zero T
		cache.current = zero
	}

	// 如果 Configuration 支持重载回调，则注册
	if rc, ok := config.(interface{ OnReload(func()) }); ok {
		rc.OnReload(func() {
			cache.reload()
		})
	}

	return cache
}

// reload 重新加载配置
func (c *OptionsCache[T]) reload() error {
	var newValue T

	// 从配置中绑定
	if err := c.config.Bind(c.section, &newValue); err != nil {
		return fmt.Errorf("failed to bind config section %s: %w", c.section, err)
	}

	c.mu.Lock()
	c.current = newValue
	c.mu.Unlock()

	return nil
}

// Get 获取当前配置值
func (c *OptionsCache[T]) Get() T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}

// Snapshot 创建当前配置的快照副本
func (c *OptionsCache[T]) Snapshot() T {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 深拷贝以避免共享状态
	var snapshot T
	data, err := json.Marshal(c.current)
	if err != nil {
		// 如果序列化失败，返回直接副本（可能是简单类型）
		return c.current
	}

	if err := json.Unmarshal(data, &snapshot); err != nil {
		return c.current
	}

	return snapshot
}

// option 实现 Option[T] 接口
type option[T any] struct {
	value T
}

func (o *option[T]) Value() T {
	return o.value
}

// NewOption 创建静态配置选项
func NewOption[T any](value T) Option[T] {
	return &option[T]{value: value}
}

// optionSnapshot 实现 OptionSnapshot[T] 接口
type optionSnapshot[T any] struct {
	snapshot T
}

func (o *optionSnapshot[T]) Value() T {
	return o.snapshot
}

// NewOptionSnapshot 创建快照配置选项
func NewOptionSnapshot[T any](snapshot T) OptionSnapshot[T] {
	return &optionSnapshot[T]{snapshot: snapshot}
}

// optionMonitor 实现 OptionMonitor[T] 接口
type optionMonitor[T any] struct {
	cache *OptionsCache[T]
}

func (o *optionMonitor[T]) Value() T {
	return o.cache.Get()
}

// NewOptionMonitor 创建监听配置选项
func NewOptionMonitor[T any](cache *OptionsCache[T]) OptionMonitor[T] {
	return &optionMonitor[T]{cache: cache}
}
