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

	di.ProvideService[*Database](c, di.WithName("master"), di.WithValue(&Database{DSN: "master_dsn"}))
	di.ProvideService[*Database](c, di.WithName("slave"), di.WithValue(&Database{DSN: "slave_dsn"}))
	di.ProvideService[*ServiceWithNamedDB](c)

	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	svc, err := di.Get[*ServiceWithNamedDB](c)
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

	di.ProvideService[*Database](c, di.WithName("master"), di.WithValue(&Database{DSN: "master_dsn"}))
	di.ProvideService[*ServiceWithOptional](c)

	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	svc, err := di.Get[*ServiceWithOptional](c)
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
	di.ProvideService[*ServiceWithSimpleOptional](c)
	di.ProvideService[*ServiceWithCommaOptional](c)
	di.ProvideService[*ServiceWithOptionalAlternative](c)

	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Test di:"?"
	svc1, err := di.Get[*ServiceWithSimpleOptional](c)
	if err != nil {
		t.Fatalf("Resolve simple optional failed: %v", err)
	}
	if svc1.Optional != nil {
		t.Error("Optional field should be nil for di:\"?\"")
	}

	// Test di:",?"
	svc2, err := di.Get[*ServiceWithCommaOptional](c)
	if err != nil {
		t.Fatalf("Resolve comma optional failed: %v", err)
	}
	if svc2.Optional != nil {
		t.Error("Optional field should be nil for di:\",?\"")
	}

	// Test di:"optional"
	svc3, err := di.Get[*ServiceWithOptionalAlternative](c)
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
	di.ProvideService[*Database](c, di.WithValue(db))
	di.ProvideService[*ServiceWithSimpleOptional](c)

	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	svc, err := di.Get[*ServiceWithSimpleOptional](c)
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
	di.ProvideService[*Database](c, di.WithName("db1"), di.WithValue(&Database{DSN: "db1"}))

	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	db1, err := di.GetNamed[*Database](c, "db1")
	if err != nil {
		t.Fatalf("ResolveNamed failed: %v", err)
	}
	if db1.DSN != "db1" {
		t.Errorf("Expected db1, got %s", db1.DSN)
	}

	_, err = di.GetNamed[*Database](c, "missing")
	if err == nil {
		t.Error("Expected error for missing named service")
	}
}
