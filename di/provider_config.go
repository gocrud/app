package di

import (
	"fmt"
	"reflect"
)

// ProviderType 提供者类型，定义如何提供实例
type ProviderType int

const (
	// ProviderTypeClass 类提供者，使用类或构造函数创建实例
	ProviderTypeClass ProviderType = iota
	// ProviderTypeValue 值提供者，直接使用静态值
	ProviderTypeValue
	// ProviderTypeFactory 工厂提供者，使用工厂函数创建实例
	ProviderTypeFactory
	// ProviderTypeExisting 别名提供者，引用已存在的类型
	ProviderTypeExisting
)

// ScopeType 作用域类型，定义实例的生命周期
type ScopeType int

const (
	// ScopeSingleton 单例作用域（默认）
	// 在整个容器生命周期内只创建一次实例，所有获取操作返回同一个实例
	// 适用场景：无状态服务、配置、日志记录器等
	ScopeSingleton ScopeType = iota

	// ScopeTransient 瞬态作用域
	// 每次获取都创建新实例，不缓存
	// 适用场景：命令对象、事件对象等需要独立状态的对象
	// 注意：性能开销较大，每次创建都会分配内存
	ScopeTransient

	// ScopeScoped 作用域内单例
	// 在同一个 Scope 内只创建一次实例，不同 Scope 之间实例相互独立
	// 适用场景：HTTP 请求级别的服务、数据库连接、工作单元等
	// 使用前必须调用 container.CreateScope() 创建作用域
	ScopeScoped
)

// ProviderOptions 提供者的通用配置选项
type ProviderOptions struct {
	// Optional 是否可选，默认为 false
	// 当设置为 true 时，如果找不到依赖不会报错，而是注入 nil 值
	Optional bool

	// Scope 作用域类型，默认为 ScopeSingleton
	// 决定实例的生命周期：Singleton（单例）、Transient（瞬态）、Scoped（作用域内单例）
	Scope ScopeType
}

// TypeProvider 类型提供者配置，用于将接口绑定到具体实现
//
// 示例：
//
//	// 绑定接口到实现（使用构造函数）
//	container.ProvideType(TypeProvider{
//		Provide: reflect.TypeOf((*UserService)(nil)).Elem(),
//		UseType: NewUserService,  // 构造函数
//	})
//
//	// 使用泛型语法糖
//	di.Bind[UserService](NewUserService)
type TypeProvider struct {
	// Provide 提供的类型，通常是接口类型
	// 可以是 reflect.Type 或 Token
	// 使用 reflect.Type 时必须是接口类型
	Provide any

	// UseType 使用的类型，可以是实例或构造函数
	// 如果是构造函数，参数将自动注入
	UseType any

	// Options 可选配置
	Options ProviderOptions
}

// ValueProvider 值提供者配置，用于注册静态值
//
// 示例：
//
//	container.ProvideValue(ValueProvider{
//		Provide: reflect.TypeOf((*Config)(nil)),
//		Value: &Config{Port: 8080},
//	})
type ValueProvider struct {
	// Provide 提供的类型或 Token
	Provide any

	// Value 静态值，将直接使用此值（不会创建新实例）
	Value any

	// Options 可选配置
	Options ProviderOptions
}

// FactoryProvider 工厂提供者配置，用于通过工厂函数创建实例
//
// 示例：
//
//	container.ProvideFactory(FactoryProvider{
//		Provide: reflect.TypeOf((*Database)(nil)),
//		Factory: func(config *Config) (*Database, error) {
//			return NewDatabase(config.DSN)
//		},
//	})
type FactoryProvider struct {
	// Provide 提供的类型或 Token
	Provide any

	// Factory 工厂函数，返回值的第一个参数是实例，可选的第二个参数是 error
	// 函数参数将自动注入（或使用 Deps 显式指定）
	Factory any

	// Deps 显式指定的依赖列表（可选）
	// 如果不指定，将根据工厂函数的参数类型自动推断
	Deps []any

	// Options 可选配置
	Options ProviderOptions
}

// ExistingProvider 别名提供者配置，用于创建类型别名
//
// 示例：
//
//	// 让 Logger 接口指向 DefaultLogger
//	container.ProvideExisting(ExistingProvider{
//		Provide: reflect.TypeOf((*Logger)(nil)).Elem(),
//		Existing: reflect.TypeOf((*DefaultLogger)(nil)),
//	})
type ExistingProvider struct {
	// Provide 提供的类型或 Token
	Provide any

	// Existing 引用的已存在类型
	// 当获取 Provide 类型时，实际返回 Existing 类型的实例
	Existing any

	// Options 可选配置
	Options ProviderOptions
}

// ProviderConfig 提供者配置（兼容旧版本，推荐使用具体的 Provider 类型）
// Deprecated: 建议使用 TypeProvider, ValueProvider, FactoryProvider, ExistingProvider
type ProviderConfig struct {
	// Provide 提供的类型或 Token
	Provide any

	// UseClass 使用指定的类（实例或构造函数）
	UseClass any

	// UseValue 使用静态值
	UseValue any

	// UseFactory 使用工厂函数
	UseFactory any

	// UseExisting 使用已存在的类型（别名）
	UseExisting any

	// Deps 工厂函数的依赖（可选，默认自动推断）
	Deps []any

	// Optional 是否可选（找不到依赖时不报错）
	Optional bool

	// Scope 作用域
	Scope ScopeType
}

// Validate 验证配置的有效性
func (pc *ProviderConfig) Validate() error {
	if pc.Provide == nil {
		return fmt.Errorf("Provide field is required")
	}

	// 确保只设置了一个 Use* 字段
	setCount := 0
	if pc.UseClass != nil {
		setCount++
	}
	if pc.UseValue != nil {
		setCount++
	}
	if pc.UseFactory != nil {
		setCount++
	}
	if pc.UseExisting != nil {
		setCount++
	}

	if setCount == 0 {
		return fmt.Errorf("must set one of: UseClass, UseValue, UseFactory, UseExisting")
	}
	if setCount > 1 {
		return fmt.Errorf("can only set one of: UseClass, UseValue, UseFactory, UseExisting")
	}

	return nil
}

// GetProviderType 获取提供者类型
func (pc *ProviderConfig) GetProviderType() ProviderType {
	if pc.UseClass != nil {
		return ProviderTypeClass
	}
	if pc.UseValue != nil {
		return ProviderTypeValue
	}
	if pc.UseFactory != nil {
		return ProviderTypeFactory
	}
	if pc.UseExisting != nil {
		return ProviderTypeExisting
	}
	return ProviderTypeClass
}

// toProviderConfig 转换为通用配置（内部使用）
func (tp *TypeProvider) toProviderConfig() *ProviderConfig {
	// 如果 Provide 是reflect.Type，检查是否为接口类型
	if typ, ok := tp.Provide.(reflect.Type); ok {
		if typ.Kind() != reflect.Interface {
			panic(fmt.Sprintf("TypeProvider.Provide requires an interface type, got %v", typ))
		}
	}

	return &ProviderConfig{
		Provide:  tp.Provide,
		UseClass: tp.UseType,
		Optional: tp.Options.Optional,
		Scope:    tp.Options.Scope,
	}
}

func (vp *ValueProvider) toProviderConfig() *ProviderConfig {
	return &ProviderConfig{
		Provide:  vp.Provide,
		UseValue: vp.Value,
		Optional: vp.Options.Optional,
		Scope:    vp.Options.Scope,
	}
}

func (fp *FactoryProvider) toProviderConfig() *ProviderConfig {
	return &ProviderConfig{
		Provide:    fp.Provide,
		UseFactory: fp.Factory,
		Deps:       fp.Deps,
		Optional:   fp.Options.Optional,
		Scope:      fp.Options.Scope,
	}
}

func (ep *ExistingProvider) toProviderConfig() *ProviderConfig {
	// 如果 Provide 是 reflect.Type，检查是否为接口类型
	if typ, ok := ep.Provide.(reflect.Type); ok {
		if typ.Kind() != reflect.Interface {
			panic(fmt.Sprintf("ExistingProvider.Provide requires an interface type, got %v", typ))
		}
	}

	return &ProviderConfig{
		Provide:     ep.Provide,
		UseExisting: ep.Existing,
		Optional:    ep.Options.Optional,
		Scope:       ep.Options.Scope,
	}
}

// resolveProvideKey 解析 Provide 字段为 typeKey
func (pc *ProviderConfig) resolveProvideKey() (typeKey, error) {
	switch v := pc.Provide.(type) {
	case reflect.Type:
		return typeKey{typ: v}, nil
	case *Token[string]:
		return typeKey{typ: v.Type(), token: v}, nil
	case *Token[int]:
		return typeKey{typ: v.Type(), token: v}, nil
	case *Token[bool]:
		return typeKey{typ: v.Type(), token: v}, nil
	// 可以继续添加其他 Token 类型...
	default:
		// 尝试通过反射获取类型
		typ := reflect.TypeOf(v)
		if typ == nil {
			return typeKey{}, fmt.Errorf("cannot determine type from Provide field")
		}
		// 处理指针和接口类型
		if typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Interface {
			return typeKey{typ: typ.Elem()}, nil
		}
		return typeKey{typ: typ}, nil
	}
}

// resolveDependency 解析单个依赖为 typeKey
func resolveDependency(dep any) (typeKey, error) {
	switch v := dep.(type) {
	case reflect.Type:
		return typeKey{typ: v}, nil
	case tokenInterface:
		return typeKey{typ: v.Type(), token: v}, nil
	default:
		typ := reflect.TypeOf(v)
		if typ == nil {
			return typeKey{}, fmt.Errorf("cannot determine type from dependency")
		}
		if typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Interface {
			return typeKey{typ: typ.Elem()}, nil
		}
		return typeKey{typ: typ}, nil
	}
}

// tokenInterface Token 的通用接口（用于类型判断）
type tokenInterface interface {
	Name() string
	Type() reflect.Type
	String() string
}
