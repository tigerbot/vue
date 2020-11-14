package mapper

import (
	"math"
	"reflect"
	"testing"
)

type BasicStruct struct {
	String string
	Int    int
	Float  float64
	Bool   bool
	priv   int
}

type testCase struct {
	root reflect.Value
	path string
	want reflect.Value
}

func (c testCase) Run(t *testing.T) {
	got := GetField(c.root, c.path)
	if gk, wk := got.Kind(), c.want.Kind(); gk != wk {
		t.Errorf("getting %s returned kind %s, expected %s", c.path, gk, wk)
	} else if gk != reflect.Invalid {
		if got.Interface() != c.want.Interface() {
			t.Errorf("getting %s returned %v, expected %v", c.path, got, c.want)
		}
	}
}

func TestBasic(t *testing.T) {
	data := BasicStruct{
		String: "simple case",
		Int:    987589,
		Float:  math.Sqrt2,
		Bool:   true,
		priv:   12345,
	}

	root := reflect.ValueOf(data)
	cases := []testCase{
		{root: root, path: "String", want: reflect.ValueOf(data.String)},
		{root: root, path: "Int", want: reflect.ValueOf(data.Int)},
		{root: root, path: "Bool", want: reflect.ValueOf(data.Bool)},
		{root: root, path: "Float", want: reflect.ValueOf(data.Float)},

		{root: root, path: "priv", want: reflect.Value{}},
		{root: root, path: "BadValue", want: reflect.Value{}},
	}

	for _, c := range cases {
		t.Run(c.path, c.Run)
	}
}

func TestDeep(t *testing.T) {
	type mid struct{ Bot BasicStruct }
	type top struct{ Mid mid }
	data := struct{ Top top }{
		Top: top{
			Mid: mid{
				Bot: BasicStruct{
					String: "great-grandchild",
					Int:    89762,
					Float:  math.SqrtE,
				},
			},
		},
	}

	root := reflect.ValueOf(data)
	cases := []testCase{
		{root: root, path: "Top", want: reflect.ValueOf(data.Top)},
		{root: root, path: "Top.Mid", want: reflect.ValueOf(data.Top.Mid)},
		{root: root, path: "Top.Mid.Bot", want: reflect.ValueOf(data.Top.Mid.Bot)},
		{root: root, path: "Top.Mid.Bot.String", want: reflect.ValueOf(data.Top.Mid.Bot.String)},
		{root: root, path: "Top.Mid.Bot.Int", want: reflect.ValueOf(data.Top.Mid.Bot.Int)},
		{root: root, path: "Top.Mid.Bot.Float", want: reflect.ValueOf(data.Top.Mid.Bot.Float)},
		{root: root, path: "Top.Mid.Bot.Bool", want: reflect.ValueOf(data.Top.Mid.Bot.Bool)},
	}

	for _, c := range cases {
		t.Run(c.path, c.Run)
	}
}

func TestEmbedding(t *testing.T) {
	data := struct {
		String string
		BasicStruct
		Float float64
	}{
		String: "root node",
		BasicStruct: BasicStruct{
			String: "embedded child",
			Int:    107734,
			Float:  3.14159,
		},
		Float: 42.0,
	}

	root := reflect.ValueOf(data)
	cases := []testCase{
		{root: root, path: "String", want: reflect.ValueOf(data.String)},
		{root: root, path: "Int", want: reflect.ValueOf(data.Int)},
		{root: root, path: "Bool", want: reflect.ValueOf(data.Bool)},
		{root: root, path: "Float", want: reflect.ValueOf(data.Float)},

		{root: root, path: "BasicStruct", want: reflect.ValueOf(data.BasicStruct)},
		{root: root, path: "BasicStruct.String", want: reflect.ValueOf(data.BasicStruct.String)},
		{root: root, path: "BasicStruct.Int", want: reflect.ValueOf(data.BasicStruct.Int)},
		{root: root, path: "BasicStruct.Float", want: reflect.ValueOf(data.BasicStruct.Float)},
		{root: root, path: "BasicStruct.Bool", want: reflect.ValueOf(data.BasicStruct.Bool)},
	}

	for _, c := range cases {
		t.Run(c.path, c.Run)
	}
}

func TestSlices(t *testing.T) {
	data := struct {
		Slice     []BasicStruct
		Array     [1]BasicStruct
		DoubleArr [1][1]BasicStruct
	}{
		Slice: []BasicStruct{{
			String: "slice child",
			Int:    2468,
			Float:  math.E,
		}},
		Array: [1]BasicStruct{{
			String: "array child",
			Int:    8642,
			Float:  math.Log2E,
		}},
		DoubleArr: [1][1]BasicStruct{{{
			String: "double array child",
			Int:    13579,
			Float:  math.Log10E,
		}}},
	}

	root := reflect.ValueOf(data)
	cases := []testCase{
		{root: root, path: "Slice[0]", want: reflect.ValueOf(data.Slice[0])},
		{root: root, path: "Slice[0].String", want: reflect.ValueOf(data.Slice[0].String)},
		{root: root, path: "Slice[0].Int", want: reflect.ValueOf(data.Slice[0].Int)},
		{root: root, path: "Slice[0].Float", want: reflect.ValueOf(data.Slice[0].Float)},

		{root: root, path: "Array", want: reflect.ValueOf(data.Array)},
		{root: root, path: "Array[0]", want: reflect.ValueOf(data.Array[0])},
		{root: root, path: "Array[0].String", want: reflect.ValueOf(data.Array[0].String)},
		{root: root, path: "Array[0].Int", want: reflect.ValueOf(data.Array[0].Int)},
		{root: root, path: "Array[0].Float", want: reflect.ValueOf(data.Array[0].Float)},

		{root: root, path: "DoubleArr", want: reflect.ValueOf(data.DoubleArr)},
		{root: root, path: "DoubleArr[0]", want: reflect.ValueOf(data.DoubleArr[0])},
		{root: root, path: "DoubleArr[0][0]", want: reflect.ValueOf(data.DoubleArr[0][0])},
		{root: root, path: "DoubleArr[0][0].String", want: reflect.ValueOf(data.DoubleArr[0][0].String)},
		{root: root, path: "DoubleArr[0][0].Int", want: reflect.ValueOf(data.DoubleArr[0][0].Int)},
		{root: root, path: "DoubleArr[0][0].Float", want: reflect.ValueOf(data.DoubleArr[0][0].Float)},

		{root: root, path: "Slice[x]", want: reflect.Value{}},
	}

	for _, c := range cases {
		t.Run(c.path, c.Run)
	}
}

func TestMaps(t *testing.T) {
	data := struct {
		Map map[string]BasicStruct
	}{
		Map: map[string]BasicStruct{
			"key": {
				String: "map child",
				Int:    951,
				Float:  math.SqrtPhi,
			},
		},
	}

	root := reflect.ValueOf(data)
	cases := []testCase{
		{root: root, path: `Map["key"]`, want: reflect.ValueOf(data.Map["key"])},
		{root: root, path: `Map["key"].String`, want: reflect.ValueOf(data.Map["key"].String)},
		{root: root, path: `Map["key"].Int`, want: reflect.ValueOf(data.Map["key"].Int)},
		{root: root, path: `Map["key"].Float`, want: reflect.ValueOf(data.Map["key"].Float)},

		{root: root, path: `Map['key"].Float`, want: reflect.Value{}},
	}

	for _, c := range cases {
		t.Run(c.path, c.Run)
	}
}
