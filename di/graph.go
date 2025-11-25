package di

import (
	"fmt"
	"reflect"
)

// graphBuilder 处理依赖图的构建和验证。
type graphBuilder struct {
	definitions map[reflect.Type]*ServiceDefinition
}

func newGraphBuilder(defs map[reflect.Type]*ServiceDefinition) *graphBuilder {
	return &graphBuilder{
		definitions: defs,
	}
}

// buildOrder 返回单例的最佳构建顺序并验证图。
func (g *graphBuilder) buildOrder() ([]reflect.Type, error) {
	dependencies := make(map[reflect.Type][]reflect.Type)

	// 1. 提取所有服务的依赖关系
	for typ, def := range g.definitions {
		deps, err := g.inspectDependencies(def)
		if err != nil {
			return nil, fmt.Errorf("检查 %v 的依赖失败: %w", typ, err)
		}
		dependencies[typ] = deps
	}

	// 2. 拓扑排序 (基于 DFS)
	visited := make(map[reflect.Type]bool)
	recursionStack := make(map[reflect.Type]bool)
	var order []reflect.Type

	var visit func(reflect.Type) error
	visit = func(u reflect.Type) error {
		visited[u] = true
		recursionStack[u] = true

		for _, v := range dependencies[u] {
			// 如果依赖项未注册，我们无法检查它（稍后会出现运行时错误），
			// 或者我们在这里失败。严格模式：在这里失败。
			// 但对于可选依赖项，我们可能会跳过。
			// 目前，如果它不在定义中，我们跳过对它的图检查
			// （这可能是运行时的依赖缺失错误，或由父级处理）。
			if _, exists := g.definitions[v]; !exists {
				continue
			}

			if !visited[v] {
				if err := visit(v); err != nil {
					return err
				}
			} else if recursionStack[v] {
				return fmt.Errorf("检测到循环依赖: %v -> %v", u, v)
			}
		}

		recursionStack[u] = false
		order = append(order, u)
		return nil
	}

	for typ := range g.definitions {
		if !visited[typ] {
			if err := visit(typ); err != nil {
				return nil, err
			}
		}
	}

	return order, nil
}

// inspectDependencies 返回服务依赖的类型列表。
// 它还会填充 ServiceDefinition.Schema。
func (g *graphBuilder) inspectDependencies(def *ServiceDefinition) ([]reflect.Type, error) {
	def.Schema = &InjectionSchema{}

	// 情况 1: 值 - 无依赖
	if def.IsValue {
		return nil, nil
	}

	// 情况 2: 工厂函数
	if def.IsFactory {
		return g.analyzeFunction(def.Impl, def.Schema)
	}

	// 情况 3: 构造函数 (如果 Impl 是函数)
	if def.Impl != nil && reflect.TypeOf(def.Impl).Kind() == reflect.Func {
		return g.analyzeFunction(def.Impl, def.Schema)
	}

	// 情况 4: 结构体注入 (ImplType)
	return g.analyzeStruct(def.ImplType, def.Schema)
}

func (g *graphBuilder) analyzeFunction(fn any, schema *InjectionSchema) ([]reflect.Type, error) {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("期望函数，得到 %v", fnType)
	}

	var deps []reflect.Type
	for i := 0; i < fnType.NumIn(); i++ {
		argType := fnType.In(i)
		deps = append(deps, argType)
		schema.Args = append(schema.Args, argType)
	}
	return deps, nil
}

func (g *graphBuilder) analyzeStruct(typ reflect.Type, schema *InjectionSchema) ([]reflect.Type, error) {
	// 解包指针
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, nil
	}

	var deps []reflect.Type
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag, hasTag := field.Tag.Lookup("di")
		if !hasTag {
			continue
		}

		isOptional := tag == "optional" || tag == "?"

		// 记录字段注入元数据
		schema.Fields = append(schema.Fields, FieldInjection{
			Index:    i,
			Name:     field.Name,
			Type:     field.Type,
			Optional: isOptional,
		})

		if isOptional {
			continue // 不在图中强制执行可选依赖
		}
		deps = append(deps, field.Type)
	}
	return deps, nil
}
