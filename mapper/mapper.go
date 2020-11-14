package mapper

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

const errMapPrefix = "cannot create map for"

var (
	mutex sync.Mutex
	cache = make(map[reflect.Type]*structMap)
)

// A structMap maps a string path to all of a structs children and grandchildren.
type structMap struct {
	Paths map[string][]int
}

// getMapping returns a mapping of field strings to int slices representing
// the traversal down the struct to reach the field.
func getMapping(t reflect.Type) *structMap {
	mutex.Lock()
	defer mutex.Unlock()

	mapping, ok := cache[t]
	if !ok {
		mapping = createMapping(t)
		cache[t] = mapping
	}
	return mapping
}

func GetField(v reflect.Value, path string) reflect.Value {
	// We want to catch all errors related to invalid paths, as the caller should be able
	// to decide what to do if the path returns an invalid value.
	defer func() {
		if err, ok := recover().(error); ok && strings.HasPrefix(err.Error(), errMapPrefix) {
			panic(err)
		}
	}()

	return getField(v, path)
}
func getField(v reflect.Value, path string) reflect.Value {
	v = reflect.Indirect(v)
	if path == "" {
		return v
	}

	if ob, cb := strings.Index(path, "["), strings.Index(path, "]"); cb > ob {
		parent := getField(v, path[:ob])
		subPath := strings.TrimPrefix(path[cb+1:], ".")
		if key := path[ob+1 : cb]; key[0] == '"' {
			if keyVal, err := strconv.Unquote(key); err == nil {
				return getField(parent.MapIndex(reflect.ValueOf(keyVal)), subPath)
			}
		} else if ind, err := strconv.Atoi(key); err == nil {
			return getField(parent.Index(ind), subPath)
		}
		return reflect.Value{}
	}

	if index, ok := getMapping(v.Type()).Paths[path]; ok {
		return v.FieldByIndex(index)
	}
	return reflect.Value{}
}

// -- helpers & utilities --
type fieldInfo struct {
	Index  []int
	Path   string
	Field  reflect.StructField
	Parent *fieldInfo
}
type typeQueue struct {
	t  reflect.Type
	fi *fieldInfo
	pp string // path prefix
}

// deref is Indirect for reflect.Types
func deref(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

// A copying append that creates a new slice each time.
func appendCopy(is []int, i int) []int {
	x := make([]int, len(is), len(is)+1)
	copy(x, is)
	return append(x, i)
}

func createMapping(t reflect.Type) *structMap {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Errorf("%s %s: %v", errMapPrefix, t, err))
		}
	}()

	var allInfo []*fieldInfo
	queue := []typeQueue{
		{deref(t), &fieldInfo{}, ""},
	}

	for len(queue) != 0 {
		// pop the first item off of the queue
		tq := queue[0]
		queue = queue[1:]

		// fail on recursive fields
		for p := tq.fi.Parent; p != nil; p = p.Parent {
			if tq.fi.Field.Type == p.Field.Type {
				panic(fmt.Errorf("cannot handle circular type %s", p.Field.Type))
			}
		}

		nChildren := tq.t.NumField()

		// iterate through all of its fields
		for fieldPos := 0; fieldPos < nChildren; fieldPos++ {
			f := tq.t.Field(fieldPos)
			// skip unexported fields
			if f.PkgPath != "" && !f.Anonymous {
				continue
			}

			fi := &fieldInfo{
				Index:  appendCopy(tq.fi.Index, fieldPos),
				Path:   tq.pp + f.Name,
				Field:  f,
				Parent: tq.fi,
			}

			allInfo = append(allInfo, fi)

			if childType := deref(f.Type); childType.Kind() == reflect.Struct {
				// For anonymous structs allow access using full path just like normal go
				queue = append(queue, typeQueue{childType, fi, fi.Path + "."})
				if f.Anonymous {
					queue = append(queue, typeQueue{childType, fi, tq.pp})
				}
			}
		}
	}

	result := &structMap{
		Paths: make(map[string][]int, len(allInfo)),
	}
	// Add the paths in reverse so that if a field name conflicts with a child of an embedded
	// sibling it will take precedence (all sibling should be added to the queue before any
	// of them can add their own children).
	for i := len(allInfo) - 1; i >= 0; i-- {
		result.Paths[allInfo[i].Path] = allInfo[i].Index
	}

	return result
}
