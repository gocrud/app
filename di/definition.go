package di

import (
	"reflect"
	"sync"
)

// ScopeType 定义了服务的生命周期。
type ScopeType int

const (
	// ScopeSingleton 每个容器创建一个实例。
	ScopeSingleton ScopeType = iota
	// ScopeTransient 每次请求创建一个新实例。
	ScopeTransient
	// ScopeScoped 每个作用域创建一个实例。
	ScopeScoped
)

// FieldInjection 包含需要注入的结构体字段的元数据。
type FieldInjection struct {
	Index    int
	Name     string
	Type     reflect.Type
	Optional bool
}

// InjectionSchema 包含预计算的注入元数据。
type InjectionSchema struct {
	Fields []FieldInjection // 用于结构体注入
	Args   []reflect.Type   // 用于函数/工厂注入
}

// ServiceDefinition 包含注册服务的元数据。
type ServiceDefinition struct {
	ID        int          // 唯一整数 ID，用于 O(1) 访问
	Type      reflect.Type // 注册的类型（接口或结构体）
	Impl      any          // 实现：值、构造函数或 nil（自动）
	ImplType  reflect.Type // 实现的具体类型
	IsFactory bool         // Impl 是否为工厂函数
	IsValue   bool         // Impl 是否为静态值
	Scope     ScopeType    // 生命周期作用域
	Tags      []string     // 用于分类的可选标签

	// Schema 包含解析器使用的预计算注入元数据。
	Schema *InjectionSchema

	// 单例的运行时状态
	singletonOnce sync.Once
	singletonInst any
	singletonErr  error
}
