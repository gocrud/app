package di

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// Container 是依赖注入容器的接口。
type Container interface {
	// Add 注册服务定义。
	Add(def *ServiceDefinition) error

	// Build 构建依赖图并进行验证。
	Build() error

	// Get 检索请求类型的实例（使用默认名称）。
	Get(typ reflect.Type) (any, error)

	// GetNamed 检索请求类型和名称的实例。
	GetNamed(typ reflect.Type, name string) (any, error)

	// CreateScope 为作用域实例创建一个新作用域。
	CreateScope() Scope

	// serviceCount 返回注册服务的总数（用于数组大小调整）。
	serviceCount() int
}

// container 是具体的实现。
type container struct {
	mu              sync.RWMutex
	definitions     map[ServiceKey]*ServiceDefinition
	built           atomic.Bool
	serviceCountVal int

	// resolver 处理实例的创建
	resolver *resolver
}

// NewContainer 创建一个新的空容器。
func NewContainer() Container {
	return &container{
		definitions: make(map[ServiceKey]*ServiceDefinition),
		resolver:    newResolver(),
	}
}

// Add 向容器添加服务定义。
func (c *container) Add(def *ServiceDefinition) error {
	if c.built.Load() {
		return fmt.Errorf("di: build 后无法注册服务")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	key := ServiceKey{Type: def.Type, Name: def.Name}

	if _, exists := c.definitions[key]; exists {
		if def.Name == "" {
			return fmt.Errorf("di: 服务 %v 已注册", def.Type)
		}
		return fmt.Errorf("di: 服务 %v (name=%s) 已注册", def.Type, def.Name)
	}

	c.definitions[key] = def
	return nil
}

// Build 构建依赖图并进行验证。
func (c *container) Build() error {
	if c.built.Load() {
		return nil // 已构建
	}

	c.mu.Lock()
	// 双重检查
	if c.built.Load() {
		c.mu.Unlock()
		return nil
	}

	// 0. 为定义分配 ID
	c.serviceCountVal = 0
	// 为了确保确定性顺序（虽然 map 迭代是随机的），
	// 只要 ID 唯一且在构建后一致，分配顺序并不重要。
	// 我们只需迭代并分配。
	for _, def := range c.definitions {
		def.ID = c.serviceCountVal
		c.serviceCountVal++
	}

	// 1. 依赖图和循环检测
	graph := newGraphBuilder(c.definitions)
	order, err := graph.buildOrder()
	if err != nil {
		c.mu.Unlock()
		return err
	}

	// 标记为已构建。此后，Add() 将失败，实际上使定义不可变。
	c.built.Store(true)
	c.mu.Unlock()

	// 2. 按拓扑顺序急切初始化单例
	// 我们在锁外执行此操作，以避免 Get() 锁定时死锁。
	for _, key := range order {
		def := c.definitions[key]
		if def.Scope == ScopeSingleton {
			if _, err := c.GetNamed(key.Type, key.Name); err != nil {
				return fmt.Errorf("di: 构建单例 %v (name=%s) 失败: %w", key.Type, key.Name, err)
			}
		}
	}

	return nil
}

// Get 检索请求类型的实例。
func (c *container) Get(typ reflect.Type) (any, error) {
	return c.GetNamed(typ, "")
}

// GetNamed 检索请求类型和名称的实例。
func (c *container) GetNamed(typ reflect.Type, name string) (any, error) {
	if !c.built.Load() {
		return nil, fmt.Errorf("di: 容器未构建")
	}

	key := ServiceKey{Type: typ, Name: name}

	// 优化：构建后定义是不可变的，因此我们可以无锁读取。
	// 这假设 c.built.Store(true) / Load() 提供了适当的内存屏障，Go 保证了这一点。
	def, ok := c.definitions[key]

	if !ok {
		if name == "" {
			return nil, fmt.Errorf("di: 未找到服务 %v", typ)
		}
		return nil, fmt.Errorf("di: 未找到服务 %v (name=%s)", typ, name)
	}

	// 单例：在定义本身上使用 sync.Once
	if def.Scope == ScopeSingleton {
		def.singletonOnce.Do(func() {
			def.singletonInst, def.singletonErr = c.resolver.createInstance(c, def)
		})
		return def.singletonInst, def.singletonErr
	}

	if def.Scope == ScopeTransient {
		return c.resolver.createInstance(c, def)
	}

	if def.Scope == ScopeScoped {
		return nil, fmt.Errorf("di: 无法从根容器解析作用域服务 %v。请使用 CreateScope()。", typ)
	}

	return nil, fmt.Errorf("di: 未知作用域 %v", def.Scope)
}

// CreateScope 为作用域实例创建一个新作用域。
func (c *container) CreateScope() Scope {
	return newScope(c)
}

func (c *container) serviceCount() int {
	return c.serviceCountVal
}
