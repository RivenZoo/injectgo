package injectgo

import (
	"fmt"
	"reflect"
)

const injectTag = "inject"

// FuncLabelSelector is used to decide a label is selected or not.
type FuncLabelSelector interface {
	IsLabelAllowed(string) bool
}

// Container receive all provided objects and function then inject all of them.
type Container struct {
	graph            *objectGraph
	namedValues      map[string]reflect.Value
	unnamedValues    []reflect.Value
	namedFunctions   map[string]InjectFunc
	unnamedFunctions []InjectFunc
	checker          *injectChecker
	detector         *cyclicDetector
}

// NewContainer
func NewContainer() (c *Container) {
	c = &Container{
		graph:            newObjectGraph(),
		namedValues:      make(map[string]reflect.Value),
		unnamedValues:    make([]reflect.Value, 0),
		namedFunctions:   make(map[string]InjectFunc),
		unnamedFunctions: make([]InjectFunc, 0),
		checker:          newInjectChecker(),
		detector:         newCyclicDetector(),
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

		// add cyclic detector
		c.detector.AddDetectObject(v)

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

	// add cyclic detector
	c.detector.AddDetectObject(v)

	c.namedValues[name] = v
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
		ifn.validate()

		c.unnamedFunctions = append(c.unnamedFunctions, ifn)
	}
}

// ProvideFuncByName use `name` as object name, panic if name is duplicate.
func (c *Container) ProvideFuncByName(name string, ifn InjectFunc) {
	ifn.validate()

	if _, ok := c.namedFunctions[name]; ok {
		panic(fmt.Errorf("duplicate function name: %s", name))
	}
	if _, ok := c.namedValues[name]; ok {
		panic(fmt.Errorf("duplicate object name: %s", name))
	}
	c.namedFunctions[name] = ifn
}

func (c *Container) newObjectsByFunctions(labelSelector FuncLabelSelector) {
	for i := range c.unnamedFunctions {
		label := c.unnamedFunctions[i].Label
		if labelSelector != nil && label != "" && !labelSelector.IsLabelAllowed(label) {
			continue
		}
		v, err := c.unnamedFunctions[i].create()
		if err != nil {
			panic(fmt.Errorf("unamed function error: %v", err))
		}

		c.unnamedFunctions[i].setReceiver(v)

		// fulfill already exists object
		c.checker.popFulfilledUnnamedValues(v)
		// extract injected struct fields
		c.checker.pushInjectedFields(v)

		// add cyclic detector
		c.detector.AddDetectObject(v)

		c.unnamedValues = append(c.unnamedValues, v)
	}
	for name, fn := range c.namedFunctions {
		label := fn.Label
		if labelSelector != nil && label != "" && !labelSelector.IsLabelAllowed(label) {
			continue
		}

		v, err := fn.create()
		if err != nil {
			panic(fmt.Errorf("function %s error: %v", name, err))
		}

		fn.setReceiver(v)

		// fulfill already exists object
		c.checker.popFulfilledNamedValues(name, v)
		// extract injected struct fields
		c.checker.pushInjectedFields(v)

		// add cyclic detector
		c.detector.AddDetectObject(v)

		c.namedValues[name] = v
	}
}

func (c *Container) provideObjects() {
	for i := range c.unnamedValues {
		c.graph.ProvideObj(c.unnamedValues[i])
	}
	for name, v := range c.namedValues {
		c.graph.ProvideNamedObj(name, v)
	}
}

// Populate call all provided functions then inject all provided and returned by function objects.
// It panics if any error occurs.
// Param labelSelector choice function with it's label. If nil passed, all function will selected.
// If Initializable is implemented, Init method will be called after object populated.
func (c *Container) Populate(labelSelector FuncLabelSelector) {
	c.newObjectsByFunctions(labelSelector)

	c.checker.popRemainedValues()
	if !c.checker.isAllFulfilled() {
		unnamed := c.checker.getUnfulfilledUnnamedValues()
		named := c.checker.getUnfulfilledNamedValues()
		panic(fmt.Errorf("named unfulfilled objects: %v, unnamed unfulfilled objects: %s",
			(namedValueMap)(named).prettify(), (fieldValueMap)(unnamed).prettify()))
	}

	existsCyclic, cyclicPath := c.detector.DetectCyclic()
	if existsCyclic {
		panic(fmt.Errorf("dependency cyclic detected, cyclic path %s", cyclicPath.prettify()))
	}

	c.provideObjects()
	c.graph.Populate()
}

// Close will call Close method if Closable is implemented.
// It panics if any error occurs.
func (c *Container) Close() {
	c.graph.Close()
}
