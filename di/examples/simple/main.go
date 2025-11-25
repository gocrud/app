package main

import (
	"fmt"

	"github.com/gocrud/app/di"
)

// 1. Define Interfaces
type Greeter interface {
	Greet(name string) string
}

// 2. Implement Services
type EnglishGreeter struct {
	Prefix string
}

func (g *EnglishGreeter) Greet(name string) string {
	return fmt.Sprintf("%s %s!", g.Prefix, name)
}

type UserController struct {
	Greeter Greeter `di:""` // Auto injection
}

func (c *UserController) SayHi() {
	fmt.Println(c.Greeter.Greet("Developer"))
}

func main() {
	// 1. Create Container
	c := di.NewContainer()

	// 2. Register Services

	// Register a value
	di.Register[string](c, di.WithValue("Hello"))

	// Register implementation (via factory to consume string value)
	di.Register[*EnglishGreeter](c, di.WithFactory(func(prefix string) *EnglishGreeter {
		return &EnglishGreeter{Prefix: prefix}
	}))

	// Bind Interface to Implementation
	// We use a factory to resolve the existing *EnglishGreeter singleton
	// instead of creating a new empty instance via di.Use[*EnglishGreeter]().
	di.Register[Greeter](c, di.WithFactory(func(g *EnglishGreeter) Greeter {
		return g
	}))

	// Register Controller (Transient)
	di.Register[*UserController](c, di.WithTransient())

	// 3. Build
	if err := c.Build(); err != nil {
		panic(err)
	}

	// 4. Resolve & Use
	controller, err := di.Resolve[*UserController](c)
	if err != nil {
		panic(err)
	}

	controller.SayHi() // Output: Hello Developer!
}
