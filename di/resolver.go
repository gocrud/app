package di

import (
	"fmt"
	"reflect"
)

type resolver struct{}

func newResolver() *resolver {
	return &resolver{}
}

// createInstance 创建 def 描述的服务的新实例。
// 它使用提供的容器 c 递归解析依赖项。
func (r *resolver) createInstance(c Container, def *ServiceDefinition) (any, error) {
	if def.IsValue {
		return def.Impl, nil
	}

	if def.IsFactory {
		return r.invokeFunction(c, def.Impl, def.Schema)
	}

	// 如果 Impl 显式提供为函数（构造函数），则使用它
	if def.Impl != nil && reflect.TypeOf(def.Impl).Kind() == reflect.Func {
		return r.invokeFunction(c, def.Impl, def.Schema)
	}

	// 否则，视为结构体注入
	return r.createStruct(c, def)
}

// invokeFunction 调用工厂或构造函数。
// 它使用预计算的 schema 将依赖项注入函数参数。
func (r *resolver) invokeFunction(c Container, fn any, schema *InjectionSchema) (any, error) {
	fnVal := reflect.ValueOf(fn)
	// 使用 schema 获取参数类型而不是反射
	argTypes := schema.Args

	args := make([]reflect.Value, len(argTypes))
	for i, argType := range argTypes {
		argVal, err := c.Get(argType)
		if err != nil {
			return nil, fmt.Errorf("参数 %d: %w", i, err)
		}
		// 处理接口赋值
		args[i] = reflect.ValueOf(argVal)
	}

	results := fnVal.Call(args)

	if len(results) == 0 {
		return nil, fmt.Errorf("工厂/构造函数没有返回值")
	}

	// 检查最后一个返回值是否为错误
	if len(results) > 1 {
		last := results[len(results)-1]
		if last.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !last.IsNil() {
				return nil, last.Interface().(error)
			}
		}
	}

	// 返回第一个值
	return results[0].Interface(), nil
}

// createStruct 实例化结构体并注入标记为 `di` 的字段。
func (r *resolver) createStruct(c Container, def *ServiceDefinition) (any, error) {
	implType := def.ImplType

	// 确保我们正在处理用于实例化的底层结构体类型
	// 但通常我们返回指向它的指针，或匹配 ImplType。

	var val reflect.Value

	if implType.Kind() == reflect.Ptr {
		// 创建 Ptr -> Struct
		val = reflect.New(implType.Elem())
	} else {
		// 创建 Struct，但如果想支持值类型上的字段注入，我们需要它是可寻址的（较少见）
		// 最佳实践：始终注册结构体指针。
		// 如果用户注册了结构体值，reflect.New(implType) 返回 *Struct。
		val = reflect.New(implType)
	}

	// 在结构体上注入字段（val 在这里始终是指针）
	if err := r.injectFields(c, val.Elem(), def.Schema); err != nil {
		return nil, err
	}

	if implType.Kind() == reflect.Ptr {
		return val.Interface(), nil
	}
	return val.Elem().Interface(), nil
}

func (r *resolver) injectFields(c Container, structVal reflect.Value, schema *InjectionSchema) error {
	// 使用预计算 schema 仅迭代需要注入的字段
	for _, fieldInfo := range schema.Fields {
		// 解析依赖
		depVal, err := c.Get(fieldInfo.Type)
		if err != nil {
			if fieldInfo.Optional {
				continue
			}
			return fmt.Errorf("字段 %s: %w", fieldInfo.Name, err)
		}

		// 设置字段
		structVal.Field(fieldInfo.Index).Set(reflect.ValueOf(depVal))
	}
	return nil
}
