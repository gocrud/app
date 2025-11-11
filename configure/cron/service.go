package cron

import (
	"context"
	"fmt"
	"sync"

	"github.com/gocrud/app/logging"
	"github.com/robfig/cron/v3"
)

// service Cron 定时任务托管服务
// 实现 HostedService 接口，与框架无缝集成
type service struct {
	cron   *cron.Cron
	logger logging.Logger
	mu     sync.RWMutex
	jobs   map[string]cron.EntryID // 任务名称到任务ID的映射
}

// options Cron 服务配置选项
type options struct {
	// Location 时区设置，默认 UTC
	Location string
	// EnableSeconds 是否启用秒级精度（默认分钟级）
	EnableSeconds bool
	// Logger 自定义日志记录器
	Logger logging.Logger
	// EnableCronLogger 是否启用 cron 库的内部调度日志（默认 false）
	EnableCronLogger bool
}

// newService 创建 Cron 托管服务
func newService(logger logging.Logger, opts ...func(*options)) *service {
	opt := &options{
		Location:         "UTC",
		EnableSeconds:    false,
		Logger:           logger,
		EnableCronLogger: false,
	}

	for _, o := range opts {
		o(opt)
	}

	// 配置 cron 选项
	cronOpts := []cron.Option{}

	// 只在启用时添加 cron 库的日志记录器
	if opt.EnableCronLogger {
		cronOpts = append(cronOpts, cron.WithLogger(newCronLogger(opt.Logger)))
	}

	cronOpts = append(cronOpts, cron.WithChain(
		cron.Recover(newCronLogger(opt.Logger)),
	))

	if opt.EnableSeconds {
		cronOpts = append(cronOpts, cron.WithSeconds())
	}

	return &service{
		cron:   cron.New(cronOpts...),
		logger: opt.Logger,
		jobs:   make(map[string]cron.EntryID),
	}
}

// addJob 添加定时任务
// spec: cron 表达式，如 "0 */5 * * * *" (每5分钟) 或 "0 0 2 * * *" (每天凌晨2点)
// name: 任务名称（用于管理和日志）
// job: 任务函数
func (s *service) addJob(spec, name string, job func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, err := s.cron.AddFunc(spec, func() {
		s.logger.Info(fmt.Sprintf("Cron job '%s' started", name))
		defer s.logger.Info(fmt.Sprintf("Cron job '%s' completed", name))
		job()
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job '%s': %w", name, err)
	}

	s.jobs[name] = entryID
	s.logger.Info(fmt.Sprintf("Cron job '%s' registered with spec '%s'", name, spec))
	return nil
}

// removeJob 移除定时任务
func (s *service) removeJob(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[name]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, name)
		s.logger.Info(fmt.Sprintf("Cron job '%s' removed", name))
	}
}

// StartAsync 实现 HostedService.StartAsync
func (s *service) StartAsync(ctx context.Context) error {
	s.logger.Info(fmt.Sprintf("CronService starting with %d jobs", len(s.jobs)))
	s.cron.Start()

	// 阻塞直到上下文取消
	<-ctx.Done()
	return nil
}

// StopAsync 实现 HostedService.StopAsync
func (s *service) StopAsync(ctx context.Context) error {
	s.logger.Info("CronService stopping")

	// 优雅停止 cron（等待正在运行的任务完成）
	stopCtx := s.cron.Stop()

	select {
	case <-stopCtx.Done():
		s.logger.Info("CronService stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("CronService stop timeout, forcing shutdown")
	}

	return nil
}

// cronLogger 适配器：将框架日志接口适配到 cron 的日志接口
type cronLogger struct {
	logger logging.Logger
}

func newCronLogger(logger logging.Logger) cron.Logger {
	return &cronLogger{logger: logger}
}

func (l *cronLogger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Info(msg, convertToFields(keysAndValues)...)
}

func (l *cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	fields := convertToFields(keysAndValues)
	fields = append(fields, logging.Field{Key: "error", Value: err.Error()})
	l.logger.Error(msg, fields...)
}

func convertToFields(keysAndValues []interface{}) []logging.Field {
	fields := make([]logging.Field, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := fmt.Sprintf("%v", keysAndValues[i])
			value := keysAndValues[i+1]
			fields = append(fields, logging.Field{Key: key, Value: value})
		}
	}
	return fields
}
