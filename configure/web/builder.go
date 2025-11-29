package web

import (
	"context"
	"fmt"
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
}

// NewBuilder 创建 Web 构建器
func NewBuilder(logger logging.Logger) *Builder {
	// 设置 Gin 为发布模式（默认）
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// 默认中间件：恢复 panic
	engine.Use(gin.Recovery())

	return &Builder{
		logger:          logger,
		port:            8080,
		engine:          engine,
		controllerCtors: make([]any, 0),
	}
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
	// RegisterRoutes 注册路由
	RegisterRoutes(router gin.IRouter)
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

// Build 构建 Web 主机
// 这里的 container 必须是全局的 DI 容器，用于后续解析 Controller
func (b *Builder) Build(container di.Container) *Host {
	// 将所有控制器构造函数/实例注册到 DI 容器中
	registeredTypes := make([]reflect.Type, 0, len(b.controllerCtors))

	for _, item := range b.controllerCtors {
		// 使用 di.RegisterAuto 进行智能注册
		// 它会自动处理构造函数或实例指针，并支持字段注入
		serviceType, err := di.RegisterAuto(container, item)
		if err != nil {
			// 如果是因为重复注册，我们记录警告并继续使用该类型
			// 但目前的 RegisterAuto 并没有返回错误类型区分，
			// 且如果注册失败我们也不知道已注册的服务类型是什么（除非我们自己再次推断）。
			// 为了健壮性，如果注册失败，我们尝试推断类型并记录，以便后续尝试 Resolve。

			// 简单的策略：如果是“已注册”错误，我们假设用户手动注册了，尝试推断类型并加入列表。
			// 但这里我们简单地记录警告，并不阻断 Build，
			// 因为如果真正出错（如不支持的类型），Resolve 时自然会失败。
			b.logger.Warn("web: failed to auto-register controller (might be already registered or invalid)",
				logging.Field{Key: "error", Value: err.Error()},
				logging.Field{Key: "item", Value: fmt.Sprintf("%T", item)})

			// 尝试手动推断类型以添加到 registeredTypes
			// 这样即使注册失败（因重复），我们也能尝试在 Start 时 Resolve 它
			inferredType := inferServiceType(item)
			if inferredType != nil {
				registeredTypes = append(registeredTypes, inferredType)
			}
			continue
		}

		registeredTypes = append(registeredTypes, serviceType)
	}

	return &Host{
		port:            b.port,
		engine:          b.engine,
		container:       container,
		controllerTypes: registeredTypes,
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

// Start 启动 Web 主机
func (h *Host) Start(ctx context.Context) error {
	// 1. 延迟解析并注册控制器路由
	if err := h.mapControllers(); err != nil {
		return fmt.Errorf("web: failed to map controllers: %w", err)
	}

	h.logger.Info("Starting web host",
		logging.Field{Key: "port", Value: h.port})

	// 启动服务器（阻塞调用）
	// 在单独的 goroutine 中监听关闭信号

	errCh := make(chan error, 1)
	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	h.logger.Info("Web host started",
		logging.Field{Key: "address", Value: h.server.Addr})

	// 等待错误或上下文取消
	select {
	case err := <-errCh:
		if err != nil {
			h.logger.Error("Web host error",
				logging.Field{Key: "error", Value: err.Error()})
			return err
		}
		return nil
	case <-ctx.Done():
		// 上下文取消，触发关闭
		return nil // Stop 会负责关闭
	}
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
		ctrl.RegisterRoutes(h.engine)
		h.logger.Debug("Mapped controller routes", logging.Field{Key: "controller", Value: typ.String()})
	}
	return nil
}

// Stop 停止 Web 主机
func (h *Host) Stop(ctx context.Context) error {
	h.logger.Info("Stopping web host")

	if err := h.server.Shutdown(ctx); err != nil {
		h.logger.Error("Failed to shutdown web host gracefully",
			logging.Field{Key: "error", Value: err.Error()})
		return err
	}

	h.logger.Info("Web host stopped")
	return nil
}
