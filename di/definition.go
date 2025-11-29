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

// ServiceKey 是服务映射的唯一键。
type ServiceKey struct {
	Type reflect.Type
	Name string
}

// FieldInjection 包含需要注入的结构体字段的元数据。
type FieldInjection struct {
	Index       int
	Name        string // 字段名
	Type        reflect.Type
	Optional    bool
	ServiceName string // 注入的服务名称
}

// InjectionSchema 包含预计算的注入元数据。
type InjectionSchema struct {
	Fields []FieldInjection // 用于结构体注入
	Args   []reflect.Type   // 用于函数/工厂注入
}

// ServiceDefinition 包含注册服务的元数据。
type ServiceDefinition struct {
	ID           int
	Type         reflect.Type
	Name         string // 服务名称
	Scope        ScopeType
	ImplType     reflect.Type // 用于结构体反射
	Impl         any          // 工厂函数或结构体指针
	IsFactory    bool
	IsValue      bool
	InjectFields bool // 是否对 IsValue 的实例执行字段注入

	Schema *InjectionSchema // 预计算的依赖图

	// 用于单例作用域
	singletonInst any
	singletonErr  error
	singletonOnce sync.Once
}
