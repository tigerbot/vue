package mapper

import (
	"math"
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

		Direct    Child
		Pointer   *Child
		Slice     []Child
		Array     [1]Child
		DoubleArr [1][1]Child
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

		Slice: []Child{{
			String: "slice child",
			Int:    2468,
			Float:  math.E,
			Grand:  Grand{true},
		}},
		Array: [1]Child{{
			String: "array child",
			Int:    8642,
			Float:  math.Log2E,
			Grand:  Grand{false},
		}},
		DoubleArr: [1][1]Child{{{
			String: "double array child",
			Int:    13579,
			Float:  math.Log10E,
			Grand:  Grand{true},
		}}},
	}

	cases := map[string]reflect.Value{
		"String": reflect.ValueOf(data.String),
		"Int":    reflect.ValueOf(data.Int),
		"Float":  reflect.ValueOf(data.Float),

		"Child":            reflect.ValueOf(data.Child),
		"Child.String":     reflect.ValueOf(data.Child.String),
		"Child.Int":        reflect.ValueOf(data.Child.Int),
		"Child.Float":      reflect.ValueOf(data.Child.Float),
		"Child.Grand":      reflect.ValueOf(data.Child.Grand),
		"Child.Grand.Bool": reflect.ValueOf(data.Child.Grand.Bool),

		"Direct":            reflect.ValueOf(data.Direct),
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

		// "Slice":  reflect.ValueOf(data.Slice),
		"Slice[0]":            reflect.ValueOf(data.Slice[0]),
		"Slice[0].String":     reflect.ValueOf(data.Slice[0].String),
		"Slice[0].Int":        reflect.ValueOf(data.Slice[0].Int),
		"Slice[0].Float":      reflect.ValueOf(data.Slice[0].Float),
		"Slice[0].Grand":      reflect.ValueOf(data.Slice[0].Grand),
		"Slice[0].Grand.Bool": reflect.ValueOf(data.Slice[0].Grand.Bool),

		"Array":               reflect.ValueOf(data.Array),
		"Array[0]":            reflect.ValueOf(data.Array[0]),
		"Array[0].String":     reflect.ValueOf(data.Array[0].String),
		"Array[0].Int":        reflect.ValueOf(data.Array[0].Int),
		"Array[0].Float":      reflect.ValueOf(data.Array[0].Float),
		"Array[0].Grand":      reflect.ValueOf(data.Array[0].Grand),
		"Array[0].Grand.Bool": reflect.ValueOf(data.Array[0].Grand.Bool),

		"DoubleArr":                  reflect.ValueOf(data.DoubleArr),
		"DoubleArr[0]":               reflect.ValueOf(data.DoubleArr[0]),
		"DoubleArr[0][0]":            reflect.ValueOf(data.DoubleArr[0][0]),
		"DoubleArr[0][0].String":     reflect.ValueOf(data.DoubleArr[0][0].String),
		"DoubleArr[0][0].Int":        reflect.ValueOf(data.DoubleArr[0][0].Int),
		"DoubleArr[0][0].Float":      reflect.ValueOf(data.DoubleArr[0][0].Float),
		"DoubleArr[0][0].Grand":      reflect.ValueOf(data.DoubleArr[0][0].Grand),
		"DoubleArr[0][0].Grand.Bool": reflect.ValueOf(data.DoubleArr[0][0].Grand.Bool),

		"priv":     {},
		"BadValue": {},
	}

	root := reflect.ValueOf(data)
	for path, want := range cases {
		got := GetField(root, path)
		if gk, wk := got.Kind(), want.Kind(); gk != wk {
			t.Errorf("getting %s returned kind %s, expected %s", path, gk, wk)
		} else if gk != reflect.Invalid {
			if got.Interface() != want.Interface() {
				t.Errorf("getting %s returned %v, expected %v", path, got, want)
			}
		}
	}
}
