package di_test

import (
	"testing"

	"github.com/gocrud/app/di"
)

type ServiceA struct {
	Val int
}

type ServiceB struct {
	A *ServiceA `di:""`
}

type InterfaceC interface {
	Do() string
}

type ServiceC struct{}

func (s *ServiceC) Do() string { return "C" }

func TestDI(t *testing.T) {
	c := di.NewContainer()

	// Register Value
	di.Register[int](c, di.WithValue(100))

	// Register Singleton
	di.Register[*ServiceA](c, di.WithFactory(func(val int) *ServiceA {
		return &ServiceA{Val: val}
	}))

	// Register Transient struct with field injection
	di.Register[*ServiceB](c, di.WithTransient())

	// Register Interface
	di.Register[InterfaceC](c, di.Use[*ServiceC]())

	err := c.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Resolve
	b, err := di.Resolve[*ServiceB](c)
	if err != nil {
		t.Fatalf("Resolve ServiceB failed: %v", err)
	}
	if b == nil {
		t.Fatal("Resolved nil ServiceB")
	}
	if b.A == nil {
		t.Fatal("Field injection failed: b.A is nil")
	}
	if b.A.Val != 100 {
		t.Errorf("Expected 100, got %d", b.A.Val)
	}

	// Resolve Interface
	iface, err := di.Resolve[InterfaceC](c)
	if err != nil {
		t.Fatalf("Resolve InterfaceC failed: %v", err)
	}
	if iface.Do() != "C" {
		t.Errorf("Expected 'C', got '%s'", iface.Do())
	}
}

func TestScope(t *testing.T) {
	c := di.NewContainer()

	type ScopedService struct {
		ID int
	}

	counter := 0
	di.Register[*ScopedService](c, di.WithScoped(), di.WithFactory(func() *ScopedService {
		counter++
		return &ScopedService{ID: counter}
	}))

	c.Build()

	scope1 := c.CreateScope()
	s1a, _ := di.Resolve[*ScopedService](scope1)
	s1b, _ := di.Resolve[*ScopedService](scope1)

	if s1a.ID != s1b.ID {
		t.Errorf("Expected same instance in scope 1, got IDs %d and %d", s1a.ID, s1b.ID)
	}
	if s1a.ID != 1 {
		t.Errorf("Expected ID 1, got %d", s1a.ID)
	}

	scope2 := c.CreateScope()
	s2a, _ := di.Resolve[*ScopedService](scope2)
	if s2a.ID != 2 {
		t.Errorf("Expected ID 2, got %d", s2a.ID)
	}
	if s1a.ID == s2a.ID {
		t.Error("Expected different instances across scopes")
	}
}
