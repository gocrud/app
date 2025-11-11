package di_test

import (
	"testing"

	"github.com/gocrud/app/di"
)

// 基准测试接口和实现
type BenchLogger interface {
	Log(msg string)
}

type BenchConsoleLogger struct{}

func (l *BenchConsoleLogger) Log(msg string) {}

type BenchDatabase interface {
	Query(sql string) error
}

type BenchMySQLDB struct{}

func (db *BenchMySQLDB) Query(sql string) error { return nil }

type BenchCache interface {
	Get(key string) string
	Set(key, value string)
}

type BenchRedisCache struct{}

func (c *BenchRedisCache) Get(key string) string { return "" }
func (c *BenchRedisCache) Set(key, value string) {}

// 简单服务
type BenchSimpleService struct {
	Logger BenchLogger `di:""`
}

// 中等复杂服务
type BenchMediumService struct {
	Logger   BenchLogger   `di:""`
	Database BenchDatabase `di:""`
	Cache    BenchCache    `di:""`
}

// 复杂服务（多层依赖）
type BenchRepository struct {
	Database BenchDatabase `di:""`
	Cache    BenchCache    `di:""`
	Logger   BenchLogger   `di:""`
}

type BenchBusinessService struct {
	Repo   *BenchRepository `di:""`
	Logger BenchLogger      `di:""`
}

type BenchAPIService struct {
	Business *BenchBusinessService `di:""`
	Logger   BenchLogger           `di:""`
	Cache    BenchCache            `di:""`
}

// Benchmark 1: 容器构建性能
func BenchmarkBuild_Simple(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		di.Reset()
		di.Bind[BenchLogger](&BenchConsoleLogger{})
		di.Provide(&BenchSimpleService{})
		di.MustBuild()
	}
}

func BenchmarkBuild_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		di.Reset()
		di.Bind[BenchLogger](&BenchConsoleLogger{})
		di.Bind[BenchDatabase](&BenchMySQLDB{})
		di.Bind[BenchCache](&BenchRedisCache{})
		di.Provide(&BenchMediumService{})
		di.MustBuild()
	}
}

func BenchmarkBuild_Complex(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		di.Reset()
		di.Bind[BenchLogger](&BenchConsoleLogger{})
		di.Bind[BenchDatabase](&BenchMySQLDB{})
		di.Bind[BenchCache](&BenchRedisCache{})
		di.Provide(&BenchRepository{})
		di.Provide(&BenchBusinessService{})
		di.Provide(&BenchAPIService{})
		di.MustBuild()
	}
}

// Benchmark 2: 注入性能（Build 后）
func BenchmarkInject_Simple(b *testing.B) {
	di.Reset()
	di.Bind[BenchLogger](&BenchConsoleLogger{})
	di.Provide(&BenchSimpleService{})
	di.MustBuild()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = di.Inject[*BenchSimpleService]()
	}
}

func BenchmarkInject_Medium(b *testing.B) {
	di.Reset()
	di.Bind[BenchLogger](&BenchConsoleLogger{})
	di.Bind[BenchDatabase](&BenchMySQLDB{})
	di.Bind[BenchCache](&BenchRedisCache{})
	di.Provide(&BenchMediumService{})
	di.MustBuild()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = di.Inject[*BenchMediumService]()
	}
}

func BenchmarkInject_Complex(b *testing.B) {
	di.Reset()
	di.Bind[BenchLogger](&BenchConsoleLogger{})
	di.Bind[BenchDatabase](&BenchMySQLDB{})
	di.Bind[BenchCache](&BenchRedisCache{})
	di.Provide(&BenchRepository{})
	di.Provide(&BenchBusinessService{})
	di.Provide(&BenchAPIService{})
	di.MustBuild()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = di.Inject[*BenchAPIService]()
	}
}

// Benchmark 3: 不同注册方式的性能对比
func BenchmarkProvide_Bind(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		di.Reset()
		di.Bind[BenchLogger](&BenchConsoleLogger{})
		di.MustBuild()
	}
}

func BenchmarkProvide_ProvideType(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		di.Reset()
		di.ProvideType(di.TypeProvider{
			Provide: di.TypeOf[BenchLogger](),
			UseType: &BenchConsoleLogger{},
		})
		di.MustBuild()
	}
}

func BenchmarkProvide_Direct(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		di.Reset()
		di.Bind[BenchLogger](&BenchConsoleLogger{})
		di.Provide(&BenchSimpleService{})
		di.MustBuild()
	}
}

// Benchmark 4: 构造函数 vs 实例
func BenchmarkProvide_Instance(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		di.Reset()
		di.Bind[BenchLogger](&BenchConsoleLogger{})
		di.MustBuild()
	}
}

func BenchmarkProvide_Constructor(b *testing.B) {
	constructor := func() *BenchConsoleLogger {
		return &BenchConsoleLogger{}
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		di.Reset()
		di.ProvideType(di.TypeProvider{
			Provide: di.TypeOf[BenchLogger](),
			UseType: constructor,
		})
		di.MustBuild()
	}
}

// Benchmark 5: 大规模注册性能
func BenchmarkBuild_LargeScale(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		di.Reset()

		// 注册基础服务
		di.Bind[BenchLogger](&BenchConsoleLogger{})
		di.Bind[BenchDatabase](&BenchMySQLDB{})
		di.Bind[BenchCache](&BenchRedisCache{})

		// 注册多个服务实例（同一类型只能注册一次）
		di.Provide(&BenchRepository{})
		di.Provide(&BenchBusinessService{})
		di.Provide(&BenchAPIService{})
		di.Provide(&BenchSimpleService{})
		di.Provide(&BenchMediumService{})

		di.MustBuild()
	}
}

// Benchmark 6: 并发注入性能
func BenchmarkInject_Concurrent(b *testing.B) {
	di.Reset()
	di.Bind[BenchLogger](&BenchConsoleLogger{})
	di.Bind[BenchDatabase](&BenchMySQLDB{})
	di.Bind[BenchCache](&BenchRedisCache{})
	di.Provide(&BenchMediumService{})
	di.MustBuild()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = di.Inject[*BenchMediumService]()
		}
	})
}

// Benchmark 7: 对比手动创建的性能
func BenchmarkManual_Simple(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger := &BenchConsoleLogger{}
		_ = &BenchSimpleService{Logger: logger}
	}
}

func BenchmarkManual_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger := &BenchConsoleLogger{}
		db := &BenchMySQLDB{}
		cache := &BenchRedisCache{}
		_ = &BenchMediumService{
			Logger:   logger,
			Database: db,
			Cache:    cache,
		}
	}
}

func BenchmarkManual_Complex(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger := &BenchConsoleLogger{}
		db := &BenchMySQLDB{}
		cache := &BenchRedisCache{}
		repo := &BenchRepository{
			Database: db,
			Cache:    cache,
			Logger:   logger,
		}
		business := &BenchBusinessService{
			Repo:   repo,
			Logger: logger,
		}
		_ = &BenchAPIService{
			Business: business,
			Logger:   logger,
			Cache:    cache,
		}
	}
}
