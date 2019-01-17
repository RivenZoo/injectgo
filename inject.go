package injectgo

import (
	"fmt"
	"reflect"

	"github.com/facebookgo/inject"
)

const injectTag = "inject"

type funcValue struct {
	fn       reflect.Value
	label    string
	receiver interface{}
}

// InjectFunc contains a function to new object and label of the function.
type InjectFunc struct {
	Fn       interface{} // func() T / func() (T, error)
	Label    string      // default selected
	Receiver interface{} // *T, receive object from Fn
}

// FuncLabelSelector is used to decide a label is selected or not.
type FuncLabelSelector interface {
	IsLabelAllowed(string) bool
}

// Container receive all provided objects and function then inject all of them.
type Container struct {
	graph            *inject.Graph
	namedValues      map[string]reflect.Value
	unnamedValues    []reflect.Value
	namedFunctions   map[string]funcValue
	unnamedFunctions []funcValue
	checker          *injectChecker
}

// NewContainer
func NewContainer() (c *Container) {
	c = &Container{
		graph:            &inject.Graph{},
		namedValues:      make(map[string]reflect.Value),
		unnamedValues:    make([]reflect.Value, 0),
		namedFunctions:   make(map[string]funcValue),
		unnamedFunctions: make([]funcValue, 0),
		checker:          newInjectChecker(),
	}
	return
}

// isStructPtrOrInterface return true if obj is pointer or interface.
func (c *Container) isStructPtrOrInterface(obj reflect.Value) bool {
	switch obj.Type().Kind() {
	case reflect.Interface:
		return true
	case reflect.Ptr:
		if reflect.Indirect(obj).Type().Kind() == reflect.Struct {
			return true
		}
	default:
	}
	return false
}

// Provide panics if objs are not pointer to struct or interface.
func (c *Container) Provide(objs ...interface{}) {
	for i := range objs {
		v := reflect.ValueOf(objs[i])
		if !c.isStructPtrOrInterface(v) {
			panic(fmt.Errorf("check obj: %v error: %v", objs[i], errValueNotPtrOrInterface))
		}
		// fulfill already exists object
		c.checker.popFulfilledUnnamedValues(v)
		// extract injected struct fields
		c.checker.pushInjectedFields(v)

		c.unnamedValues = append(c.unnamedValues, v)
	}
}

// ProvideByName panics if name is duplicate.
// Param name should match other object inject tag like `inject:"Name"`.
func (c *Container) ProvideByName(name string, obj interface{}) {
	v := reflect.ValueOf(obj)
	if !c.isStructPtrOrInterface(v) {
		panic(fmt.Errorf("check obj: %v error: %v", obj, errValueNotPtrOrInterface))
	}
	if _, ok := c.namedValues[name]; ok {
		panic(fmt.Errorf("duplicate object name: %s", name))
	}

	// fulfill already exists object
	c.checker.popFulfilledNamedValues(name, v)
	// extract injected struct fields
	c.checker.pushInjectedFields(v)

	c.namedValues[name] = v
}

func (c *Container) validateFunc(name string, fn reflect.Value) {
	t := fn.Type()
	if t.NumIn() != 0 {
		panic(fmt.Errorf("func %s should not accept arguments", name))
	}
	if t.NumOut() <= 0 || t.NumOut() > 2 {
		panic(fmt.Errorf("func %s should be at most 2 return values", name))
	}
	if t.NumOut() == 2 {
		if t.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			panic(fmt.Errorf("func %s second return value should be error", name))
		}
	}
}

// ProvideFunc support function types:
//	- func() T
//	- func() (T, error)
// If use unsupport function as arguments, it will panic.
// Param label is associated with fn and can be selected.
// Only selected function will call.
// If label is empty, by default it is selected.
func (c *Container) ProvideFunc(funcs ...InjectFunc) {
	for i := range funcs {
		ifn := funcs[i]
		v := reflect.Indirect(reflect.ValueOf(ifn.Fn))
		if v.Type().Kind() != reflect.Func {
			panic(errValueNotFunction)
		}
		c.validateFunc("unamed", v)
		c.unnamedFunctions = append(c.unnamedFunctions, funcValue{fn: v,
			label:    ifn.Label,
			receiver: ifn.Receiver,
		})
	}
}

// ProvideFuncByName use `name` as object name, panic if name is duplicate.
func (c *Container) ProvideFuncByName(name string, ifn InjectFunc) {
	v := reflect.Indirect(reflect.ValueOf(ifn.Fn))
	if v.Type().Kind() != reflect.Func {
		panic(errValueNotFunction)
	}
	c.validateFunc(name, v)
	if _, ok := c.namedFunctions[name]; ok {
		panic(fmt.Errorf("duplicate function name: %s", name))
	}
	if _, ok := c.namedValues[name]; ok {
		panic(fmt.Errorf("duplicate object name: %s", name))
	}
	c.namedFunctions[name] = funcValue{fn: v,
		label:    ifn.Label,
		receiver: ifn.Receiver,
	}
}

func (c *Container) callProvidedFunc(fn reflect.Value) (reflect.Value, error) {
	ret := fn.Call(nil)
	if len(ret) == 1 {
		return ret[0], nil
	}
	if len(ret) == 2 {
		var err error
		if !ret[1].IsNil() {
			err = ret[1].Interface().(error)
		}
		return ret[0], err
	}
	panic(fmt.Errorf("call unsupport function"))
}

func setFunctionReceiver(v reflect.Value, receiver reflect.Value) {
	receiver.Elem().Set(v)
}

func (c *Container) newObjectsByFunctions(labelSelector FuncLabelSelector) {
	for i := range c.unnamedFunctions {
		label := c.unnamedFunctions[i].label
		if labelSelector != nil && label != "" && !labelSelector.IsLabelAllowed(label) {
			continue
		}
		v, err := c.callProvidedFunc(c.unnamedFunctions[i].fn)
		if err != nil {
			panic(fmt.Errorf("unamed function error: %v", err))
		}

		if c.unnamedFunctions[i].receiver != nil {
			setFunctionReceiver(v, reflect.ValueOf(c.unnamedFunctions[i].receiver))
		}

		// fulfill already exists object
		c.checker.popFulfilledUnnamedValues(v)
		// extract injected struct fields
		c.checker.pushInjectedFields(v)
		c.unnamedValues = append(c.unnamedValues, v)
	}
	for name, fn := range c.namedFunctions {
		label := fn.label
		if labelSelector != nil && label != "" && !labelSelector.IsLabelAllowed(label) {
			continue
		}
		v, err := c.callProvidedFunc(fn.fn)
		if err != nil {
			panic(fmt.Errorf("function %s error: %v", name, err))
		}

		if fn.receiver != nil {
			setFunctionReceiver(v, reflect.ValueOf(fn.receiver))
		}

		// fulfill already exists object
		c.checker.popFulfilledNamedValues(name, v)
		// extract injected struct fields
		c.checker.pushInjectedFields(v)

		c.namedValues[name] = v
	}
}

func (c *Container) provideObjects() {
	for i := range c.unnamedValues {
		err := c.graph.Provide(&inject.Object{Value: c.unnamedValues[i].Interface()})
		if err != nil {
			panic(fmt.Errorf("provide value: %v error: %v", c.unnamedValues[i], err))
		}
	}
	for name, v := range c.namedValues {
		err := c.graph.Provide(&inject.Object{Name: name, Value: v.Interface()})
		if err != nil {
			panic(fmt.Errorf("provide value: %s=%v error: %v", name, v, err))
		}
	}
}

// Populate call all provided functions then inject all provided and returned by function objects.
// It panics if any error occurs.
// Param labelSelector choice function with it's label. If nil passed, all function will selected.
func (c *Container) Populate(labelSelector FuncLabelSelector) {
	c.newObjectsByFunctions(labelSelector)

	c.checker.popRemainedValues()
	if !c.checker.isAllFulfilled() {
		unnamed := c.checker.getUnfulfilledUnnamedValues()
		named := c.checker.getUnfulfilledNamedValues()
		panic(fmt.Errorf("named unfulfilled objects: %v, unnamed unfulfilled objects: %v",
			named, unnamed))
	}

	c.provideObjects()
	err := c.graph.Populate()
	if err != nil {
		panic(fmt.Errorf("populate error: %v", err))
	}
}
