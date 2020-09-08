package mapper

import (
	"reflect"
	"testing"
)

func TestMapper(t *testing.T) {
	type Grand struct {
		Bool bool
	}
	type Child struct {
		String string
		Int    int
		Float  float64
		Grand  Grand
	}
	type Node struct {
		String string
		Child  // Between fields of same name as child to test embedding precedence
		Float  float64
		priv   bool

		Direct  Child
		Pointer *Child
	}

	data := Node{
		String: "root node",
		Child: Child{
			String: "embedded child",
			Int:    107734,
			Float:  3.14159,
			Grand:  Grand{},
		},
		Float: 42.0,
		priv:  false,

		Direct: Child{
			String: "direct child",
			Int:    123456,
			Float:  6.54321,
			Grand:  Grand{true},
		},
		Pointer: &Child{
			String: "pointer child",
			Int:    654321,
			Float:  1.23456,
			Grand:  Grand{false},
		},
	}

	cases := map[string]reflect.Value{
		"String": reflect.ValueOf(data.String),
		"Child":  reflect.ValueOf(data.Child),
		"Int":    reflect.ValueOf(data.Int),
		"Float":  reflect.ValueOf(data.Float),
		"Direct": reflect.ValueOf(data.Direct),

		"Child.String":     reflect.ValueOf(data.Child.String),
		"Child.Int":        reflect.ValueOf(data.Child.Int),
		"Child.Float":      reflect.ValueOf(data.Child.Float),
		"Child.Grand":      reflect.ValueOf(data.Child.Grand),
		"Child.Grand.Bool": reflect.ValueOf(data.Child.Grand.Bool),

		"Direct.String":     reflect.ValueOf(data.Direct.String),
		"Direct.Int":        reflect.ValueOf(data.Direct.Int),
		"Direct.Float":      reflect.ValueOf(data.Direct.Float),
		"Direct.Grand":      reflect.ValueOf(data.Direct.Grand),
		"Direct.Grand.Bool": reflect.ValueOf(data.Direct.Grand.Bool),

		"Pointer.String":     reflect.ValueOf(data.Pointer.String),
		"Pointer.Int":        reflect.ValueOf(data.Pointer.Int),
		"Pointer.Float":      reflect.ValueOf(data.Pointer.Float),
		"Pointer.Grand":      reflect.ValueOf(data.Pointer.Grand),
		"Pointer.Grand.Bool": reflect.ValueOf(data.Pointer.Grand.Bool),

		"priv":     {},
		"BadValue": {},
	}

	var m Mapper
	root := reflect.ValueOf(data)
	for path, want := range cases {
		got := m.GetField(root, path)
		if gk, wk := got.Kind(), want.Kind(); gk != wk {
			t.Errorf("getting %s returned kind %s, expected %s", path, gk, gk)
		} else if gk != reflect.Invalid {
			if got.Interface() != want.Interface() {
				t.Errorf("getting %s returned %v, expected %v", path, got, want)
			}
		}
	}
}
