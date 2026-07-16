package api

import (
	"testing"
)

// TestSchemeNew tests creating a new scheme.
func TestSchemeNew(t *testing.T) {
	scheme := NewScheme()
	if scheme == nil {
		t.Fatal("NewScheme returned nil")
	}
}

// TestSchemeRegister tests registering a type factory.
func TestSchemeRegister(t *testing.T) {
	scheme := NewScheme()

	factory := func() any {
		return map[string]interface{}{"id": "test"}
	}

	err := scheme.Register("TestType", factory)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
}

// TestSchemeDuplicateRegister tests registering the same type twice.
func TestSchemeDuplicateRegister(t *testing.T) {
	scheme := NewScheme()

	factory1 := func() any { return map[string]interface{}{} }
	factory2 := func() any { return map[string]interface{}{} }

	if err := scheme.Register("TestType", factory1); err != nil {
		t.Fatalf("First register failed: %v", err)
	}

	err := scheme.Register("TestType", factory2)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

// TestSchemeCreateObject tests creating an object from a registered type.
func TestSchemeCreateObject(t *testing.T) {
	scheme := NewScheme()

	factory := func() any {
		return map[string]interface{}{"id": "created"}
	}

	scheme.Register("TestType", factory)

	obj, err := scheme.New("TestType")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if obj == nil {
		t.Fatal("New returned nil")
	}

	m, ok := obj.(map[string]interface{})
	if !ok {
		t.Errorf("Object is %T, expected map", obj)
	}

	if m["id"] != "created" {
		t.Errorf("Object id is %v, expected 'created'", m["id"])
	}
}

// TestSchemeUnregisteredType tests creating from unregistered type.
func TestSchemeUnregisteredType(t *testing.T) {
	scheme := NewScheme()

	_, err := scheme.New("NonExistent")
	if err == nil {
		t.Error("Expected error for unregistered type")
	}
}

// TestSchemeRegisterMultipleTypes tests registering multiple types.
func TestSchemeRegisterMultipleTypes(t *testing.T) {
	scheme := NewScheme()

	types := []string{"Type1", "Type2", "Type3"}
	factories := make([]func() any, len(types))
	for i := 0; i < len(types); i++ {
		idx := i
		factories[i] = func() any {
			return map[string]interface{}{"index": idx}
		}
	}

	for i, typeName := range types {
		if err := scheme.Register(typeName, factories[i]); err != nil {
			t.Fatalf("Register %s failed: %v", typeName, err)
		}
	}

	for _, typeName := range types {
		obj, err := scheme.New(typeName)
		if err != nil {
			t.Fatalf("New %s failed: %v", typeName, err)
		}

		m, _ := obj.(map[string]interface{})
		if m == nil {
			t.Errorf("New %s returned non-map", typeName)
		}
	}
}

// TestSchemeFactoryReturnTypes tests factories that return different types.
func TestSchemeFactoryReturnTypes(t *testing.T) {
	scheme := NewScheme()

	// String factory
	scheme.Register("StringType", func() any {
		return "string value"
	})

	// Integer factory
	scheme.Register("IntType", func() any {
		return 42
	})

	// Struct factory
	type TestStruct struct {
		Value string
	}
	scheme.Register("StructType", func() any {
		return &TestStruct{Value: "test"}
	})

	// Verify each factory returns the correct type
	obj1, _ := scheme.New("StringType")
	if _, ok := obj1.(string); !ok {
		t.Error("StringType factory didn't return string")
	}

	obj2, _ := scheme.New("IntType")
	if _, ok := obj2.(int); !ok {
		t.Error("IntType factory didn't return int")
	}

	obj3, _ := scheme.New("StructType")
	if _, ok := obj3.(*TestStruct); !ok {
		t.Error("StructType factory didn't return *TestStruct")
	}
}
