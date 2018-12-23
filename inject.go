package injectgo

import (
	"fmt"
	"reflect"

	"github.com/facebookgo/inject"
)

type Container struct {
	graph            *inject.Graph
	namedValues      map[string]reflect.Value
	unnamedValues    []reflect.Value
	namedFunctions   map[string]reflect.Value
	unnamedFunctions []reflect.Value
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
		if t.Out(1) != reflect.TypeOf(errValueNotFunction) {
			panic(fmt.Errorf("func %s second return value should be error", name))
		}
	}
}

// ProvideFunc support function types:
//	- func() T
//	- func() T, error
// If use unsupport function as arguments, it will panic.
func (c *Container) ProvideFunc(fn interface{}) {
	v := reflect.Indirect(reflect.ValueOf(fn))
	if v.Type().Kind() != reflect.Func {
		panic(errValueNotFunction)
	}
	c.validateFunc("unamed", v)
	c.unnamedFunctions = append(c.unnamedFunctions, v)
}

// ProvideFuncByName use `name` as object name, panic if name is duplicate.
func (c *Container) ProvideFuncByName(name string, fn interface{}) {
	v := reflect.Indirect(reflect.ValueOf(fn))
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
	c.namedFunctions[name] = v
}

func (c *Container) callProvidedFunc(fn reflect.Value) (reflect.Value, error) {
	ret := fn.Call(nil)
	if len(ret) == 1 {
		return ret[0], nil
	}
	if len(ret) == 2 {
		return ret[0], ret[1].Interface().(error)
	}
	panic(fmt.Errorf("call unsupport function"))
}

// Populate call all provided functions then inject all provided and returned by function objects.
// It panics if any error occurs.
func (c *Container) Populate() {
	for i := range c.unnamedFunctions {
		v, err := c.callProvidedFunc(c.unnamedFunctions[i])
		if err != nil {
			panic(fmt.Errorf("unamed function error: %v", err))
		}
		c.unnamedValues = append(c.unnamedValues, v)
	}
	for name, fn := range c.namedFunctions {
		v, err := c.callProvidedFunc(fn)
		if err != nil {
			panic(fmt.Errorf("function %s error: %v", name, err))
		}
		c.namedValues[name] = v
	}
	for i := range c.unnamedValues {
		err := c.graph.Provide(&inject.Object{Value: c.unnamedValues[i]})
		if err != nil {
			panic(fmt.Errorf("provide value: %v error: %v", c.unnamedValues[i], err))
		}
	}
	for name, v := range c.namedValues {
		err := c.graph.Provide(&inject.Object{Name: name, Value: v})
		if err != nil {
			panic(fmt.Errorf("provide value: %s=%v error: %v", name, v, err))
		}
	}
	err := c.graph.Populate()
	if err != nil {
		panic(fmt.Errorf("populate error: %v", err))
	}
}
