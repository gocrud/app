package di

import (
	"reflect"
	"testing"
)

type TestStruct struct {
	Val string
}

func NewTestStruct(val string) *TestStruct {
	return &TestStruct{Val: val}
}

func TestConstructorInvoker(t *testing.T) {
	info := &providerInfo{
		value: NewTestStruct,
	}

	invoker := createConstructorInvoker(info)

	args := []reflect.Value{reflect.ValueOf("test")}
	res, err := invoker(args)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	ts, ok := res.(*TestStruct)
	if !ok || ts.Val != "test" {
		t.Error("Result mismatch")
	}
}

func BenchmarkInvoker(b *testing.B) {
	info := &providerInfo{
		value: NewTestStruct,
	}
	invoker := createConstructorInvoker(info)
	args := []reflect.Value{reflect.ValueOf("test")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		invoker(args)
	}
}

func BenchmarkReflectCall(b *testing.B) {
	fn := reflect.ValueOf(NewTestStruct)
	args := []reflect.Value{reflect.ValueOf("test")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fn.Call(args)
	}
}
