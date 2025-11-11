package main

import (
	"fmt"

	"github.com/gocrud/app/di"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	GetUserByID(id int) string
}

// UserRepositoryImpl 用户仓储实现
type UserRepositoryImpl struct {
	DBName string
}

func (r *UserRepositoryImpl) GetUserByID(id int) string {
	return fmt.Sprintf("User %d from %s", id, r.DBName)
}

// UserService 用户服务
type UserService struct {
	Repo UserRepository `di:""`
}

func (s *UserService) GetUser(id int) {
	user := s.Repo.GetUserByID(id)
	fmt.Println("UserService:", user)
}

// OrderService 订单服务
type OrderService struct {
	UserRepo UserRepository `di:""`
}

func (s *OrderService) CreateOrder(userID int) {
	user := s.UserRepo.GetUserByID(userID)
	fmt.Printf("Creating order for %s\n", user)
}

func main() {
	// 创建容器
	container := di.NewContainer()

	// 注册依赖
	di.BindWith[UserRepository](container, &UserRepositoryImpl{DBName: "MySQL"})
	container.Provide(&UserService{})
	container.Provide(&OrderService{})

	// 构建容器
	if err := container.Build(); err != nil {
		panic(err)
	}

	fmt.Println("=== 演示: var + Inject 模式 ===")
	fmt.Println()

	// 方式1: 注入接口
	fmt.Println("--- 1. 注入接口 ---")
	var repo UserRepository
	container.MustInject(&repo)
	fmt.Println(repo.GetUserByID(1))

	// 方式2: 注入服务（结构体指针）
	fmt.Println("\n--- 2. 注入服务 ---")
	var userSvc *UserService
	container.MustInject(&userSvc)
	userSvc.GetUser(2)

	var orderSvc *OrderService
	container.MustInject(&orderSvc)
	orderSvc.CreateOrder(3)

	// 方式3: 使用 Inject 带错误处理
	fmt.Println("\n--- 3. 带错误处理的注入 ---")
	var repo2 UserRepository
	if err := container.Inject(&repo2); err != nil {
		fmt.Println("注入失败:", err)
	} else {
		fmt.Println("注入成功:", repo2.GetUserByID(4))
	}

	// 方式4: 批量注入多个服务
	fmt.Println("\n--- 4. 批量注入 ---")
	var (
		svc1 *UserService
		svc2 *OrderService
	)
	container.MustInject(&svc1)
	container.MustInject(&svc2)

	svc1.GetUser(5)
	svc2.CreateOrder(6)
}
