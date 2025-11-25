package main

import (
	"fmt"
	"time"

	"github.com/gocrud/app/di"
)

type RequestContext struct {
	ID        string
	Timestamp time.Time
}

type RequestHandler struct {
	Ctx *RequestContext `di:""`
}

func main() {
	c := di.NewContainer()

	// Register RequestContext as Scoped
	// It will be created once per scope
	di.Register[*RequestContext](c, di.WithScoped(), di.WithFactory(func() *RequestContext {
		return &RequestContext{
			ID:        fmt.Sprintf("REQ-%d", time.Now().UnixNano()),
			Timestamp: time.Now(),
		}
	}))

	// Register Handler as Transient (or Singleton, but it depends on Scoped RequestContext)
	// If Handler is Singleton, it CANNOT depend on Scoped service (this should be a validation error or runtime error).
	// So Handler must be Transient or Scoped.
	di.Register[*RequestHandler](c, di.WithTransient())

	c.Build()

	// Simulate Request 1
	fmt.Println("--- Request 1 ---")
	scope1 := c.CreateScope()
	handleRequest(scope1)
	scope1.Dispose()

	// Simulate Request 2
	fmt.Println("\n--- Request 2 ---")
	scope2 := c.CreateScope()
	handleRequest(scope2)
	scope2.Dispose()
}

func handleRequest(scope di.Scope) {
	// In a real web framework, this would be done in middleware
	
	// Resolve Handler from Scope
	handler, err := di.Resolve[*RequestHandler](scope)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Handling request %s at %s\n", handler.Ctx.ID, handler.Ctx.Timestamp.Format(time.RFC3339))

	// Verify that resolving again in the same scope returns the same Context
	ctx2, _ := di.Resolve[*RequestContext](scope)
	if ctx2 != handler.Ctx {
		fmt.Println("Error: Scoped instances do not match!")
	} else {
		fmt.Println("Scoped instances match within the same scope.")
	}
}
