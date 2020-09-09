// Package vue is the progressive framework for wasm applications.
package vue

import (
	"reflect"
	"syscall/js"

	"github.com/norunners/vue/mapper"
)

// ViewModel is a vue view model, e.g. VM.
type ViewModel struct {
	comp   *Comp
	vnode  *vnode
	data   reflect.Value
	funcs  map[string]js.Func
	props  map[string]interface{}
	cache  map[string]interface{}
	subs   subs
	bus    *bus
	mapper *mapper.Mapper
}

// New creates a new view model from the given options.
func New(options ...Option) *ViewModel {
	comp := Component(options...)
	return newViewModel(comp, nil, nil, new(mapper.Mapper))
}

// newViewModel creates a new view model from the given component with props.
func newViewModel(comp *Comp, bus *bus, props map[string]interface{}, m *mapper.Mapper) *ViewModel {
	var vnode *vnode
	if comp.isSub {
		vnode = newSubNode(comp.tmpl)
	} else {
		vnode = newNode(comp.el)
	}

	vm := &ViewModel{
		comp:   comp,
		props:  props,
		vnode:  vnode,
		mapper: m,
		data:   comp.newData(),
		subs:   newSubs(comp.subs),
		funcs:  make(map[string]js.Func, 0),
	}
	vm.bus = newBus(bus, vm)
	vm.updateComputed()
	vm.render()
	return vm
}

// must panics on errors.
func must(err error) {
	if err != nil {
		panic(err)
	}
}
