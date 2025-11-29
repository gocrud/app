package cron

import (
	"fmt"
	"reflect"

	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
)

// Builder Cron 配置构建器
type Builder struct {
	enableSeconds    bool
	enableCronLogger bool
	location         string
	jobs             []jobDefinition
}

// jobDefinition 任务定义
// type jobDefinition struct { // moved to service.go
// 	spec    string
// 	name    string
// 	handler any // 可以是 func() 或依赖注入的函数
// }

// NewBuilder 创建 Cron 构建器
func NewBuilder() *Builder {
	return &Builder{
		enableSeconds:    false,
		enableCronLogger: false,
		location:         "UTC",
		jobs:             make([]jobDefinition, 0),
	}
}

// WithSeconds 启用秒级精度
func (b *Builder) WithSeconds() *Builder {
	b.enableSeconds = true
	return b
}

// WithLocation 设置时区
func (b *Builder) WithLocation(location string) *Builder {
	b.location = location
	return b
}

// EnableCronLogger 启用 cron 库的内部调度日志
func (b *Builder) EnableCronLogger() *Builder {
	b.enableCronLogger = true
	return b
}

// AddJob 添加简单任务（无依赖注入）
func (b *Builder) AddJob(spec, name string, handler func()) *Builder {
	b.jobs = append(b.jobs, jobDefinition{
		spec:    spec,
		name:    name,
		handler: handler,
	})
	return b
}

// AddJobWithDI 添加带依赖注入的任务
// handler 可以是任何函数，参数会自动从 DI 容器解析
//
// 示例：
//
//	builder.AddJobWithDI("0 */5 * * * *", "sync-data", func(svc *DataService, logger logging.Logger) {
//	    svc.Sync()
//	})
func (b *Builder) AddJobWithDI(spec, name string, handler any) *Builder {
	b.jobs = append(b.jobs, jobDefinition{
		spec:    spec,
		name:    name,
		handler: handler,
	})
	return b
}

// build 构建 CronService（内部使用）
func (b *Builder) build(logger logging.Logger) (*service, error) {
	// 创建 cronService
	cronSvc := newService(logger, func(opts *options) {
		opts.EnableSeconds = b.enableSeconds
		opts.EnableCronLogger = b.enableCronLogger
		opts.Location = b.location
		opts.Logger = logger
	})

	// 注册所有简单任务 (需要 DI 的任务延迟到 Start 时注册)
	// 但目前我们没有机制区分，所以这里只处理简单任务
	// 对于带 DI 的任务，我们在 wrapper 中处理，但是容器此时未知
	// 所以我们需要在 Start 时才真正的 AddJob 吗？
	// 或者我们通过 Inject 方法传入 Container

	// 暂存 jobs 到 service
	cronSvc.jobDefs = b.jobs // 需要在 service 中添加字段

	return cronSvc, nil
}

// wrapHandlerWithDI 包装处理器，注入依赖
// 移动到单独的辅助函数文件或作为 service 的方法
// 这里暂时保留以兼容 build 方法，但实际已在 service.go 中重新实现逻辑
// 我们可以将逻辑移到 helper.go
func wrapHandlerWithDI(container di.Container, logger logging.Logger, handler any) (func(), error) {
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	// 检查是否为函数
	if handlerType.Kind() != reflect.Func {
		return nil, fmt.Errorf("handler must be a function, got %v", handlerType.Kind())
	}

	// 返回包装函数
	wrappedFunc := func() {
		// 解析函数参数
		numIn := handlerType.NumIn()
		args := make([]reflect.Value, numIn)

		for i := 0; i < numIn; i++ {
			paramType := handlerType.In(i)

			// 从容器获取实例
			instance, err := container.Get(paramType)
			if err != nil {
				if logger != nil {
					logger.Error(fmt.Sprintf("Failed to resolve parameter %d (%v) for cron job", i, paramType),
						logging.Field{Key: "error", Value: err.Error()})
				}
				return
			}

			args[i] = reflect.ValueOf(instance)
		}

		// 调用处理函数
		defer func() {
			if r := recover(); r != nil {
				if logger != nil {
					logger.Error("Cron job panicked",
						logging.Field{Key: "panic", Value: r})
				} else {
					fmt.Println("Cron job panicked:", r)
				}
			}
		}()

		handlerValue.Call(args)
	}

	return wrappedFunc, nil
}
