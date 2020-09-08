package mapper

import (
	"fmt"
	"reflect"
	"sync"
)

// fieldInfo holds metadata for a single struct field.
type fieldInfo struct {
	Index    []int
	Path     string
	Field    reflect.StructField
	Parent   *fieldInfo
	Children []*fieldInfo
}

// A structMap maps a string path to all of a structs children and grandchildren.
type structMap struct {
	Paths map[string]*fieldInfo
}

// Mapper is a general purpose Mapper of names to struct fields. A Mapper
// behaves like most marshallers in the standard library, but without tag support.
type Mapper struct {
	mutex sync.Mutex
	cache map[reflect.Type]*structMap
}

// getMapping returns a mapping of field strings to int slices representing
// the traversal down the struct to reach the field.
func (m *Mapper) getMapping(t reflect.Type) *structMap {
	m.mutex.Lock()
	mapping, ok := m.cache[t]
	if !ok {
		if m.cache == nil {
			m.cache = make(map[reflect.Type]*structMap)
		}

		mapping = createMapping(t)
		m.cache[t] = mapping
	}
	m.mutex.Unlock()
	return mapping
}

func (m *Mapper) GetField(v reflect.Value, path string) reflect.Value {
	v = reflect.Indirect(v)
	tm := m.getMapping(v.Type())

	if fi, ok := tm.Paths[path]; ok {
		return v.FieldByIndex(fi.Index)
	}
	return reflect.Value{}
}

// -- helpers & utilities --
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

// createMapping returns a mapping for the t type, using the tagName, mapFunc and
// tagMapFunc to determine the canonical names of fields.
func createMapping(t reflect.Type) *structMap {
	var allInfo []*fieldInfo
	queue := []typeQueue{
		{deref(t), &fieldInfo{}, ""},
	}

	for len(queue) != 0 {
		// pop the first item off of the queue
		tq := queue[0]
		queue = queue[1:]

		// ignore recursive field
		for p := tq.fi.Parent; p != nil; p = p.Parent {
			if tq.fi.Field.Type == p.Field.Type {
				panic(fmt.Errorf("cannot handle circular type %s", p.Field.Type))
			}
		}

		nChildren := tq.t.NumField()
		tq.fi.Children = make([]*fieldInfo, nChildren)

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

			tq.fi.Children[fieldPos] = fi
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
		Paths: make(map[string]*fieldInfo, len(allInfo)),
	}
	// Add the paths in reverse so that if a field name conflicts with a child of an embedded
	// sibling it will take precedence (all sibling should be added to the queue before any
	// of them can add their own children).
	for i := len(allInfo) - 1; i >= 0; i-- {
		result.Paths[allInfo[i].Path] = allInfo[i]
	}

	return result
}
