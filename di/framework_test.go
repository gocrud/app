package di_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gocrud/app/di"
)

// 定义 Token
var (
	DbHostToken = di.NewToken[string]("db.host")
	DbPortToken = di.NewToken[int]("db.port")
	ApiKeyToken = di.NewToken[string]("api.key")
)

// 测试类型
type Logger interface {
	Log(msg string)
}

type ConsoleLogger struct {
	Prefix string
}

func (l *ConsoleLogger) Log(msg string) {
	fmt.Printf("[%s] %s\n", l.Prefix, msg)
}

type FileLogger struct {
	Path string
}

func (l *FileLogger) Log(msg string) {
	fmt.Printf("[File:%s] %s\n", l.Path, msg)
}

type Config struct {
	IsAuthorized bool
}

type Database struct {
	Host string
	Port int
}

type UserRepository struct {
	DB     *Database `di:""`
	Logger Logger    `di:""`
}

type UserService struct {
	Repo   *UserRepository
	Logger Logger
	Config *Config
}

func NewUserService(repo *UserRepository, logger Logger, cfg *Config) *UserService {
	return &UserService{
		Repo:   repo,
		Logger: logger,
		Config: cfg,
	}
}

// Test 1: 基本功能 - Provide
func TestProvideBasic(t *testing.T) {
	di.Reset()

	di.Provide(&Database{Host: "localhost", Port: 5432})
	di.Bind[Logger](&ConsoleLogger{Prefix: "TEST"})
	di.Provide(&UserRepository{})

	if err := di.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	repo := di.Inject[*UserRepository]()
	if repo == nil {
		t.Fatal("Expected UserRepository to be injected")
	}
	if repo.DB == nil {
		t.Fatal("Expected Database to be injected into UserRepository")
	}
	if repo.Logger == nil {
		t.Fatal("Expected Logger to be injected into UserRepository")
	}
}

// Test 2: UseValue - 注册静态值
func TestProvideValue(t *testing.T) {
	di.Reset()

	// 使用 Token 注册基本类型
	di.ProvideValue(di.ValueProvider{Provide: DbHostToken, Value: "localhost"})
	di.ProvideValue(di.ValueProvider{Provide: DbPortToken, Value: 5432})
	di.ProvideValue(di.ValueProvider{Provide: ApiKeyToken, Value: "secret-key"})

	// 注册配置对象
	di.Provide(&Config{IsAuthorized: true})

	if err := di.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 使用 Token 注入
	dbHost := di.Inject[string](DbHostToken)
	if dbHost != "localhost" {
		t.Errorf("Expected 'localhost', got '%s'", dbHost)
	}

	dbPort := di.Inject[int](DbPortToken)
	if dbPort != 5432 {
		t.Errorf("Expected 5432, got %d", dbPort)
	}

	apiKey := di.Inject[string](ApiKeyToken)
	if apiKey != "secret-key" {
		t.Errorf("Expected 'secret-key', got '%s'", apiKey)
	}

	config := di.Inject[*Config]()
	if !config.IsAuthorized {
		t.Error("Expected IsAuthorized to be true")
	}
}

// Test 3: ProvideType - 接口绑定
func TestProvideType(t *testing.T) {
	di.Reset()

	// 使用 Bind 语法糖绑定接口
	di.Bind[Logger](&FileLogger{Path: "/tmp/app.log"})

	if err := di.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	logger := di.Inject[Logger]()
	if logger == nil {
		t.Fatal("Expected Logger to be injected")
	}

	// 应该获得 FileLogger
	if _, ok := logger.(*FileLogger); !ok {
		t.Error("Expected FileLogger")
	}
}

// Test 4: UseFactory - 工厂函数（自动推断依赖）
func TestProvideFactoryAutoDeps(t *testing.T) {
	di.Reset()

	// 注册依赖
	di.ProvideValue(di.ValueProvider{Provide: DbHostToken, Value: "localhost"})
	di.ProvideValue(di.ValueProvider{Provide: DbPortToken, Value: 3306})
	di.Bind[Logger](&ConsoleLogger{Prefix: "APP"})

	// 使用工厂创建 Database（自动推断依赖：logger, host, port）
	di.ProvideFactory(di.FactoryProvider{
		Provide: di.TypeOf[*Database](),
		Factory: func(logger Logger, host string, port int) *Database {
			logger.Log("Creating database connection")
			return &Database{Host: host, Port: port}
		},
		Deps: []any{
			di.TypeOf[Logger](),
			DbHostToken,
			DbPortToken,
		},
	})

	if err := di.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	db := di.Inject[*Database]()
	if db.Host != "localhost" || db.Port != 3306 {
		t.Errorf("Expected localhost:3306, got %s:%d", db.Host, db.Port)
	}
}

// Test 5: UseFactory - 工厂函数（显式指定依赖）
func TestProvideFactoryExplicitDeps(t *testing.T) {
	di.Reset()

	di.Provide(&Config{IsAuthorized: true})
	di.Provide(&Database{Host: "localhost", Port: 5432})
	di.Bind[Logger](&ConsoleLogger{Prefix: "REPO"})
	di.Provide(&UserRepository{})

	// 使用工厂创建服务（显式指定依赖）
	di.ProvideFactory(di.FactoryProvider{
		Provide: di.TypeOf[*UserService](),
		Factory: func(repo *UserRepository, logger Logger, cfg *Config) *UserService {
			if cfg.IsAuthorized {
				logger.Log("Creating authorized user service")
			}
			return NewUserService(repo, logger, cfg)
		},
		Deps: []any{
			di.TypeOf[*UserRepository](),
			di.TypeOf[Logger](),
			di.TypeOf[*Config](),
		},
	})

	if err := di.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	service := di.Inject[*UserService]()
	if service == nil {
		t.Fatal("Expected UserService to be injected")
	}
	if service.Repo == nil {
		t.Fatal("Expected Repo to be injected")
	}
	if service.Logger == nil {
		t.Fatal("Expected Logger to be injected")
	}
	if !service.Config.IsAuthorized {
		t.Error("Expected IsAuthorized to be true")
	}
}

// Test 6: UseExisting - 别名
func TestProvideExisting(t *testing.T) {
	di.Reset()

	di.Provide(&ConsoleLogger{Prefix: "MAIN"})

	// 创建别名
	di.BindTo[Logger](di.TypeOf[*ConsoleLogger]())

	if err := di.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	logger1 := di.Inject[*ConsoleLogger]()
	logger2 := di.Inject[Logger]()

	// 应该是同一个实例
	if logger1 != logger2.(*ConsoleLogger) {
		t.Error("Expected same instance")
	}
}

// Test 7: 字段注入
func TestFieldInjection(t *testing.T) {
	di.Reset()

	di.Provide(&Database{Host: "localhost", Port: 5432})
	di.Bind[Logger](&ConsoleLogger{Prefix: "APP"})
	di.Provide(&UserRepository{}) // 字段会自动注入

	if err := di.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	repo := di.Inject[*UserRepository]()
	if repo.DB == nil {
		t.Fatal("Expected DB to be injected")
	}
	if repo.Logger == nil {
		t.Fatal("Expected Logger to be injected")
	}
	if repo.DB.Host != "localhost" {
		t.Errorf("Expected localhost, got %s", repo.DB.Host)
	}
}

// Test 8: 错误处理 - 构造函数返回 error
func TestConstructorError(t *testing.T) {
	di.Reset()

	di.Provide(func() (*UserService, error) {
		return nil, fmt.Errorf("construction failed")
	})

	err := di.Build()
	if err == nil {
		t.Fatal("Expected Build to fail")
	}
	// 验证错误包含关键信息
	errMsg := err.Error()
	if !strings.Contains(errMsg, "UserService") || !strings.Contains(errMsg, "construction failed") {
		t.Errorf("Unexpected error: %v", err)
	}
}

// Test 9: TryInject
func TestTryInject(t *testing.T) {
	di.Reset()

	di.Provide(&ConsoleLogger{Prefix: "TEST"})
	di.MustBuild()

	logger, err := di.TryInject[*ConsoleLogger]()
	if err != nil {
		t.Fatalf("TryInject failed: %v", err)
	}
	if logger == nil {
		t.Fatal("Expected logger to be non-nil")
	}

	_, err = di.TryInject[*FileLogger]()
	if err == nil {
		t.Fatal("Expected error for non-existent type")
	}
}
