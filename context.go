package vue

import (
	"fmt"
	"reflect"
	"strings"
)

// Context is received by functions to interact with the component.
type Context interface {
	Data() interface{}
	Get(field string) interface{}
	Set(field string, value interface{})
	Go(method string, args ...interface{})
	Emit(event string, args ...interface{})
}

// Data returns the data for the component.
// Props and computed are excluded from data.
func (vm *ViewModel) Data() interface{} {
	return vm.data.Interface()
}

// Get returns the data field value.
// Props and computed are included to get.
func (vm *ViewModel) Get(field string) interface{} {
	if rv := vm.getValue(field); rv.IsValid() {
		return rv.Interface()
	}
	panic(fmt.Errorf("unknown data field: %s", field))
}
func (vm *ViewModel) getValue(field string) reflect.Value {
	if rv := vm.mapper.GetField(vm.data, field); rv.IsValid() {
		return rv
	}

	var topLevel, subPath string
	if ind := strings.IndexAny(field, ".["); ind < 0 {
		topLevel = field
	} else if field[ind] == '[' {
		topLevel, subPath = field[:ind], field[ind:]
	} else {
		topLevel, subPath = field[:ind], field[ind+1:]
	}

	value, ok := vm.props[topLevel]
	if !ok {
		value, ok = vm.cache[topLevel]
	}

	var rv reflect.Value
	if ok {
		rv = reflect.ValueOf(value)
		if subPath != "" {
			rv = vm.mapper.GetField(rv, subPath)
		}
	}

	return rv
}

// Set assigns the data field to the given value.
// Props and computed are excluded to set.
func (vm *ViewModel) Set(field string, newVal interface{}) {
	fieldVal := vm.mapper.GetField(vm.data, field)
	if fieldVal.Kind() == reflect.Invalid {
		panic(fmt.Errorf("unknown data field: %s", field))
	}

	oldVal := fieldVal.Interface()
	if reflect.DeepEqual(oldVal, newVal) {
		return
	}

	fieldVal.Set(reflect.ValueOf(newVal))
	if watcher, ok := vm.comp.watchers[field]; ok {
		watcher.Call([]reflect.Value{
			reflect.ValueOf(vm),
			reflect.ValueOf(newVal),
			reflect.ValueOf(oldVal),
		})
	}

	vm.render()
}

// Go asynchronously calls the given method with optional arguments.
// Blocking functions must be called asynchronously.
func (vm *ViewModel) Go(method string, args ...interface{}) {
	values := make([]reflect.Value, 0, len(args))
	for _, arg := range args {
		values = append(values, reflect.ValueOf(arg))
	}
	go vm.call(method, values)
}

// Emit dispatches the given event with optional arguments.
func (vm *ViewModel) Emit(event string, args ...interface{}) {
	vm.bus.pub(event, "", args)
}

// call calls the given method with optional values then calls render.
func (vm *ViewModel) call(method string, values []reflect.Value) {
	if function, ok := vm.comp.methods[method]; ok {
		values = append([]reflect.Value{reflect.ValueOf(vm)}, values...)
		function.Call(values)
		vm.render()
	}
}

// updateComputed evaluates every computed field for the component and stores the results in a
// cache. If the resulting values differ from previous it will also trigger the relevant watchers.
func (vm *ViewModel) updateComputed() {
	oldCache := vm.cache
	vm.cache = make(map[string]interface{}, len(vm.comp.computed))

	values := []reflect.Value{reflect.ValueOf(vm)}
	for computed, function := range vm.comp.computed {
		vm.cache[computed] = function.Call(values)[0].Interface()
	}

	for field := range oldCache {
		watcher, ok := vm.comp.watchers[field]
		if !ok || reflect.DeepEqual(oldCache[field], vm.cache[field]) {
			continue
		}

		watcher.Call([]reflect.Value{
			reflect.ValueOf(vm),
			reflect.ValueOf(vm.cache[field]),
			reflect.ValueOf(oldCache[field]),
		})
	}
}
