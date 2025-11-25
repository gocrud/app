package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocrud/app/logging"
)

// Builder Web 主机构建器（基于 Gin）
type Builder struct {
	logger logging.Logger
	port   int
	engine *gin.Engine
}

// NewBuilder 创建 Web 构建器
func NewBuilder(logger logging.Logger) *Builder {
	// 设置 Gin 为发布模式（默认）
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// 默认中间件：恢复 panic
	engine.Use(gin.Recovery())

	return &Builder{
		logger: logger,
		port:   8080,
		engine: engine,
	}
}

// UsePort 设置端口
func (b *Builder) UsePort(port int) *Builder {
	b.port = port
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

// Use 使用全局中间件
func (b *Builder) Use(middleware ...gin.HandlerFunc) *Builder {
	b.engine.Use(middleware...)
	return b
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
func (b *Builder) Build() *Host {
	return &Host{
		port:   b.port,
		engine: b.engine,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", b.port),
			Handler: b.engine, // Gin Engine 实现了 http.Handler
		},
		logger: b.logger,
	}
}

// Host Web 主机
type Host struct {
	port   int
	engine *gin.Engine
	server *http.Server
	logger logging.Logger
}

// Start 启动 Web 主机
func (h *Host) Start(ctx context.Context) error {
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
