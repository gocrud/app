package database_test

import (
	"testing"

	"github.com/gocrud/app/config"
	"github.com/gocrud/app/configure/database"
	"github.com/gocrud/app/core"
	"github.com/gocrud/app/di"
	"github.com/gocrud/app/logging"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name string
}

type MockDBService struct {
	Master *gorm.DB `di:"master"`
	Slave  *gorm.DB `di:"slave,?"`
}

// DBConfig 模拟用户定义的配置结构
type DBConfig struct {
	DSN          string `json:"dsn"`
	MaxOpenConns int    `json:"max_open_conns"`
}

func TestDatabaseConfiguration(t *testing.T) {
	builder := core.NewApplicationBuilder()

	// 1. 配置内存配置源
	builder.ConfigureConfiguration(func(cb *config.ConfigurationBuilder) {
		cb.AddInMemory(map[string]any{
			"db": map[string]any{
				"master": map[string]any{
					"dsn":            "file::memory:?cache=shared",
					"max_open_conns": 5,
				},
			},
		})
	})

	// 2. 配置 Database (演示 config.Load 的使用)
	configurator := database.Configure(func(b *database.Builder) {
		// 使用 config.Load 从 Context 获取强类型配置
		dbConf, err := config.Load[DBConfig](b.ConfigContext().GetConfiguration(), "db.master")
		if err != nil {
			// 在实际应用中可能 panic 或记录错误，这里简化处理
			b.Add("config_error", nil, nil) // 触发 builder 错误
			return
		}

		b.Add("master", sqlite.Open(dbConf.DSN), func(o *database.DatabaseOptions) {
			o.MaxOpenConns = dbConf.MaxOpenConns
			o.AutoMigrate = []any{&User{}}
		})
	})

	builder.Configure(func(ctx *core.BuildContext) {
		configurator(ctx)
	})

	// Register Mock Service
	builder.Configure(func(ctx *core.BuildContext) {
		di.Register[*MockDBService](ctx.Container())
	})

	app := builder.Build()

	// Resolve Service
	var svc *MockDBService
	app.GetService(&svc)

	if svc.Master == nil {
		t.Fatal("Master DB should not be nil")
	}

	// Verify config was applied
	sqlDB, _ := svc.Master.DB()
	stats := sqlDB.Stats()
	if stats.MaxOpenConnections != 5 {
		t.Errorf("Expected MaxOpenConns 5, got %d", stats.MaxOpenConnections)
	}

	// Test DB interaction
	if err := svc.Master.Create(&User{Name: "test"}).Error; err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}
}

func TestDatabaseBuilder_Errors(t *testing.T) {
	logger := logging.NewLogger()
	// 手动构造 BuildContext 比较麻烦，这里简单测试 Builder 逻辑
	// 由于 NewBuilder 需要 ctx，我们可以传 nil (如果只是测试基本 Add 逻辑且不依赖 Context 的话)
	// 但我们的 Add 方法不依赖 ctx，除了我们刚才加的 config.Load 是在外部调用的。
	builder := database.NewBuilder(nil)

	// Missing dialector
	builder.Add("invalid", nil, nil)

	// Duplicate
	builder.Add("dup", sqlite.Open("a"), nil)
	builder.Add("dup", sqlite.Open("b"), nil)

	_, err := builder.Build(logger)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	t.Logf("Got expected error: %v", err)
}
