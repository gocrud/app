package di

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// Container DI容器，管理所有依赖的注册、构建和获取
//
// 容器提供以下核心功能：
//   - 依赖注册：支持多种注册方式（类型绑定、值、工厂、别名）
//   - 依赖解析：自动解析构造函数参数和字段注入
//   - 生命周期管理：支持 Singleton、Transient、Scoped 三种作用域
//   - 并发安全：所有操作都是线程安全的
//   - 循环依赖检测：在 Build 阶段检测并报错
//
// 使用流程：
//  1. 创建容器：container := di.NewContainer()
//  2. 注册服务：container.Provide(...)
//  3. 构建容器：container.Build()
//  4. 获取实例：instance, _ := container.Get(...)
type Container struct {
	mu           sync.RWMutex              // 读写锁，保护容器状态
	providers    map[typeKey]*providerInfo // 类型到提供者的映射
	instances    map[typeKey]any           // 已构建的单例实例缓存
	built        atomic.Bool               // 是否已构建标志
	buildMu      sync.Mutex                // 构建锁，防止并发构建
	resolveData  map[typeKey]*resolveInfo  // 预解析的依赖数据（Build时生成）
	currentScope *Scope                    // 当前作用域（用于 Scoped 实例）
}

// Scope 作用域，用于管理 Scoped 生命周期的实例
//
// 作用域在以下场景中使用：
//   - HTTP 请求处理：每个请求创建独立的作用域
//   - 工作单元模式：数据库事务范围内的依赖共享
//   - 任务处理：每个任务有独立的依赖实例
//
// 使用示例：
//
//	scope := container.CreateScope()
//	defer scope.Dispose()  // 确保释放资源
//	instance, _ := scope.GetByType(someType)
type Scope struct {
	parent    *Container      // 父容器引用
	instances map[typeKey]any // 作用域内的实例缓存
	mu        sync.RWMutex    // 作用域锁
	disposed  atomic.Bool     // 是否已释放标志
}

// typeKey 类型键，用于在容器中唯一标识一个依赖
type typeKey struct {
	typ   reflect.Type   // 类型信息
	token tokenInterface // 可选的 Token（用于区分同类型的不同依赖）
}

// providerInfo 提供者信息，存储如何创建实例的所有信息
type providerInfo struct {
	value        any            // 值、构造函数或工厂函数
	providerType ProviderType   // 提供者类型
	isFunc       bool           // 是否是函数（构造函数或工厂）
	funcType     reflect.Type   // 函数类型
	returnType   reflect.Type   // 返回类型
	paramTypes   []reflect.Type // 参数类型列表
	deps         []typeKey      // 显式指定的依赖（用于工厂函数）
	existingKey  typeKey        // UseExisting 指向的类型
	optional     bool           // 是否可选
	scope        ScopeType      // 作用域
}

// resolveInfo 预解析的依赖信息（在Build时生成，提高运行时性能）
type resolveInfo struct {
	instance      any               // 已实例化的对象（仅 Singleton）
	fieldInjects  []fieldInjectInfo // 字段注入信息
	isConstructed bool              // 是否已构造
}

// fieldInjectInfo 字段注入信息
type fieldInjectInfo struct {
	fieldIndex int          // 字段索引
	targetKey  typeKey      // 目标类型键
	fieldType  reflect.Type // 字段类型
}

// NewContainer 创建新的 DI 容器
//
// 返回一个空的容器实例，需要通过 Provide 系列方法注册依赖，
// 然后调用 Build() 完成构建后才能使用 Get 获取实例。
func NewContainer() *Container {
	return &Container{
		providers:   make(map[typeKey]*providerInfo),
		instances:   make(map[typeKey]any),
		resolveData: make(map[typeKey]*resolveInfo),
	}
}

// register 注册提供者（保留向后兼容）
func (c *Container) register(value any) error {
	// 如果传入的是构造函数，自动使用其首个返回值类型作为提供类型
	if value != nil {
		val := reflect.ValueOf(value)
		if val.Kind() == reflect.Func {
			funcType := val.Type()
			if funcType.NumOut() == 0 {
				return fmt.Errorf("constructor function must return at least one value")
			}

			return c.registerWithConfig(ProviderConfig{
				Provide:  funcType.Out(0),
				UseClass: value,
			})
		}
	}

	// 默认行为
	return c.registerWithConfig(ProviderConfig{
		Provide:  value,
		UseClass: value,
	})
}

// registerWithConfig 使用 ProviderConfig 注册提供者
func (c *Container) registerWithConfig(config ProviderConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.built.Load() {
		return fmt.Errorf("cannot register after Build() is called")
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid provider config: %w", err)
	}

	// 解析提供的类型键
	tk, err := config.resolveProvideKey()
	if err != nil {
		return fmt.Errorf("failed to resolve provide key: %w", err)
	}

	// 检查是否已注册
	if _, exists := c.providers[tk]; exists {
		return fmt.Errorf("type %v is already registered", tk.typ)
	}

	info := &providerInfo{
		providerType: config.GetProviderType(),
		optional:     config.Optional,
		scope:        config.Scope,
	}

	switch config.GetProviderType() {
	case ProviderTypeValue:
		// 静态值
		info.value = config.UseValue
		info.returnType = reflect.TypeOf(config.UseValue)

	case ProviderTypeClass:
		// 类或构造函数
		val := reflect.ValueOf(config.UseClass)
		typ := val.Type()

		if typ.Kind() == reflect.Func {
			// 构造函数
			if typ.NumOut() == 0 {
				return fmt.Errorf("constructor function must return at least one value")
			}
			info.isFunc = true
			info.funcType = typ
			info.returnType = typ.Out(0)
			info.value = config.UseClass

			// 记录参数类型
			info.paramTypes = make([]reflect.Type, typ.NumIn())
			for i := 0; i < typ.NumIn(); i++ {
				info.paramTypes[i] = typ.In(i)
			}
		} else {
			// 直接使用实例
			info.value = config.UseClass
			info.returnType = typ
		}

	case ProviderTypeFactory:
		// 工厂函数
		val := reflect.ValueOf(config.UseFactory)
		typ := val.Type()

		if typ.Kind() != reflect.Func {
			return fmt.Errorf("UseFactory must be a function")
		}
		if typ.NumOut() == 0 {
			return fmt.Errorf("factory function must return at least one value")
		}

		info.isFunc = true
		info.funcType = typ
		info.returnType = typ.Out(0)
		info.value = config.UseFactory

		// 处理依赖
		if len(config.Deps) > 0 {
			// 使用显式指定的依赖
			info.deps = make([]typeKey, len(config.Deps))
			for i, dep := range config.Deps {
				depKey, err := resolveDependency(dep)
				if err != nil {
					return fmt.Errorf("failed to resolve dependency at index %d: %w", i, err)
				}
				info.deps[i] = depKey
			}
		} else {
			// 自动推断依赖（从函数参数）
			info.paramTypes = make([]reflect.Type, typ.NumIn())
			for i := 0; i < typ.NumIn(); i++ {
				info.paramTypes[i] = typ.In(i)
			}
		}

	case ProviderTypeExisting:
		// 别名
		existingKey, err := resolveDependency(config.UseExisting)
		if err != nil {
			return fmt.Errorf("failed to resolve UseExisting: %w", err)
		}
		info.existingKey = existingKey
		info.returnType = existingKey.typ
	}

	c.providers[tk] = info
	return nil
}

// validateScopeDependencies 验证作用域依赖的合法性
// 规则：Singleton 不能依赖 Transient 或 Scoped（会导致单例持有短生命周期对象）
func (c *Container) validateScopeDependencies() error {
	for tk, info := range c.providers {
		// 只检查 Singleton 的依赖
		if info.scope != ScopeSingleton {
			continue
		}

		// 检查构造函数/工厂函数的参数依赖
		if info.isFunc {
			for i, paramType := range info.paramTypes {
				paramTk := typeKey{typ: paramType}
				paramInfo, exists := c.providers[paramTk]
				if !exists {
					continue // 在 buildInstance 中会处理缺失的依赖
				}

				// Singleton 不能依赖 Transient 或 Scoped
				if paramInfo.scope == ScopeTransient {
					return fmt.Errorf(
						"DI: Singleton type %v cannot depend on Transient type %v at parameter index %d. "+
							"This would cause the singleton to hold a reference to a transient instance, "+
							"violating the transient lifecycle contract",
						tk.typ, paramType, i)
				}
				if paramInfo.scope == ScopeScoped {
					return fmt.Errorf(
						"DI: Singleton type %v cannot depend on Scoped type %v at parameter index %d. "+
							"This would cause the singleton to hold a reference to a scoped instance, "+
							"violating the scoped lifecycle contract",
						tk.typ, paramType, i)
				}
			}
		}

		// 检查工厂函数的显式依赖
		if info.providerType == ProviderTypeFactory && len(info.deps) > 0 {
			for i, depKey := range info.deps {
				depInfo, exists := c.providers[depKey]
				if !exists {
					continue
				}

				if depInfo.scope == ScopeTransient {
					return fmt.Errorf(
						"DI: Singleton type %v cannot depend on Transient type %v at dependency index %d",
						tk.typ, depKey.typ, i)
				}
				if depInfo.scope == ScopeScoped {
					return fmt.Errorf(
						"DI: Singleton type %v cannot depend on Scoped type %v at dependency index %d",
						tk.typ, depKey.typ, i)
				}
			}
		}

		// 检查字段注入依赖（针对非函数类型）
		if !info.isFunc && info.providerType == ProviderTypeClass {
			val := reflect.ValueOf(info.value)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			if val.Kind() == reflect.Struct {
				typ := val.Type()
				for i := 0; i < typ.NumField(); i++ {
					field := typ.Field(i)
					tag := field.Tag.Get("di")

					// 跳过没有di tag的字段
					if tag == "-" {
						continue
					}
					if _, hasTag := field.Tag.Lookup("di"); !hasTag {
						continue
					}

					fieldTk := typeKey{typ: field.Type}
					fieldInfo, exists := c.providers[fieldTk]
					if !exists {
						continue
					}

					if fieldInfo.scope == ScopeTransient {
						return fmt.Errorf(
							"DI: Singleton type %v cannot have field %s injected with Transient type %v",
							tk.typ, field.Name, field.Type)
					}
					if fieldInfo.scope == ScopeScoped {
						return fmt.Errorf(
							"DI: Singleton type %v cannot have field %s injected with Scoped type %v",
							tk.typ, field.Name, field.Type)
					}
				}
			}
		}
	}

	return nil
}

// Build 构建容器，预解析所有依赖并创建单例实例
//
// 此方法必须在所有 Provide 调用之后、Get 调用之前执行。
// Build 过程包括：
//  1. 验证作用域依赖规则（Singleton 不能依赖 Transient/Scoped）
//  2. 检测循环依赖
//  3. 拓扑排序，确定最优构建顺序
//  4. 创建所有 Singleton 实例
//  5. 预解析字段注入信息
//
// 如果构建失败，返回详细的错误信息。
// 构建成功后，容器状态变为已构建，不能再注册新的依赖。
//
// 线程安全：可以从多个 goroutine 调用，但只会构建一次。
func (c *Container) Build() error {
	c.buildMu.Lock()
	defer c.buildMu.Unlock()

	if c.built.Load() {
		return fmt.Errorf("Build() already called")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 验证作用域依赖关系
	if err := c.validateScopeDependencies(); err != nil {
		return err
	}

	// 使用拓扑排序确定构建顺序
	buildOrder, err := c.topologicalSort()
	if err != nil {
		return err
	}

	// 共享 building map 减少内存分配
	building := make(map[typeKey]bool, len(c.providers))

	// 按拓扑顺序构建所有实例
	for _, tk := range buildOrder {
		info := c.providers[tk]

		// 根据作用域决定是否预创建实例
		switch info.scope {
		case ScopeSingleton:
			// 单例：预创建并缓存
			if _, err := c.buildInstance(tk, info, building); err != nil {
				return fmt.Errorf("DI: failed to build %v: %w", tk.typ, err)
			}
		case ScopeTransient, ScopeScoped:
			// 瞬态和作用域：只验证依赖关系，不预创建
			if err := c.validateDependencies(tk, info, building); err != nil {
				return fmt.Errorf("DI: failed to validate dependencies for %v: %w", tk.typ, err)
			}
		}
	}

	c.built.Store(true)
	return nil
}

// validateDependencies 验证依赖关系（不创建实例）
// 递归检查所有依赖是否可解析，并检测循环依赖
func (c *Container) validateDependencies(tk typeKey, info *providerInfo, building map[typeKey]bool) error {
	// 检查循环依赖
	if building[tk] {
		return fmt.Errorf("circular dependency detected for type %v", tk.typ)
	}
	building[tk] = true
	defer delete(building, tk)

	// 验证构造函数/工厂函数的参数
	if info.isFunc {
		for i, paramType := range info.paramTypes {
			paramTk := typeKey{typ: paramType}
			paramInfo, exists := c.providers[paramTk]
			if !exists {
				if info.optional {
					continue
				}
				return fmt.Errorf("no provider found for parameter type %v at index %d", paramType, i)
			}

			// 递归验证依赖
			if paramInfo.scope == ScopeSingleton {
				// Singleton 依赖：需要确保已构建
				if _, exists := c.resolveData[paramTk]; !exists {
					if _, err := c.buildInstance(paramTk, paramInfo, building); err != nil {
						return err
					}
				}
			} else {
				// Transient/Scoped 依赖：只验证
				if err := c.validateDependencies(paramTk, paramInfo, building); err != nil {
					return err
				}
			}
		}
	}

	// 验证工厂函数的显式依赖
	if info.providerType == ProviderTypeFactory && len(info.deps) > 0 {
		for _, depKey := range info.deps {
			depInfo, exists := c.providers[depKey]
			if !exists {
				return fmt.Errorf("dependency not found: %v", depKey.typ)
			}

			if depInfo.scope == ScopeSingleton {
				if _, exists := c.resolveData[depKey]; !exists {
					if _, err := c.buildInstance(depKey, depInfo, building); err != nil {
						return err
					}
				}
			} else {
				if err := c.validateDependencies(depKey, depInfo, building); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// buildInstance 构建单个实例（递归解析依赖）
func (c *Container) buildInstance(tk typeKey, info *providerInfo, building map[typeKey]bool) (any, error) {
	// 检查是否已构建
	if resolveInfo, exists := c.resolveData[tk]; exists && resolveInfo.isConstructed {
		return resolveInfo.instance, nil
	}

	// 检查循环依赖
	if building[tk] {
		return nil, fmt.Errorf("circular dependency detected for type %v", tk.typ)
	}
	building[tk] = true
	defer delete(building, tk)

	var instance any
	var err error

	switch info.providerType {
	case ProviderTypeValue:
		// 直接使用值
		instance = info.value

	case ProviderTypeExisting:
		// 别名：解析指向的类型
		targetInfo, exists := c.providers[info.existingKey]
		if !exists {
			return nil, fmt.Errorf("UseExisting target not found: %v", info.existingKey.typ)
		}
		instance, err = c.buildInstance(info.existingKey, targetInfo, building)
		if err != nil {
			return nil, err
		}

	case ProviderTypeFactory:
		// 工厂函数
		instance, err = c.invokeFactory(info, building)
		if err != nil {
			return nil, err
		}

	case ProviderTypeClass:
		// 类或构造函数
		if info.isFunc {
			instance, err = c.invokeConstructor(info, building)
			if err != nil {
				return nil, err
			}
		} else {
			instance = info.value
		}
	}

	// 解析字段注入
	fieldInjects, err := c.resolveFieldInjections(instance, building)
	if err != nil {
		return nil, err
	}

	// 执行字段注入
	if err := c.performFieldInjections(instance, fieldInjects); err != nil {
		return nil, err
	}

	// 保存解析信息
	c.resolveData[tk] = &resolveInfo{
		instance:      instance,
		fieldInjects:  fieldInjects,
		isConstructed: true,
	}
	c.instances[tk] = instance

	return instance, nil
}

// createTransientInstance 创建瞬态实例（不缓存）
func (c *Container) createTransientInstance(tk typeKey, info *providerInfo) (any, error) {
	var instance any
	var err error

	switch info.providerType {
	case ProviderTypeValue:
		// 直接使用值
		instance = info.value

	case ProviderTypeExisting:
		// 别名：获取指向的类型
		targetInfo, exists := c.providers[info.existingKey]
		if !exists {
			return nil, fmt.Errorf("UseExisting target not found: %v", info.existingKey.typ)
		}
		return c.getInstanceByScope(info.existingKey, targetInfo)

	case ProviderTypeFactory:
		// 工厂函数
		instance, err = c.invokeFactoryTransient(info)
		if err != nil {
			return nil, err
		}

	case ProviderTypeClass:
		// 类或构造函数
		if info.isFunc {
			instance, err = c.invokeConstructorTransient(info)
			if err != nil {
				return nil, err
			}
		} else {
			instance = info.value
		}
	}

	// 解析字段注入
	fieldInjects, err := c.resolveFieldInjectionsTransient(instance)
	if err != nil {
		return nil, err
	}

	// 执行字段注入
	if err := c.performFieldInjections(instance, fieldInjects); err != nil {
		return nil, err
	}

	return instance, nil
}

// topologicalSort 拓扑排序，返回最优构建顺序
// 无依赖的类型排在前面，减少递归深度
func (c *Container) topologicalSort() ([]typeKey, error) {
	// 计算每个类型的依赖数量
	dependencyCount := make(map[typeKey]int)
	dependents := make(map[typeKey][]typeKey) // 记录依赖关系

	for tk, info := range c.providers {
		count := 0

		// 统计构造函数参数依赖
		if info.isFunc {
			for _, paramType := range info.paramTypes {
				paramTk := typeKey{typ: paramType}
				if _, exists := c.providers[paramTk]; exists {
					count++
					dependents[paramTk] = append(dependents[paramTk], tk)
				}
			}
		}

		// 统计字段注入依赖
		if !info.isFunc {
			val := reflect.ValueOf(info.value)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			if val.Kind() == reflect.Struct {
				typ := val.Type()
				for i := 0; i < typ.NumField(); i++ {
					field := typ.Field(i)
					tag := field.Tag.Get("di")
					if tag == "-" {
						continue
					}
					if _, hasTag := field.Tag.Lookup("di"); hasTag {
						targetTk := typeKey{typ: field.Type}
						if _, exists := c.providers[targetTk]; exists {
							count++
							dependents[targetTk] = append(dependents[targetTk], tk)
						}
					}
				}
			}
		}

		dependencyCount[tk] = count
	}

	// Kahn 算法进行拓扑排序
	var result []typeKey
	queue := make([]typeKey, 0, len(c.providers))

	// 将无依赖的节点加入队列
	for tk, count := range dependencyCount {
		if count == 0 {
			queue = append(queue, tk)
		}
	}

	// 处理队列
	for len(queue) > 0 {
		// 取出队首
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// 更新依赖此节点的其他节点
		for _, dependent := range dependents[current] {
			dependencyCount[dependent]--
			if dependencyCount[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// 检查是否所有节点都被处理（检测循环依赖）
	if len(result) != len(c.providers) {
		// 找出未处理的节点（参与循环依赖的节点）
		var remaining []typeKey
		for tk := range dependencyCount {
			found := false
			for _, r := range result {
				if r == tk {
					found = true
					break
				}
			}
			if !found {
				remaining = append(remaining, tk)
			}
		}
		return nil, fmt.Errorf("circular dependency detected among types: %v", remaining)
	}

	return result, nil
}

// invokeConstructor 调用构造函数
func (c *Container) invokeConstructor(info *providerInfo, building map[typeKey]bool) (any, error) {
	fn := reflect.ValueOf(info.value)

	// 解析所有参数
	args := make([]reflect.Value, len(info.paramTypes))
	for i, paramType := range info.paramTypes {
		paramTk := typeKey{typ: paramType}

		// 查找提供者
		paramInfo, exists := c.providers[paramTk]
		if !exists {
			return nil, fmt.Errorf("no provider found for parameter type %v at index %d", paramType, i)
		}

		// 递归构建参数
		paramInstance, err := c.buildInstance(paramTk, paramInfo, building)
		if err != nil {
			return nil, err
		}

		args[i] = reflect.ValueOf(paramInstance)
	}

	// 调用构造函数
	results := fn.Call(args)
	if len(results) == 0 {
		return nil, fmt.Errorf("constructor returned no values")
	}

	// 检查最后一个返回值是否是error
	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				// 构造函数返回了错误
				return nil, fmt.Errorf("constructor failed: %w", lastResult.Interface().(error))
			}
		}
	}

	// 检查第一个返回值是否为nil（避免缓存nil实例）
	firstResult := results[0]
	if firstResult.Kind() == reflect.Ptr || firstResult.Kind() == reflect.Interface {
		if firstResult.IsNil() {
			return nil, fmt.Errorf("constructor returned nil instance")
		}
	}

	return firstResult.Interface(), nil
}

// invokeConstructorTransient 调用构造函数（用于瞬态实例）
func (c *Container) invokeConstructorTransient(info *providerInfo) (any, error) {
	fn := reflect.ValueOf(info.value)

	// 解析所有参数
	args := make([]reflect.Value, len(info.paramTypes))
	for i, paramType := range info.paramTypes {
		paramTk := typeKey{typ: paramType}

		// 查找提供者
		paramInfo, exists := c.providers[paramTk]
		if !exists {
			return nil, fmt.Errorf("no provider found for parameter type %v at index %d", paramType, i)
		}

		// 根据依赖的作用域获取实例
		paramInstance, err := c.getInstanceByScope(paramTk, paramInfo)
		if err != nil {
			return nil, err
		}

		args[i] = reflect.ValueOf(paramInstance)
	}

	// 调用构造函数
	results := fn.Call(args)
	if len(results) == 0 {
		return nil, fmt.Errorf("constructor returned no values")
	}

	// 检查最后一个返回值是否是error
	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				return nil, fmt.Errorf("constructor failed: %w", lastResult.Interface().(error))
			}
		}
	}

	// 检查第一个返回值是否为nil
	firstResult := results[0]
	if firstResult.Kind() == reflect.Ptr || firstResult.Kind() == reflect.Interface {
		if firstResult.IsNil() {
			return nil, fmt.Errorf("constructor returned nil instance")
		}
	}

	return firstResult.Interface(), nil
}

// invokeFactory 调用工厂函数
func (c *Container) invokeFactory(info *providerInfo, building map[typeKey]bool) (any, error) {
	fn := reflect.ValueOf(info.value)

	var args []reflect.Value

	if len(info.deps) > 0 {
		// 使用显式指定的依赖
		args = make([]reflect.Value, len(info.deps))
		for i, depKey := range info.deps {
			depInfo, exists := c.providers[depKey]
			if !exists {
				return nil, fmt.Errorf("dependency not found: %v", depKey.typ)
			}

			depInstance, err := c.buildInstance(depKey, depInfo, building)
			if err != nil {
				return nil, fmt.Errorf("failed to build dependency %v: %w", depKey.typ, err)
			}

			args[i] = reflect.ValueOf(depInstance)
		}
	} else {
		// 自动推断依赖（从函数参数类型）
		args = make([]reflect.Value, len(info.paramTypes))
		for i, paramType := range info.paramTypes {
			paramTk := typeKey{typ: paramType}

			paramInfo, exists := c.providers[paramTk]
			if !exists {
				return nil, fmt.Errorf("no provider found for factory parameter type %v at index %d", paramType, i)
			}

			paramInstance, err := c.buildInstance(paramTk, paramInfo, building)
			if err != nil {
				return nil, err
			}

			args[i] = reflect.ValueOf(paramInstance)
		}
	}

	// 调用工厂函数
	results := fn.Call(args)
	if len(results) == 0 {
		return nil, fmt.Errorf("factory returned no values")
	}

	// 检查最后一个返回值是否为error
	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				return nil, fmt.Errorf("factory failed: %w", lastResult.Interface().(error))
			}
		}
	}

	// 检查第一个返回值是否为nil
	firstResult := results[0]
	if firstResult.Kind() == reflect.Ptr || firstResult.Kind() == reflect.Interface {
		if firstResult.IsNil() {
			return nil, fmt.Errorf("factory returned nil instance")
		}
	}

	return firstResult.Interface(), nil
}

// invokeFactoryTransient 调用工厂函数（用于瞬态实例）
func (c *Container) invokeFactoryTransient(info *providerInfo) (any, error) {
	fn := reflect.ValueOf(info.value)

	var args []reflect.Value

	if len(info.deps) > 0 {
		// 使用显式指定的依赖
		args = make([]reflect.Value, len(info.deps))
		for i, depKey := range info.deps {
			depInfo, exists := c.providers[depKey]
			if !exists {
				return nil, fmt.Errorf("dependency not found: %v", depKey.typ)
			}

			depInstance, err := c.getInstanceByScope(depKey, depInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to get dependency %v: %w", depKey.typ, err)
			}

			args[i] = reflect.ValueOf(depInstance)
		}
	} else {
		// 自动推断依赖（从函数参数类型）
		args = make([]reflect.Value, len(info.paramTypes))
		for i, paramType := range info.paramTypes {
			paramTk := typeKey{typ: paramType}

			paramInfo, exists := c.providers[paramTk]
			if !exists {
				return nil, fmt.Errorf("no provider found for factory parameter type %v at index %d", paramType, i)
			}

			paramInstance, err := c.getInstanceByScope(paramTk, paramInfo)
			if err != nil {
				return nil, err
			}

			args[i] = reflect.ValueOf(paramInstance)
		}
	}

	// 调用工厂函数
	results := fn.Call(args)
	if len(results) == 0 {
		return nil, fmt.Errorf("factory returned no values")
	}

	// 检查最后一个返回值是否为error
	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				return nil, fmt.Errorf("factory failed: %w", lastResult.Interface().(error))
			}
		}
	}

	// 检查第一个返回值是否为nil
	firstResult := results[0]
	if firstResult.Kind() == reflect.Ptr || firstResult.Kind() == reflect.Interface {
		if firstResult.IsNil() {
			return nil, fmt.Errorf("factory returned nil instance")
		}
	}

	return firstResult.Interface(), nil
}

// resolveFieldInjections 解析字段注入信息
func (c *Container) resolveFieldInjections(instance any, building map[typeKey]bool) ([]fieldInjectInfo, error) {
	val := reflect.ValueOf(instance)
	// 检查nil指针
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, fmt.Errorf("cannot resolve field injections for nil pointer instance")
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, nil
	}

	typ := val.Type()
	var fieldInjects []fieldInjectInfo

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("di")

		// 跳过没有tag或显式标记为-的字段
		if tag == "-" {
			continue
		}

		// 只处理有di tag的字段
		if _, hasTag := field.Tag.Lookup("di"); !hasTag {
			continue
		}

		// 检查是否为可选字段
		optional := tag == "?" || tag == "optional"

		// 解析依赖
		targetTk := typeKey{typ: field.Type}

		// 查找提供者
		targetInfo, exists := c.providers[targetTk]
		if !exists {
			if optional {
				// 可选字段，跳过
				continue
			}
			return nil, fmt.Errorf("no provider found for field %s (type %v)", field.Name, field.Type)
		}

		// 递归构建依赖
		if _, err := c.buildInstance(targetTk, targetInfo, building); err != nil {
			if optional {
				// 可选字段，跳过
				continue
			}
			return nil, fmt.Errorf("failed to build dependency for field %s: %w", field.Name, err)
		}

		fieldInjects = append(fieldInjects, fieldInjectInfo{
			fieldIndex: i,
			targetKey:  targetTk,
			fieldType:  field.Type,
		})
	}

	return fieldInjects, nil
}

// resolveFieldInjectionsTransient 解析字段注入信息（用于瞬态实例）
func (c *Container) resolveFieldInjectionsTransient(instance any) ([]fieldInjectInfo, error) {
	val := reflect.ValueOf(instance)
	// 检查nil指针
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, fmt.Errorf("cannot resolve field injections for nil pointer instance")
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, nil
	}

	typ := val.Type()
	var fieldInjects []fieldInjectInfo

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("di")

		// 跳过没有tag或显式标记为-的字段
		if tag == "-" {
			continue
		}

		// 只处理有di tag的字段
		if _, hasTag := field.Tag.Lookup("di"); !hasTag {
			continue
		}

		// 检查是否为可选字段
		optional := tag == "?" || tag == "optional"

		// 解析依赖
		targetTk := typeKey{typ: field.Type}

		// 查找提供者
		_, exists := c.providers[targetTk]
		if !exists {
			if optional {
				continue
			}
			return nil, fmt.Errorf("no provider found for field %s (type %v)", field.Name, field.Type)
		}

		// 不需要预先获取实例，在 performFieldInjections 中动态获取
		fieldInjects = append(fieldInjects, fieldInjectInfo{
			fieldIndex: i,
			targetKey:  targetTk,
			fieldType:  field.Type,
		})
	}

	return fieldInjects, nil
}

// performFieldInjections 执行字段注入
func (c *Container) performFieldInjections(instance any, fieldInjects []fieldInjectInfo) error {
	if len(fieldInjects) == 0 {
		return nil
	}

	val := reflect.ValueOf(instance)
	// 检查nil指针
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return fmt.Errorf("cannot perform field injection on nil pointer instance")
		}
		val = val.Elem()
	}

	instanceType := val.Type()

	for _, inject := range fieldInjects {
		fieldVal := val.Field(inject.fieldIndex)
		field := instanceType.Field(inject.fieldIndex)

		if !fieldVal.CanSet() {
			return fmt.Errorf("DI: cannot inject field '%s.%s' (type: %v) - field must be exported (start with uppercase letter)",
				instanceType.Name(), field.Name, inject.fieldType)
		}

		// 动态获取依赖实例（支持不同作用域）
		var targetInstance any
		var err error

		// 查找 provider 信息（已在 Build 中持有锁）
		info, exists := c.providers[inject.targetKey]
		if !exists {
			return fmt.Errorf("DI: %s.%s needs %v, but it's not registered in the container",
				instanceType.Name(), field.Name, inject.targetKey.typ)
		}

		// 使用内部方法，避免重复获取锁
		targetInstance, err = c.getInstanceByScopeInternal(inject.targetKey, info)
		if err != nil {
			return fmt.Errorf("DI: failed to get instance for %s.%s: %w",
				instanceType.Name(), field.Name, err)
		}

		fieldVal.Set(reflect.ValueOf(targetInstance))
	}

	return nil
}

// Get 获取实例（Build后调用，根据作用域返回实例）
func (c *Container) Get(tk typeKey) (any, error) {
	if !c.built.Load() {
		return nil, fmt.Errorf("must call Build() before Get()")
	}

	c.mu.RLock()
	info, exists := c.providers[tk]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no provider found for type %v", tk.typ)
	}

	return c.getInstanceByScope(tk, info)
}

// getInstanceByScope 根据作用域获取或创建实例
func (c *Container) getInstanceByScope(tk typeKey, info *providerInfo) (any, error) {
	// 如果 info 为 nil，则从 providers 中查找
	if info == nil {
		c.mu.RLock()
		var exists bool
		info, exists = c.providers[tk]
		c.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("no provider found for type %v", tk.typ)
		}
	}

	return c.getInstanceByScopeInternal(tk, info)
}

// getInstanceByScopeInternal 内部方法，不获取 providers 锁（假设调用者已持有锁或不需要锁）
func (c *Container) getInstanceByScopeInternal(tk typeKey, info *providerInfo) (any, error) {
	switch info.scope {
	case ScopeSingleton:
		// 单例：从缓存读取
		// 注意：这里可能在 Build() 中调用，此时已持有 c.mu.Lock()
		// 所以我们需要小心处理
		instance, exists := c.instances[tk]
		if !exists {
			return nil, fmt.Errorf("singleton instance not found for type %v (this should not happen)", tk.typ)
		}
		return instance, nil

	case ScopeTransient:
		// 瞬态：每次创建新实例
		return c.createTransientInstance(tk, info)

	case ScopeScoped:
		// 作用域：从当前作用域获取
		return c.getScopedInstance(tk, info)

	default:
		return nil, fmt.Errorf("unknown scope type: %v", info.scope)
	}
}

// getScopedInstance 获取作用域实例
func (c *Container) getScopedInstance(tk typeKey, info *providerInfo) (any, error) {
	// 获取当前作用域
	scope := c.GetCurrentScope()

	if scope == nil {
		// 没有当前作用域，返回错误提示
		return nil, fmt.Errorf(
			"DI: type %v is registered as Scoped, but no scope is active. "+
				"Please create a scope using container.CreateScope() and set it as current using container.SetCurrentScope()",
			tk.typ)
	}

	// 从作用域获取实例
	return scope.getInstanceByScope(tk, info)
}

// GetByType 通过类型获取实例（公开方法）
func (c *Container) GetByType(typ reflect.Type) (any, error) {
	tk := typeKey{typ: typ}
	return c.Get(tk)
}

// Provide 注册值或构造函数到容器实例
func (c *Container) Provide(value any) {
	if err := c.register(value); err != nil {
		panic("Container.Provide failed: " + err.Error())
	}
}

// ProvideType 使用类型提供者注册到容器实例
func (c *Container) ProvideType(provider TypeProvider) {
	if err := c.registerWithConfig(*provider.toProviderConfig()); err != nil {
		panic("Container.ProvideType failed: " + err.Error())
	}
}

// ProvideValue 使用值提供者注册到容器实例
func (c *Container) ProvideValue(provider ValueProvider) {
	if err := c.registerWithConfig(*provider.toProviderConfig()); err != nil {
		panic("Container.ProvideValue failed: " + err.Error())
	}
}

// ProvideFactory 使用工厂提供者注册到容器实例
func (c *Container) ProvideFactory(provider FactoryProvider) {
	if err := c.registerWithConfig(*provider.toProviderConfig()); err != nil {
		panic("Container.ProvideFactory failed: " + err.Error())
	}
}

// ProvideExisting 使用别名提供者注册到容器实例
func (c *Container) ProvideExisting(provider ExistingProvider) {
	if err := c.registerWithConfig(*provider.toProviderConfig()); err != nil {
		panic("Container.ProvideExisting failed: " + err.Error())
	}
}

// ProvideWithConfig 使用 ProviderConfig 注册到容器实例（支持完整配置）
func (c *Container) ProvideWithConfig(config ProviderConfig) {
	if err := c.registerWithConfig(config); err != nil {
		panic("Container.ProvideWithConfig failed: " + err.Error())
	}
}

// CreateScope 创建一个新的作用域
//
// 作用域用于管理 Scoped 生命周期的实例。在作用域内，
// Scoped 实例只创建一次并被缓存，多次获取返回同一个实例。
//
// 使用示例：
//
//	scope := container.CreateScope()
//	defer scope.Dispose()  // 确保释放资源
//
//	service, _ := scope.GetByType(reflect.TypeOf((*Service)(nil)))
//
// 注意：必须在 Build() 之后调用，否则会 panic。
func (c *Container) CreateScope() *Scope {
	if !c.built.Load() {
		panic("Cannot create scope before Build() is called")
	}

	return &Scope{
		parent:    c,
		instances: make(map[typeKey]any),
	}
}

// SetCurrentScope 设置当前作用域
//
// 此方法用于将作用域与当前执行上下文关联。
// 设置后，通过容器的 Get 方法获取 Scoped 实例时会使用此作用域。
//
// 使用场景：
//   - HTTP 中间件：在请求开始时设置作用域，请求结束时清理
//   - Goroutine 本地存储：每个 goroutine 有独立的作用域
//
// 注意：在并发场景中需要特别小心，确保不同请求/任务的作用域不会混淆。
func (c *Container) SetCurrentScope(scope *Scope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentScope = scope
}

// GetCurrentScope 获取当前作用域
//
// 返回通过 SetCurrentScope 设置的作用域，如果没有设置则返回 nil。
func (c *Container) GetCurrentScope() *Scope {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentScope
}

// ClearCurrentScope 清除当前作用域
//
// 在请求或任务结束后调用，清理作用域引用。
func (c *Container) ClearCurrentScope() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentScope = nil
}

// Get 从作用域获取实例
//
// 根据类型键获取实例，遵循以下规则：
//   - Singleton：从父容器的缓存获取
//   - Scoped：从当前作用域的缓存获取，不存在则创建
//   - Transient：每次创建新实例
//
// 如果作用域已被释放，返回错误。
func (s *Scope) Get(tk typeKey) (any, error) {
	if s.disposed.Load() {
		return nil, fmt.Errorf("scope has been disposed")
	}

	s.mu.RLock()
	info, exists := s.parent.providers[tk]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no provider found for type %v", tk.typ)
	}

	return s.getInstanceByScope(tk, info)
}

// GetByType 从作用域通过类型获取实例
//
// 这是 Get 方法的便捷版本，直接接受 reflect.Type 参数。
//
// 示例：
//
//	service, err := scope.GetByType(reflect.TypeOf((*UserService)(nil)))
func (s *Scope) GetByType(typ reflect.Type) (any, error) {
	tk := typeKey{typ: typ}
	return s.Get(tk)
}

// Inject 通过指针注入实例到目标变量（失败时 panic）
//
// 使用示例：
//
//	var svc *UserService
//	scope.Inject(&svc)
//
// 支持 Token 注入：
//
//	var svc *UserService
//	scope.Inject(&svc, token)
func (s *Scope) Inject(target any, tokenOrNil ...any) {
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("Scope.Inject: target must be a pointer, got %v", targetVal.Kind()))
	}

	if targetVal.IsNil() {
		panic("Scope.Inject: target pointer is nil")
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
			panic("Scope.Inject: invalid token parameter")
		}
	} else {
		// 按类型注入
		tk = typeKey{typ: elemType}
	}

	instance, err := s.Get(tk)
	if err != nil {
		panic(fmt.Sprintf("Scope.Inject failed: %v", err))
	}

	// 设置值
	elemVal.Set(reflect.ValueOf(instance))
}

// getInstanceByScope 根据作用域获取或创建实例
func (s *Scope) getInstanceByScope(tk typeKey, info *providerInfo) (any, error) {
	switch info.scope {
	case ScopeSingleton:
		// 单例：从父容器获取
		s.parent.mu.RLock()
		instance, exists := s.parent.instances[tk]
		s.parent.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("singleton instance not found for type %v", tk.typ)
		}
		return instance, nil

	case ScopeTransient:
		// 瞬态：每次创建新实例
		return s.createTransientInstance(tk, info)

	case ScopeScoped:
		// 作用域：从当前作用域获取或创建
		s.mu.RLock()
		instance, exists := s.instances[tk]
		s.mu.RUnlock()

		if exists {
			return instance, nil
		}

		// 创建新实例并缓存到作用域
		s.mu.Lock()
		defer s.mu.Unlock()

		// 双重检查
		if instance, exists := s.instances[tk]; exists {
			return instance, nil
		}

		instance, err := s.createTransientInstance(tk, info)
		if err != nil {
			return nil, err
		}

		s.instances[tk] = instance
		return instance, nil

	default:
		return nil, fmt.Errorf("unknown scope type: %v", info.scope)
	}
}

// createTransientInstance 创建瞬态实例（在作用域内）
func (s *Scope) createTransientInstance(tk typeKey, info *providerInfo) (any, error) {
	var instance any
	var err error

	switch info.providerType {
	case ProviderTypeValue:
		instance = info.value

	case ProviderTypeExisting:
		targetInfo, exists := s.parent.providers[info.existingKey]
		if !exists {
			return nil, fmt.Errorf("UseExisting target not found: %v", info.existingKey.typ)
		}
		return s.getInstanceByScope(info.existingKey, targetInfo)

	case ProviderTypeFactory:
		instance, err = s.invokeFactoryTransient(info)
		if err != nil {
			return nil, err
		}

	case ProviderTypeClass:
		if info.isFunc {
			instance, err = s.invokeConstructorTransient(info)
			if err != nil {
				return nil, err
			}
		} else {
			instance = info.value
		}
	}

	// 解析并执行字段注入
	fieldInjects, err := s.parent.resolveFieldInjectionsTransient(instance)
	if err != nil {
		return nil, err
	}

	if err := s.performFieldInjections(instance, fieldInjects); err != nil {
		return nil, err
	}

	return instance, nil
}

// invokeConstructorTransient 调用构造函数（在作用域内）
func (s *Scope) invokeConstructorTransient(info *providerInfo) (any, error) {
	fn := reflect.ValueOf(info.value)

	args := make([]reflect.Value, len(info.paramTypes))
	for i, paramType := range info.paramTypes {
		paramTk := typeKey{typ: paramType}
		paramInfo, exists := s.parent.providers[paramTk]
		if !exists {
			return nil, fmt.Errorf("no provider found for parameter type %v at index %d", paramType, i)
		}

		paramInstance, err := s.getInstanceByScope(paramTk, paramInfo)
		if err != nil {
			return nil, err
		}

		args[i] = reflect.ValueOf(paramInstance)
	}

	results := fn.Call(args)
	if len(results) == 0 {
		return nil, fmt.Errorf("constructor returned no values")
	}

	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				return nil, fmt.Errorf("constructor failed: %w", lastResult.Interface().(error))
			}
		}
	}

	firstResult := results[0]
	if firstResult.Kind() == reflect.Ptr || firstResult.Kind() == reflect.Interface {
		if firstResult.IsNil() {
			return nil, fmt.Errorf("constructor returned nil instance")
		}
	}

	return firstResult.Interface(), nil
}

// invokeFactoryTransient 调用工厂函数（在作用域内）
func (s *Scope) invokeFactoryTransient(info *providerInfo) (any, error) {
	fn := reflect.ValueOf(info.value)

	var args []reflect.Value

	if len(info.deps) > 0 {
		args = make([]reflect.Value, len(info.deps))
		for i, depKey := range info.deps {
			depInfo, exists := s.parent.providers[depKey]
			if !exists {
				return nil, fmt.Errorf("dependency not found: %v", depKey.typ)
			}

			depInstance, err := s.getInstanceByScope(depKey, depInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to get dependency %v: %w", depKey.typ, err)
			}

			args[i] = reflect.ValueOf(depInstance)
		}
	} else {
		args = make([]reflect.Value, len(info.paramTypes))
		for i, paramType := range info.paramTypes {
			paramTk := typeKey{typ: paramType}
			paramInfo, exists := s.parent.providers[paramTk]
			if !exists {
				return nil, fmt.Errorf("no provider found for factory parameter type %v at index %d", paramType, i)
			}

			paramInstance, err := s.getInstanceByScope(paramTk, paramInfo)
			if err != nil {
				return nil, err
			}

			args[i] = reflect.ValueOf(paramInstance)
		}
	}

	results := fn.Call(args)
	if len(results) == 0 {
		return nil, fmt.Errorf("factory returned no values")
	}

	if len(results) > 1 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				return nil, fmt.Errorf("factory failed: %w", lastResult.Interface().(error))
			}
		}
	}

	firstResult := results[0]
	if firstResult.Kind() == reflect.Ptr || firstResult.Kind() == reflect.Interface {
		if firstResult.IsNil() {
			return nil, fmt.Errorf("factory returned nil instance")
		}
	}

	return firstResult.Interface(), nil
}

// performFieldInjections 执行字段注入（在作用域内）
func (s *Scope) performFieldInjections(instance any, fieldInjects []fieldInjectInfo) error {
	if len(fieldInjects) == 0 {
		return nil
	}

	val := reflect.ValueOf(instance)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return fmt.Errorf("cannot perform field injection on nil pointer instance")
		}
		val = val.Elem()
	}

	instanceType := val.Type()

	for _, inject := range fieldInjects {
		fieldVal := val.Field(inject.fieldIndex)
		field := instanceType.Field(inject.fieldIndex)

		if !fieldVal.CanSet() {
			return fmt.Errorf("DI: cannot inject field '%s.%s' (type: %v) - field must be exported",
				instanceType.Name(), field.Name, inject.fieldType)
		}

		targetInfo, exists := s.parent.providers[inject.targetKey]
		if !exists {
			return fmt.Errorf("DI: %s.%s needs %v, but it's not registered",
				instanceType.Name(), field.Name, inject.targetKey.typ)
		}

		targetInstance, err := s.getInstanceByScope(inject.targetKey, targetInfo)
		if err != nil {
			return fmt.Errorf("DI: failed to get instance for %s.%s: %w",
				instanceType.Name(), field.Name, err)
		}

		fieldVal.Set(reflect.ValueOf(targetInstance))
	}

	return nil
}

// Dispose 释放作用域资源
//
// 释放作用域持有的所有实例缓存。释放后，作用域不能再使用。
// 建议使用 defer 确保资源被正确释放：
//
//	scope := container.CreateScope()
//	defer scope.Dispose()
//
// 多次调用 Dispose 是安全的，后续调用会被忽略。
func (s *Scope) Dispose() {
	if s.disposed.Load() {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 标记为已释放
	s.disposed.Store(true)

	// 清理实例缓存
	s.instances = nil
}
