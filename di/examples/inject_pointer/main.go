package main

import (
	"fmt"

	"github.com/gocrud/app/di"
)

type Service struct {
	Name string
}

func main() {
	c := di.NewContainer()

	// Register a pointer type
	di.Register[*Service](c, di.WithFactory(func() *Service {
		return &Service{Name: "Pointer Service"}
	}))

	c.Build()

	// Resolve the pointer
	svc, err := di.Resolve[*Service](c)
	if err != nil {
		panic(err)
	}

	fmt.Println("Resolved:", svc.Name)
}
