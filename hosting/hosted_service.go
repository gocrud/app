package hosting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gocrud/app/logging"
)

// HostedService 托管服务接口（类似于 .NET Core IHostedService）
// 框架会自动在 goroutine 中调用 Start，用户无需自己启动 goroutine
type HostedService interface {
	// Start 启动服务。该方法应阻塞执行，直到 context 被取消或发生错误。
	// 框架会在独立的 goroutine 中调用此方法。
	Start(ctx context.Context) error

	// Stop 执行优雅关闭逻辑。
	// 注意：当 Start 的 context 被取消时，服务应自动停止。
	// Stop 方法用于执行额外的清理工作（可选）。
	Stop(ctx context.Context) error
}

// HostedServiceManager 托管服务管理器
type HostedServiceManager struct {
	services []HostedService
	logger   logging.Logger
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

// NewHostedServiceManager 创建托管服务管理器
func NewHostedServiceManager(logger logging.Logger) *HostedServiceManager {
	return &HostedServiceManager{
		services: make([]HostedService, 0),
		logger:   logger,
	}
}

// Add 添加托管服务
func (m *HostedServiceManager) Add(service HostedService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services = append(m.services, service)
}

// StartAll 启动所有托管服务
// 框架层面处理并发，每个服务在独立的 goroutine 中启动
func (m *HostedServiceManager) StartAll(ctx context.Context) <-chan error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	errCh := make(chan error, len(m.services))

	m.logger.Info(fmt.Sprintf("Starting %d hosted services", len(m.services)))

	// 并发启动所有服务
	for i, service := range m.services {
		m.wg.Add(1)
		go func(index int, svc HostedService) {
			defer m.wg.Done()

			m.logger.Debug(fmt.Sprintf("Starting hosted service %d", index+1))

			// 在 goroutine 中调用 Start
			if err := svc.Start(ctx); err != nil {
				// 区分正常的 context 取消和真正的错误
				if err == context.Canceled || err == context.DeadlineExceeded {
					m.logger.Debug(fmt.Sprintf("Hosted service %d stopped (context done)", index+1))
				} else {
					m.logger.Error(fmt.Sprintf("Hosted service %d error", index+1),
						logging.Field{Key: "error", Value: err.Error()})
					// 发送错误到通道
					select {
					case errCh <- err:
					default:
						// 通道满，忽略错误（不应该发生，因为通道缓冲区大小等于服务数量）
					}
				}
				return
			}

			m.logger.Info(fmt.Sprintf("Hosted service %d completed", index+1))
		}(i, service)
	}

	m.logger.Info("All hosted services started")
	return errCh
}

// StopAll 停止所有托管服务
// 框架层面处理并发停止
func (m *HostedServiceManager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.logger.Info(fmt.Sprintf("Stopping %d hosted services", len(m.services)))

	var wg sync.WaitGroup

	// 反向并发停止服务
	for i := len(m.services) - 1; i >= 0; i-- {
		service := m.services[i]
		index := i

		wg.Add(1)
		go func(idx int, svc HostedService) {
			defer wg.Done()

			m.logger.Debug(fmt.Sprintf("Stopping hosted service %d", idx+1))

			// 在 goroutine 中调用 Stop
			if err := svc.Stop(ctx); err != nil {
				m.logger.Error(fmt.Sprintf("Failed to stop hosted service %d", idx+1),
					logging.Field{Key: "error", Value: err.Error()})
			} else {
				m.logger.Info(fmt.Sprintf("Hosted service %d stopped successfully", idx+1))
			}
		}(index, service)
	}

	// 等待所有服务停止完成
	wg.Wait()

	m.logger.Info("All hosted services stopped")
	return nil
}

// Wait 等待所有服务完成
func (m *HostedServiceManager) Wait() {
	m.wg.Wait()
}

// BackgroundService 后台服务基类
type BackgroundService struct {
	name   string
	logger logging.Logger
	stopCh chan struct{}
	doneCh chan struct{}
}

// NewBackgroundService 创建后台服务
func NewBackgroundService(name string, logger logging.Logger) *BackgroundService {
	return &BackgroundService{
		name:   name,
		logger: logger,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// Start 启动后台服务
func (s *BackgroundService) Start(ctx context.Context) error {
	s.logger.Info(fmt.Sprintf("BackgroundService '%s' starting", s.name))
	
	// 阻塞直到停止信号或上下文取消
	select {
	case <-s.stopCh:
		s.logger.Info(fmt.Sprintf("BackgroundService '%s' stopped by signal", s.name))
	case <-ctx.Done():
		s.logger.Info(fmt.Sprintf("BackgroundService '%s' context cancelled", s.name))
	}

	s.Done()
	return nil
}

// Stop 停止后台服务
func (s *BackgroundService) Stop(ctx context.Context) error {
	s.logger.Info(fmt.Sprintf("BackgroundService '%s' stopping", s.name))
	close(s.stopCh)

	// 等待服务停止或超时
	select {
	case <-s.doneCh:
		s.logger.Info(fmt.Sprintf("BackgroundService '%s' stopped gracefully", s.name))
	case <-ctx.Done():
		s.logger.Warn(fmt.Sprintf("BackgroundService '%s' stop timeout", s.name))
		return ctx.Err()
	}

	return nil
}

// ShouldStop 检查是否应该停止
func (s *BackgroundService) ShouldStop() bool {
	select {
	case <-s.stopCh:
		return true
	default:
		return false
	}
}

// StopChan 返回停止通道，用于在 select 中监听
func (s *BackgroundService) StopChan() <-chan struct{} {
	return s.stopCh
}

// Done 标记服务完成
func (s *BackgroundService) Done() {
	// 防止多次关闭
	select {
	case <-s.doneCh:
		return
	default:
		close(s.doneCh)
	}
}

// TimedHostedService 定时托管服务
type TimedHostedService struct {
	*BackgroundService
	interval time.Duration
	task     func(ctx context.Context) error
}

// NewTimedHostedService 创建定时托管服务
func NewTimedHostedService(name string, interval time.Duration, task func(ctx context.Context) error, logger logging.Logger) *TimedHostedService {
	return &TimedHostedService{
		BackgroundService: NewBackgroundService(name, logger),
		interval:          interval,
		task:              task,
	}
}

// Start 启动定时服务
func (s *TimedHostedService) Start(ctx context.Context) error {
	s.logger.Info(fmt.Sprintf("TimedHostedService '%s' running with interval %v", s.name, s.interval))
	return s.run(ctx)
}

func (s *TimedHostedService) run(ctx context.Context) error {
	defer s.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.logger.Debug(fmt.Sprintf("TimedHostedService '%s' executing task", s.name))
			if err := s.task(ctx); err != nil {
				s.logger.Error(fmt.Sprintf("TimedHostedService '%s' task failed", s.name),
					logging.Field{Key: "error", Value: err.Error()})
			}
		case <-s.stopCh:
			s.logger.Info(fmt.Sprintf("TimedHostedService '%s' stopped", s.name))
			return nil
		case <-ctx.Done():
			s.logger.Info(fmt.Sprintf("TimedHostedService '%s' context cancelled", s.name))
			return ctx.Err()
		}
	}
}
