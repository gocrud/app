package di_test

import (
	"testing"

	"github.com/gocrud/app/di"
)

type Database struct {
	DSN string
}

type ServiceWithNamedDB struct {
	Master *Database `di:"master"`
	Slave  *Database `di:"slave"`
}

type ServiceWithOptional struct {
	Required *Database `di:"master"`
	Optional *Database `di:"missing,?"`
}

type ServiceWithSimpleOptional struct {
	Optional *Database `di:"?"`
}

type ServiceWithCommaOptional struct {
	Optional *Database `di:",?"`
}

type ServiceWithOptionalAlternative struct {
	Optional *Database `di:"optional"`
}

func TestNamedInjection(t *testing.T) {
	c := di.NewContainer()

	di.Register[*Database](c, di.WithName("master"), di.WithValue(&Database{DSN: "master_dsn"}))
	di.Register[*Database](c, di.WithName("slave"), di.WithValue(&Database{DSN: "slave_dsn"}))
	di.Register[*ServiceWithNamedDB](c)

	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	svc, err := di.Resolve[*ServiceWithNamedDB](c)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if svc.Master.DSN != "master_dsn" {
		t.Errorf("Expected master DSN, got %s", svc.Master.DSN)
	}
	if svc.Slave.DSN != "slave_dsn" {
		t.Errorf("Expected slave DSN, got %s", svc.Slave.DSN)
	}
}

func TestOptionalInjection(t *testing.T) {
	c := di.NewContainer()

	di.Register[*Database](c, di.WithName("master"), di.WithValue(&Database{DSN: "master_dsn"}))
	di.Register[*ServiceWithOptional](c)

	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	svc, err := di.Resolve[*ServiceWithOptional](c)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if svc.Required == nil {
		t.Error("Required field is nil")
	}
	if svc.Optional != nil {
		t.Error("Optional field should be nil")
	}
}

func TestSimpleOptionalInjection(t *testing.T) {
	c := di.NewContainer()
	di.Register[*ServiceWithSimpleOptional](c)
	di.Register[*ServiceWithCommaOptional](c)
	di.Register[*ServiceWithOptionalAlternative](c)

	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Test di:"?"
	svc1, err := di.Resolve[*ServiceWithSimpleOptional](c)
	if err != nil {
		t.Fatalf("Resolve simple optional failed: %v", err)
	}
	if svc1.Optional != nil {
		t.Error("Optional field should be nil for di:\"?\"")
	}

	// Test di:",?"
	svc2, err := di.Resolve[*ServiceWithCommaOptional](c)
	if err != nil {
		t.Fatalf("Resolve comma optional failed: %v", err)
	}
	if svc2.Optional != nil {
		t.Error("Optional field should be nil for di:\",?\"")
	}

	// Test di:"optional"
	svc3, err := di.Resolve[*ServiceWithOptionalAlternative](c)
	if err != nil {
		t.Fatalf("Resolve keyword optional failed: %v", err)
	}
	if svc3.Optional != nil {
		t.Error("Optional field should be nil for di:\"optional\"")
	}
}

func TestSimpleOptionalInjection_WithRegisteredService(t *testing.T) {
	// 测试当可选服务实际存在时，是否能正确注入
	c := di.NewContainer()
	db := &Database{DSN: "default"}
	di.Register[*Database](c, di.WithValue(db))
	di.Register[*ServiceWithSimpleOptional](c)

	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	svc, err := di.Resolve[*ServiceWithSimpleOptional](c)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if svc.Optional == nil {
		t.Error("Optional field should NOT be nil when service exists")
	}
	if svc.Optional != db {
		t.Error("Optional field injected wrong instance")
	}
}

func TestNamedResolve(t *testing.T) {
	c := di.NewContainer()
	di.Register[*Database](c, di.WithName("db1"), di.WithValue(&Database{DSN: "db1"}))

	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	db1, err := di.ResolveNamed[*Database](c, "db1")
	if err != nil {
		t.Fatalf("ResolveNamed failed: %v", err)
	}
	if db1.DSN != "db1" {
		t.Errorf("Expected db1, got %s", db1.DSN)
	}

	_, err = di.ResolveNamed[*Database](c, "missing")
	if err == nil {
		t.Error("Expected error for missing named service")
	}
}
