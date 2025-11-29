package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
)

// Builder Web 主机构建器（基于 Gin）
type Builder struct {
	logger          logging.Logger
	port            int
	engine          *gin.Engine
	controllerCtors []any // 存储控制器构造函数或实例
	registeredTypes []reflect.Type
}

// NewBuilder 创建 Web 构建器
func NewBuilder() *Builder {
	// 设置 Gin 为发布模式（默认）
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// 默认中间件：恢复 panic
	engine.Use(gin.Recovery())

	return &Builder{
		port:            8080,
		engine:          engine,
		controllerCtors: make([]any, 0),
		registeredTypes: make([]reflect.Type, 0),
	}
}

// UseLogger 设置日志记录器
func (b *Builder) UseLogger(logger logging.Logger) *Builder {
	b.logger = logger
	return b
}

// UsePort 设置端口
func (b *Builder) UsePort(port int) *Builder {
	b.port = port
	return b
}

// Use 使用全局中间件
func (b *Builder) Use(middleware ...gin.HandlerFunc) *Builder {
	b.engine.Use(middleware...)
	return b
}

// Controller 简单的控制器接口标记
type Controller interface {
	// MountRoutes 注册路由
	MountRoutes(router gin.IRouter)
}

// AddControllers 注册控制器
// 传入参数可以是：
// 1. 控制器的构造函数 (例如 NewUserController) -> 推荐，支持构造函数注入
// 2. 控制器实例指针 (例如 &UserController{}) -> 支持字段注入 (di tag)
// 这些控制器将在 Host 启动时通过 DI 容器进行解析和路由注册
func (b *Builder) AddControllers(controllers ...any) *Builder {
	b.controllerCtors = append(b.controllerCtors, controllers...)
	return b
}

// Get 注册 GET 路由
func (b *Builder) Get(path string, handlers ...gin.HandlerFunc) *Builder {
	b.engine.GET(path, handlers...)
	return b
}

// Post 注册 POST 路由
func (b *Builder) Post(path string, handlers ...gin.HandlerFunc) *Builder {
	b.engine.POST(path, handlers...)
	return b
}

// Put 注册 PUT 路由
func (b *Builder) Put(path string, handlers ...gin.HandlerFunc) *Builder {
	b.engine.PUT(path, handlers...)
	return b
}

// Delete 注册 DELETE 路由
func (b *Builder) Delete(path string, handlers ...gin.HandlerFunc) *Builder {
	b.engine.DELETE(path, handlers...)
	return b
}

// Patch 注册 PATCH 路由
func (b *Builder) Patch(path string, handlers ...gin.HandlerFunc) *Builder {
	b.engine.PATCH(path, handlers...)
	return b
}

// Any 注册任意方法路由
func (b *Builder) Any(path string, handlers ...gin.HandlerFunc) *Builder {
	b.engine.Any(path, handlers...)
	return b
}

// Group 创建路由组
func (b *Builder) Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return b.engine.Group(relativePath, handlers...)
}

// Static 服务静态文件
func (b *Builder) Static(relativePath, root string) *Builder {
	b.engine.Static(relativePath, root)
	return b
}

// StaticFS 服务静态文件系统
func (b *Builder) StaticFS(relativePath string, fs http.FileSystem) *Builder {
	b.engine.StaticFS(relativePath, fs)
	return b
}

// StaticFile 服务单个静态文件
func (b *Builder) StaticFile(relativePath, filepath string) *Builder {
	b.engine.StaticFile(relativePath, filepath)
	return b
}

// LoadHTMLGlob 加载 HTML 模板（通配符）
func (b *Builder) LoadHTMLGlob(pattern string) *Builder {
	b.engine.LoadHTMLGlob(pattern)
	return b
}

// LoadHTMLFiles 加载 HTML 模板（文件列表）
func (b *Builder) LoadHTMLFiles(files ...string) *Builder {
	b.engine.LoadHTMLFiles(files...)
	return b
}

// NoRoute 处理 404
func (b *Builder) NoRoute(handlers ...gin.HandlerFunc) *Builder {
	b.engine.NoRoute(handlers...)
	return b
}

// NoMethod 处理 405
func (b *Builder) NoMethod(handlers ...gin.HandlerFunc) *Builder {
	b.engine.NoMethod(handlers...)
	return b
}

// SetMode 设置 Gin 模式
func (b *Builder) SetMode(mode string) *Builder {
	gin.SetMode(mode)
	return b
}

// Engine 获取 Gin 引擎（用于高级定制）
func (b *Builder) Engine() *gin.Engine {
	return b.engine
}

// RegisterServices 注册服务到 DI 容器
// 必须在容器 Build 之前调用
func (b *Builder) RegisterServices(container di.Container) error {
	for _, item := range b.controllerCtors {
		serviceType, err := di.Provide(container, item)
		if err != nil {
			if b.logger != nil {
				b.logger.Warn("web: failed to auto-register controller (might be already registered or invalid)",
					logging.Field{Key: "error", Value: err.Error()},
					logging.Field{Key: "item", Value: fmt.Sprintf("%T", item)})
			}

			// 尝试手动推断类型以添加到 registeredTypes
			inferredType := inferServiceType(item)
			if inferredType != nil {
				b.registeredTypes = append(b.registeredTypes, inferredType)
			}
			continue
		}

		b.registeredTypes = append(b.registeredTypes, serviceType)
	}
	return nil
}

// Build 构建 Web 主机
// 这里的 container 必须是全局的 DI 容器，用于后续解析 Controller
func (b *Builder) Build(container di.Container) *Host {
	return &Host{
		port:            b.port,
		engine:          b.engine,
		container:       container,
		controllerTypes: b.registeredTypes,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", b.port),
			Handler: b.engine,
		},
		logger: b.logger,
	}
}

// inferServiceType 尝试推断服务类型（仅用于错误恢复）
func inferServiceType(target any) reflect.Type {
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Func {
		if val.Type().NumOut() > 0 {
			return val.Type().Out(0)
		}
	} else if val.Kind() == reflect.Ptr {
		return val.Type()
	} else if t, ok := target.(reflect.Type); ok {
		return t
	}
	return nil
}

// Host Web 主机
type Host struct {
	port            int
	engine          *gin.Engine
	server          *http.Server
	logger          logging.Logger
	container       di.Container
	controllerTypes []reflect.Type
}

// Address 获取监听地址 (e.g., "[::]:50234")
// 仅在 Start 后有效
func (h *Host) Address() string {
	if h.server != nil {
		return h.server.Addr
	}
	return ""
}

// Start 启动 Web 主机
// 注意：此方法会阻塞，直到服务退出。框架会在独立的 Goroutine 中调用它。
func (h *Host) Start(ctx context.Context) error {
	// 1. 延迟解析并注册控制器路由
	if err := h.mapControllers(); err != nil {
		return fmt.Errorf("web: failed to map controllers: %w", err)
	}

	// 2. 监听端口 (同步，确保端口可用)
	addr := fmt.Sprintf(":%d", h.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("web: failed to listen on %s: %w", addr, err)
	}

	// 更新 server 地址
	h.server.Addr = ln.Addr().String()

	if h.logger != nil {
		h.logger.Info("Web host started",
			logging.Field{Key: "address", Value: h.server.Addr})
	}

	// 3. 启动服务 (阻塞)
	// Serve 会一直阻塞直到 Shutdown 被调用或发生错误
	if err := h.server.Serve(ln); err != nil && err != http.ErrServerClosed {
		if h.logger != nil {
			h.logger.Error("Web host error", logging.Field{Key: "error", Value: err.Error()})
		}
		return err
	}

	return nil
}

// Stop 停止 Web 主机
func (h *Host) Stop(ctx context.Context) error {
	if h.logger != nil {
		h.logger.Info("Stopping web host")
	}

	if err := h.server.Shutdown(ctx); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to shutdown web host gracefully",
				logging.Field{Key: "error", Value: err.Error()})
		}
		return err
	}

	if h.logger != nil {
		h.logger.Info("Web host stopped")
	}
	return nil
}

// mapControllers 从容器解析并注册控制器
func (h *Host) mapControllers() error {
	for _, typ := range h.controllerTypes {
		// 从容器获取实例
		instance, err := h.container.Get(typ)
		if err != nil {
			return fmt.Errorf("failed to resolve controller %v: %w", typ, err)
		}

		// 断言为 Controller 接口
		ctrl, ok := instance.(Controller)
		if !ok {
			return fmt.Errorf("instance %v does not implement web.Controller interface", typ)
		}

		// 注册路由
		ctrl.MountRoutes(h.engine)
		if h.logger != nil {
			h.logger.Debug("Mapped controller routes", logging.Field{Key: "controller", Value: typ.String()})
		}
	}
	return nil
}
