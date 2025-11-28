package di

import (
	"fmt"
	"reflect"
	"strings"
)

// graphBuilder 处理依赖图的构建和验证。
type graphBuilder struct {
	definitions map[ServiceKey]*ServiceDefinition
}

func newGraphBuilder(defs map[ServiceKey]*ServiceDefinition) *graphBuilder {
	return &graphBuilder{
		definitions: defs,
	}
}

// buildOrder 返回单例的最佳构建顺序并验证图。
func (g *graphBuilder) buildOrder() ([]ServiceKey, error) {
	dependencies := make(map[ServiceKey][]ServiceKey)

	// 1. 提取所有服务的依赖关系
	for key, def := range g.definitions {
		deps, err := g.inspectDependencies(def)
		if err != nil {
			return nil, fmt.Errorf("检查 %v (name=%s) 的依赖失败: %w", key.Type, key.Name, err)
		}
		dependencies[key] = deps
	}

	// 2. 拓扑排序 (基于 DFS)
	visited := make(map[ServiceKey]bool)
	recursionStack := make(map[ServiceKey]bool)
	var order []ServiceKey

	var visit func(ServiceKey) error
	visit = func(u ServiceKey) error {
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
				return fmt.Errorf("检测到循环依赖: %v(name=%s) -> %v(name=%s)", u.Type, u.Name, v.Type, v.Name)
			}
		}

		recursionStack[u] = false
		order = append(order, u)
		return nil
	}

	for key := range g.definitions {
		if !visited[key] {
			if err := visit(key); err != nil {
				return nil, err
			}
		}
	}

	return order, nil
}

// inspectDependencies 返回服务依赖的类型列表。
// 它还会填充 ServiceDefinition.Schema。
func (g *graphBuilder) inspectDependencies(def *ServiceDefinition) ([]ServiceKey, error) {
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

func (g *graphBuilder) analyzeFunction(fn any, schema *InjectionSchema) ([]ServiceKey, error) {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("期望函数，得到 %v", fnType)
	}

	var deps []ServiceKey
	for i := 0; i < fnType.NumIn(); i++ {
		argType := fnType.In(i)
		// 工厂函数参数暂不支持命名注入，默认为空名称
		key := ServiceKey{Type: argType, Name: ""}
		deps = append(deps, key)
		schema.Args = append(schema.Args, argType)
	}
	return deps, nil
}

func (g *graphBuilder) analyzeStruct(typ reflect.Type, schema *InjectionSchema) ([]ServiceKey, error) {
	// 解包指针
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, nil
	}

	var deps []ServiceKey
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tagValue, hasTag := field.Tag.Lookup("di")
		if !hasTag {
			continue
		}

		// 解析 tag: "name,option1,option2"
		parts := strings.Split(tagValue, ",")
		name := strings.TrimSpace(parts[0])
		isOptional := false

		// 处理 "di:?" 或 "di:optional" 的情况，此时 name 应为空
		if name == "?" || name == "optional" {
			name = ""
			isOptional = true
		}

		for _, part := range parts[1:] {
			part = strings.TrimSpace(part)
			if part == "optional" || part == "?" {
				isOptional = true
			}
		}

		// 记录字段注入元数据
		schema.Fields = append(schema.Fields, FieldInjection{
			Index:       i,
			Name:        field.Name,
			Type:        field.Type,
			Optional:    isOptional,
			ServiceName: name,
		})

		if isOptional {
			continue // 不在图中强制执行可选依赖
		}
		deps = append(deps, ServiceKey{Type: field.Type, Name: name})
	}
	return deps, nil
}
