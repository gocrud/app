package main

import "github.com/gocrud/app/di"

type ConcreteService struct {
	Name string
}

func main() {
	di.Reset()

	// 尝试用 TypeOf 绑定非接口类型，应该会 panic
	di.ProvideType(di.TypeProvider{
		Provide: di.TypeOf[*ConcreteService](), // 这是指针类型，不是接口
		UseType: &ConcreteService{Name: "test"},
	})

	di.MustBuild()
}
