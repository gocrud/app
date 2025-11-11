package main

import (
	"fmt"
	"reflect"

	"github.com/gocrud/app/di"
)

// ===== 接口定义 =====

type Logger interface {
	Log(msg string)
}

type RequestContext interface {
	GetRequestID() string
	SetValue(key string, value any)
	GetValue(key string) any
}

type UserRepository interface {
	GetUserByID(id int) string
}

type UserService interface {
	GetUserProfile(id int) string
}

// ===== 实现 =====

type ConsoleLogger struct {
	instanceID int
}

var loggerInstanceCounter int

func NewConsoleLogger() Logger {
	loggerInstanceCounter++
	return &ConsoleLogger{instanceID: loggerInstanceCounter}
}

func (l *ConsoleLogger) Log(msg string) {
	fmt.Printf("[Logger #%d] %s\n", l.instanceID, msg)
}

type HttpRequestContext struct {
	requestID string
	data      map[string]any
	logger    Logger `di:""`
}

var requestContextCounter int

func NewRequestContext(logger Logger) RequestContext {
	requestContextCounter++
	return &HttpRequestContext{
		requestID: fmt.Sprintf("REQ-%d", requestContextCounter),
		data:      make(map[string]any),
		logger:    logger,
	}
}

func (ctx *HttpRequestContext) GetRequestID() string {
	return ctx.requestID
}

func (ctx *HttpRequestContext) SetValue(key string, value any) {
	ctx.data[key] = value
	ctx.logger.Log(fmt.Sprintf("[%s] Set %s", ctx.requestID, key))
}

func (ctx *HttpRequestContext) GetValue(key string) any {
	return ctx.data[key]
}

type UserRepo struct {
	ctx    RequestContext `di:""`
	logger Logger         `di:""`
}

func NewUserRepo(ctx RequestContext, logger Logger) UserRepository {
	return &UserRepo{ctx: ctx, logger: logger}
}

func (r *UserRepo) GetUserByID(id int) string {
	r.logger.Log(fmt.Sprintf("[%s] Querying user %d from database", r.ctx.GetRequestID(), id))
	return fmt.Sprintf("User-%d", id)
}

type UserSvc struct {
	repo   UserRepository `di:""`
	ctx    RequestContext `di:""`
	logger Logger         `di:""`
}

func NewUserService(repo UserRepository, ctx RequestContext, logger Logger) UserService {
	return &UserSvc{repo: repo, ctx: ctx, logger: logger}
}

func (s *UserSvc) GetUserProfile(id int) string {
	s.logger.Log(fmt.Sprintf("[%s] Getting user profile for %d", s.ctx.GetRequestID(), id))
	userName := s.repo.GetUserByID(id)
	s.ctx.SetValue("lastUser", userName)
	return fmt.Sprintf("Profile of %s", userName)
}

// ===== 主程序 =====

func main() {
	fmt.Println("=== DI Container Scope Demo ===")
	fmt.Println()

	// 创建容器
	container := di.NewContainer()

	// 1. 注册 Logger 为 Singleton（全局共享）
	container.ProvideType(di.TypeProvider{
		Provide: reflect.TypeOf((*Logger)(nil)).Elem(),
		UseType: NewConsoleLogger,
		Options: di.ProviderOptions{Scope: di.ScopeSingleton},
	})

	// 2. 注册 RequestContext 为 Scoped（每个请求一个）
	container.ProvideType(di.TypeProvider{
		Provide: reflect.TypeOf((*RequestContext)(nil)).Elem(),
		UseType: NewRequestContext,
		Options: di.ProviderOptions{Scope: di.ScopeScoped},
	})

	// 3. 注册 UserRepository 为 Scoped（每个请求一个）
	container.ProvideType(di.TypeProvider{
		Provide: reflect.TypeOf((*UserRepository)(nil)).Elem(),
		UseType: NewUserRepo,
		Options: di.ProviderOptions{Scope: di.ScopeScoped},
	})

	// 4. 注册 UserService 为 Transient（每次都新建）
	container.ProvideType(di.TypeProvider{
		Provide: reflect.TypeOf((*UserService)(nil)).Elem(),
		UseType: NewUserService,
		Options: di.ProviderOptions{Scope: di.ScopeTransient},
	})

	// 构建容器
	if err := container.Build(); err != nil {
		panic(err)
	}

	fmt.Println("Container built successfully!")
	fmt.Println()

	// 模拟处理 HTTP 请求的函数
	handleRequest := func(requestNum int) {
		fmt.Printf("\n--- Handling Request #%d ---\n", requestNum)

		// 为每个请求创建作用域
		scope := container.CreateScope()
		defer scope.Dispose()

		// 设置当前作用域
		container.SetCurrentScope(scope)
		defer container.ClearCurrentScope()

		// 获取两次 UserService（Transient，应该是不同实例）
		service1, _ := container.GetByType(reflect.TypeOf((*UserService)(nil)).Elem())
		service2, _ := container.GetByType(reflect.TypeOf((*UserService)(nil)).Elem())

		userService1 := service1.(UserService)
		userService2 := service2.(UserService)

		// 使用 service
		profile1 := userService1.GetUserProfile(100 + requestNum)
		fmt.Printf("Result: %s\n", profile1)

		profile2 := userService2.GetUserProfile(200 + requestNum)
		fmt.Printf("Result: %s\n", profile2)

		// 验证：同一作用域内，RequestContext 应该是同一个
		ctx, _ := scope.GetByType(reflect.TypeOf((*RequestContext)(nil)).Elem())
		requestContext := ctx.(RequestContext)
		fmt.Printf("Request ID: %s\n", requestContext.GetRequestID())
		fmt.Printf("Last User: %v\n", requestContext.GetValue("lastUser"))
	}

	// 处理3个请求
	handleRequest(1)
	handleRequest(2)
	handleRequest(3)

	fmt.Println("\n=== Summary ===")
	fmt.Printf("Logger instances created: %d (Expected: 1, because it's Singleton)\n", loggerInstanceCounter)
	fmt.Printf("RequestContext instances created: %d (Expected: 3, one per request scope)\n", requestContextCounter)
}
