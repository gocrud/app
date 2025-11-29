package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
	"github.com/stretchr/testify/assert"
)

// ---------------- Helper ----------------

func newTestLogger() logging.Logger {
	builder := logging.NewLoggingBuilder()
	builder.AddConsole(logging.ConsoleLoggerOptions{
		Output:      os.Stdout,
		ColorOutput: false,
	})
	factory := builder.Build()
	return factory.CreateLogger("test")
}

// ---------------- Mock Controllers ----------------

// SimpleController 普通控制器
type SimpleController struct {
	Check string
}

func (c *SimpleController) RegisterRoutes(router gin.IRouter) {
	router.GET("/simple", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "simple")
	})
}

// DepService 模拟依赖服务
type DepService struct {
	Value string
}

// ControllerWithDep 带依赖的控制器 (构造函数注入)
type ControllerWithDep struct {
	Svc *DepService
}

func NewControllerWithDep(svc *DepService) *ControllerWithDep {
	return &ControllerWithDep{Svc: svc}
}

func (c *ControllerWithDep) RegisterRoutes(router gin.IRouter) {
	router.GET("/dep", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, c.Svc.Value)
	})
}

// ControllerWithTag 带 Tag 的控制器 (实例注入)
type ControllerWithTag struct {
	Svc *DepService `di:""`
}

func (c *ControllerWithTag) RegisterRoutes(router gin.IRouter) {
	router.GET("/tag", func(ctx *gin.Context) {
		// 如果 Svc 未注入，这里会 panic，测试框架会捕获
		ctx.String(http.StatusOK, "tag:"+c.Svc.Value)
	})
}

// ---------------- Tests ----------------

func TestWebBuilder_AddControllers(t *testing.T) {
	// 1. Setup Environment
	logger := newTestLogger()
	container := di.NewContainer()

	// 注册依赖服务
	di.RegisterAuto(container, func() *DepService {
		return &DepService{Value: "injected-value"}
	})

	// 2. Create Builder & Add Controllers
	builder := NewBuilder(logger)

	// 方式 A: 构造函数
	builder.AddControllers(NewControllerWithDep)

	// 方式 B: 实例指针 (带 Tag)
	builder.AddControllers(&ControllerWithTag{})

	// 方式 C: 实例指针 (无依赖)
	builder.AddControllers(&SimpleController{})

	// 3. Build Host
	// 这里会触发 RegisterAuto 注册到容器
	host := builder.Build(container)

	// 4. Build Container
	// 必须在 host.Build 之后，Start 之前构建容器
	err := container.Build()
	assert.NoError(t, err)

	// 5. Map Controllers (通常在 Start 中调用，这里手动调用以测试)
	// 这会触发 Resolve 和 RegisterRoutes
	err = host.mapControllers()
	assert.NoError(t, err)

	// 6. Verify Routes using httptest
	router := host.engine

	// Case 1: Simple
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/simple", nil)
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "simple", w1.Body.String())

	// Case 2: Dependency Injection (Constructor)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/dep", nil)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "injected-value", w2.Body.String())

	// Case 3: Dependency Injection (Tag/Instance)
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/tag", nil)
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Equal(t, "tag:injected-value", w3.Body.String())
}

func TestWebBuilder_DuplicateRegistration(t *testing.T) {
	logger := newTestLogger()
	container := di.NewContainer()
	builder := NewBuilder(logger)

	// 故意添加两次相同的控制器
	builder.AddControllers(NewControllerWithDep)
	builder.AddControllers(NewControllerWithDep)

	// Build 不应 panic，而是记录警告并继续
	host := builder.Build(container)

	// 确保注册表中仍然包含该控制器（可能被去重或包含两个，只要不报错）
	// 我们的实现逻辑是：尝试注册，如果报错则 log warn，但仍然加入 types 列表。
	// 这样 container.Get(type) 可能会被调用两次，这对于 Singleton 也是安全的。
	assert.NotEmpty(t, host.controllerTypes)
}
