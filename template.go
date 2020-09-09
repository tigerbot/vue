package vue

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/cbroglie/mustache"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	v      = "v-"
	vBind  = "v-bind"
	vFor   = "v-for"
	vHtml  = "v-html"
	vIf    = "v-if"
	vModel = "v-model"
	vOn    = "v-on"
)

var attrOrder = []string{vFor, vIf, vModel, vOn, vBind, vHtml}

// render executes and renders the prepared state.
func (vm *ViewModel) render() {
	node := vm.execute()

	vm.subs.reset()
	if vm.comp.isSub {
		var ok bool
		if node, ok = firstElement(node); !ok {
			must(fmt.Errorf("failed to find first element from node: %s", node.Data))
		}
	}
	vm.vnode.render(node, vm.subs)
	vm.subs.reset()
}

// execute executes the template with the given data to be rendered.
func (vm *ViewModel) execute() *html.Node {
	node := parseNode(vm.comp.tmpl)

	vm.executeElement(node)
	vm.executeText(node)

	return node
}

// executeElement recursively traverses the html node and templates the elements.
// The next node is always returned which allows execution to jump around as needed.
func (vm *ViewModel) executeElement(node *html.Node) *html.Node {
	// Leave the text nodes to be executed.
	if node.Type != html.ElementNode {
		return node.NextSibling
	}

	// Order attributes before execution.
	orderAttrs(node)

	// Execute attributes.
	for i := 0; i < len(node.Attr); i++ {
		attr := node.Attr[i]
		if strings.HasPrefix(attr.Key, v) {
			node.Attr = append(node.Attr[:i], node.Attr[i+1:]...)
			i--

			// The current node is not longer valid in favor of the next node.
			if next, modified := vm.executeAttr(node, attr); modified {
				return next
			}
		}
	}

	// Execute subcomponent.
	if vm.subs.newInstance(node.Data, vm) {
		return node.NextSibling
	}

	// Execute children.
	for child := node.FirstChild; child != nil; {
		child = vm.executeElement(child)
	}

	return node.NextSibling
}

// executeText recursively executes the text node.
func (vm *ViewModel) executeText(node *html.Node) {
	switch node.Type {
	case html.TextNode:
		if strings.TrimSpace(node.Data) == "" {
			return
		}

		var err error
		node.Data, err = mustache.Render(node.Data, vm.data, vm.props, vm.cache)
		must(err)
	case html.ElementNode:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			vm.executeText(child)
		}
	}
}

// executeAttr executes the given vue attribute.
// The next node will be executed next if the html was modified unless it is nil.
func (vm *ViewModel) executeAttr(node *html.Node, attr html.Attribute) (*html.Node, bool) {
	vals := strings.Split(attr.Key, ":")
	typ, part := vals[0], ""
	if len(vals) > 1 {
		part = vals[1]
	}

	var next *html.Node
	var modified bool
	switch typ {
	case vBind:
		vm.executeAttrBind(node, part, attr.Val)
	case vFor:
		next, modified = vm.executeAttrFor(node, attr.Val)
	case vHtml:
		vm.executeAttrHtml(node, attr.Val)
	case vIf:
		next, modified = vm.executeAttrIf(node, attr.Val)
	case vModel:
		vm.executeAttrModel(node, attr.Val)
	case vOn:
		vm.executeAttrOn(node, part, attr.Val)
	default:
		must(fmt.Errorf("unknown vue attribute: %v", typ))
	}
	return next, modified
}

// executeAttrBind executes the vue bind attribute.
func (vm *ViewModel) executeAttrBind(node *html.Node, key, field string) {
	value := vm.Get(field)
	if value == nil {
		must(fmt.Errorf("unknown data field: %s", field))
	}

	prop := strings.Title(key)
	if ok := vm.subs.putProp(node.Data, prop, value); ok {
		return
	}

	if key == "class" {
		class := formatAttrClass(value)
		node.Attr = append(node.Attr, html.Attribute{Key: key, Val: class})
		return
	}

	if key == "style" {
		style := formatAttrStyle(value)
		node.Attr = append(node.Attr, html.Attribute{Key: key, Val: style})
		return
	}

	// Remove attribute if bound to a false value of type bool.
	if val, ok := value.(bool); ok && !val {
		return
	}

	val := fmt.Sprintf("%v", value)
	node.Attr = append(node.Attr, html.Attribute{Key: key, Val: val})
}

// executeAttrFor executes the vue for attribute.
func (vm *ViewModel) executeAttrFor(node *html.Node, value string) (*html.Node, bool) {
	vals := strings.Split(value, "in")
	name := bytes.TrimSpace([]byte(vals[0]))
	field := strings.TrimSpace(vals[1])

	slice := vm.getValue(field)
	if k := slice.Kind(); k != reflect.Slice && k != reflect.Array {
		must(fmt.Errorf("slice not found for field: %s", field))
	}

	n := slice.Len()
	if n == 0 {
		next := node.NextSibling
		node.Parent.RemoveChild(node)
		return next, true
	}

	elem := bytes.NewBuffer(nil)
	must(html.Render(elem, node))

	buf := bytes.NewBuffer(nil)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("%s[%d]", field, i)

		b := bytes.Replace(elem.Bytes(), name, []byte(key), -1)
		_, err := buf.Write(b)
		must(err)
	}

	nodes := parseNodes(buf)
	for _, child := range nodes {
		node.Parent.InsertBefore(child, node)
	}
	node.Parent.RemoveChild(node)
	// The first child is the next node to execute.
	return nodes[0], true
}

// executeAttrHtml executes the vue html attribute.
func (vm *ViewModel) executeAttrHtml(node *html.Node, field string) {
	html, ok := vm.Get(field).(string)
	if !ok {
		must(fmt.Errorf("data field is not of type string: %T", field))
	}

	nodes := parseNodes(strings.NewReader(html))
	for _, child := range nodes {
		node.AppendChild(child)
	}
}

// executeAttrIf executes the vue if attribute.
func (vm *ViewModel) executeAttrIf(node *html.Node, field string) (*html.Node, bool) {
	negate := strings.HasPrefix(field, "!")
	if negate {
		field = field[1:]
	}

	if val, ok := vm.Get(field).(bool); ok && val != negate {
		return nil, false
	}

	next := node.NextSibling
	node.Parent.RemoveChild(node)
	return next, true
}

// executeAttrModel executes the vue model attribute.
func (vm *ViewModel) executeAttrModel(node *html.Node, field string) {
	typ := "input"
	node.Attr = append(node.Attr, html.Attribute{Key: typ, Val: field})

	val, ok := vm.Get(field).(string)
	if !ok {
		must(fmt.Errorf("data field is not of type string: %T", field))
	}
	node.Attr = append(node.Attr, html.Attribute{Key: "value", Val: val})

	vm.addEventListener(typ, vm.vModel)
}

// executeAttrOn executes the vue on attribute.
func (vm *ViewModel) executeAttrOn(node *html.Node, typ, method string) {
	event := strings.Split(typ, ".")[0]
	node.Attr = append(node.Attr, html.Attribute{Key: typ, Val: method})

	vm.addEventListener(event, vm.vOn)
	vm.bus.sub(event, method)
}

// parseNode parses the template into an html node.
// The node returned is a placeholder, not to be rendered.
func parseNode(tmpl string) *html.Node {
	nodes := parseNodes(strings.NewReader(tmpl))
	node := &html.Node{Type: html.ElementNode}
	for _, child := range nodes {
		node.AppendChild(child)
	}
	return node
}

// parseNodes parses the reader into html nodes.
func parseNodes(reader io.Reader) []*html.Node {
	nodes, err := html.ParseFragment(reader, &html.Node{
		Type:     html.ElementNode,
		Data:     "div",
		DataAtom: atom.Div,
	})
	must(err)
	return nodes
}

// firstElement finds the first child element of a node.
// Returns false if a child element is not found.
func firstElement(node *html.Node) (*html.Node, bool) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			return child, true
		}
	}
	return nil, false
}

// orderAttrs orders the attributes of the node which orders the template execution.
func orderAttrs(node *html.Node) {
	n := len(node.Attr)
	if n == 0 {
		return
	}
	attrs := make([]html.Attribute, 0, n)
	for _, prefix := range attrOrder {
		for _, attr := range node.Attr {
			if strings.HasPrefix(attr.Key, prefix) {
				attrs = append(attrs, attr)
			}
		}
	}
	// Append other attributes which are not vue attributes.
	for _, attr := range node.Attr {
		if !strings.HasPrefix(attr.Key, v) {
			attrs = append(attrs, attr)
		}
	}
	node.Attr = attrs
}

// formatAttrClass formats the value into a class attribute.
// For example: { Active: true, DangerText: true } -> "active danger-text"
// For type: struct { Active: bool `css:"active"`, DangerText: bool `css:"danger-text"` }
func formatAttrClass(value interface{}) string {
	elem := reflect.Indirect(reflect.ValueOf(value))
	typ := elem.Type()
	n := elem.NumField()
	buf := bytes.NewBuffer(nil)
	format := "%s"
	for i := 0; i < n; i++ {
		if field := elem.Field(i); field.CanInterface() {
			value := field.Interface()
			if val, ok := value.(bool); ok && val {
				typ := typ.Field(i)
				class := typ.Tag.Get("css")
				if class == "" {
					class = strings.ToLower(typ.Name)
				}
				fmt.Fprintf(buf, format, class)
				format = " %s"
			}
		}
	}
	return buf.String()
}

// formatAttrStyle formats the value into a style attribute.
// For example: { Color: red, FontSize: 8px } -> "color: red; font-size: 8px"
// For type: struct { Color: string `css:"color"`, FontSize: string `css:"font-size"` }
func formatAttrStyle(value interface{}) string {
	elem := reflect.Indirect(reflect.ValueOf(value))
	typ := elem.Type()
	n := elem.NumField()
	buf := bytes.NewBuffer(nil)
	format := "%s: %v"
	for i := 0; i < n; i++ {
		if field := elem.Field(i); field.CanInterface() {
			typ := typ.Field(i)
			style := typ.Tag.Get("css")
			if style == "" {
				style = strings.ToLower(typ.Name)
			}
			value := field.Interface()
			fmt.Fprintf(buf, format, style, value)
			format = "; %s: %v"
		}
	}
	return buf.String()
}
