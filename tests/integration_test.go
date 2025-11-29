package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocrud/app/config"
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/web"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// MockDialector 最小化实现，跳过实际 DB 连接
type MockDialector struct{}

func (m MockDialector) Name() string                                                        { return "mock" }
func (m MockDialector) Initialize(db *gorm.DB) error                                        { return nil }
func (m MockDialector) Migrator(db *gorm.DB) gorm.Migrator                                  { return nil }
func (m MockDialector) DataTypeOf(field *schema.Field) string                               { return "" }
func (m MockDialector) DefaultValueOf(field *schema.Field) clause.Expression                { return clause.Expr{} }
func (m MockDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {}
func (m MockDialector) QuoteTo(writer clause.Writer, str string)                            {}
func (m MockDialector) Explain(sql string, vars ...interface{}) string                      { return "" }

// TestService 模拟业务服务
type TestService struct {
	DB     *gorm.DB             `di:""`
	Config config.Configuration `di:""`
}

func (s *TestService) GetAppName() string {
	if s.Config == nil {
		return "no-config"
	}
	return s.Config.Get("app.name")
}

// TestController 模拟控制器
type TestController struct {
	Service *TestService
}

// NewTestController 使用构造函数注入
func NewTestController(svc *TestService) *TestController {
	return &TestController{Service: svc}
}

func (c *TestController) MountRoutes(r gin.IRouter) {
	r.GET("/ping", func(ctx *gin.Context) {
		name := "unknown"
		if c.Service != nil {
			name = c.Service.GetAppName()
		}
		// Verify DB injection
		if c.Service != nil && c.Service.DB == nil {
			name += "-nodb"
		}
		ctx.String(200, "pong: "+name)
	})
}

func TestIntegration(t *testing.T) {
	rt := core.NewRuntime()

	// 手动设置配置环境变量
	t.Setenv("TEST_APP_NAME", "IntegrationTest")

	// 应用模块
	err := rt.Apply(
		// 1. Config
		func(rt *core.Runtime) error {
			cfg := config.NewConfiguration()
			// 加载环境变量
			cfg.LoadEnv("TEST_")
			// 注册到容器
			di.ProvideService[config.Configuration](rt.Container, di.WithValue(cfg))
			return nil
		},

		// 2. Database (Mock Component)
		func(rt *core.Runtime) error {
			mockDB := &gorm.DB{}
			// 注册默认数据库
			return rt.Provide(mockDB, di.WithValue(mockDB))
		},

		// 3. Web (Random Port)
		web.New(web.WithControllers(NewTestController), web.WithPort(0)),
	)
	if err != nil {
		t.Fatalf("Apply options failed: %v", err)
	}

	// 注册业务服务
	if err := rt.Provide(&TestService{}); err != nil {
		t.Fatalf("Provide TestService failed: %v", err)
	}

	// 构建容器
	if err := rt.Container.Build(); err != nil {
		t.Fatalf("Container build failed: %v", err)
	}

	// 启动应用
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rt.Lifecycle.Start(ctx, rt.Container); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer rt.Lifecycle.Stop(ctx)

	// 验证
	host := core.GetFeature[*web.Host](rt)
	if host == nil {
		t.Fatal("Web Host feature not found")
	}

	addr := ""
	for i := 0; i < 20; i++ {
		addr = host.Address()
		if addr != "" && addr != ":0" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if addr == "" {
		t.Fatal("Web Host address is empty after waiting")
	}
	t.Logf("Web Host running at %s", addr)

	resp, err := http.Get(fmt.Sprintf("http://%s/ping", addr))
	if err != nil {
		t.Fatalf("HTTP Get failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Read body failed: %v", err)
	}

	expected := "pong: IntegrationTest"
	if string(body) != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, string(body))
	}
}

// TestWorker for HostedService test
type TestWorker struct {
	Started chan struct{}
	Stopped chan struct{}
	StopCh  chan struct{}
}

func (w *TestWorker) Start(ctx context.Context) error {
	close(w.Started)
	<-w.StopCh // 模拟阻塞直到 Stop 被调用
	return nil
}

func (w *TestWorker) Stop(ctx context.Context) error {
	close(w.StopCh)
	// 模拟等待清理
	time.Sleep(10 * time.Millisecond)
	close(w.Stopped)
	return nil
}

func TestHostedService(t *testing.T) {
	rt := core.NewRuntime()

	worker := &TestWorker{
		Started: make(chan struct{}),
		Stopped: make(chan struct{}),
		StopCh:  make(chan struct{}),
	}

	err := rt.Apply(
		// Register pre-initialized pointer
		core.WithHostedService(worker),
	)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if err := rt.Container.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	ctx := context.Background()
	if err := rt.Lifecycle.Start(ctx, rt.Container); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	select {
	case <-worker.Started:
	case <-time.After(100 * time.Millisecond):
		t.Error("Worker should be started")
	}

	if err := rt.Lifecycle.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	select {
	case <-worker.Stopped:
	case <-time.After(100 * time.Millisecond):
		t.Error("Worker should be stopped")
	}
}
