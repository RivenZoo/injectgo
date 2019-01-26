package injectgo

import (
	"fmt"
	"reflect"
	"strings"
)

type injectChecker struct {
	unfulfilledUnnamedValues     map[reflect.Type]reflect.Value
	unfulfilledUnnamedInterfaces map[reflect.Type]reflect.Value
	unfulfilledNamedValues       map[string]reflect.Value

	namedValues   map[string]reflect.Value
	unnamedValues map[reflect.Type]reflect.Value
}

func newInjectChecker() *injectChecker {
	return &injectChecker{
		unfulfilledUnnamedValues:     make(map[reflect.Type]reflect.Value),
		unfulfilledUnnamedInterfaces: make(map[reflect.Type]reflect.Value),
		unfulfilledNamedValues:       make(map[string]reflect.Value),
		namedValues:                  make(map[string]reflect.Value),
		unnamedValues:                make(map[reflect.Type]reflect.Value),
	}
}

func (c *injectChecker) pushInjectedFields(obj reflect.Value) {
	t := reflect.Indirect(obj).Type()
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if inj, ok := field.Tag.Lookup(injectTag); ok {
			if inj != "" {
				if _, ok := c.unfulfilledNamedValues[inj]; !ok {
					c.unfulfilledNamedValues[inj] = obj
				}
				continue
			}

			switch field.Type.Kind() {
			case reflect.Interface:
				if _, ok := c.unfulfilledUnnamedInterfaces[field.Type]; !ok {
					c.unfulfilledUnnamedInterfaces[field.Type] = obj
				}
			case reflect.Ptr:
				if _, ok := c.unfulfilledUnnamedValues[field.Type]; !ok {
					c.unfulfilledUnnamedValues[field.Type] = obj
				}
			default:
				panic(fmt.Errorf("field %v of object %v wrong type", field, obj))
			}
		}
	}
}

func (c *injectChecker) popFulfilledUnnamedValues(obj reflect.Value) {
	t := obj.Type()

	if _, ok := c.unnamedValues[t]; !ok {
		c.unnamedValues[t] = obj
	}
	if _, ok := c.unfulfilledUnnamedValues[t]; ok {
		delete(c.unfulfilledUnnamedValues, t)
	}
	toDel := make([]reflect.Type, 0)
	for k, _ := range c.unfulfilledUnnamedInterfaces {
		if t.AssignableTo(k) {
			toDel = append(toDel, k)
		}
	}
	for i := range toDel {
		delete(c.unfulfilledUnnamedInterfaces, toDel[i])
	}
}

func (c *injectChecker) popFulfilledNamedValues(name string, obj reflect.Value) {
	if _, ok := c.namedValues[name]; !ok {
		c.namedValues[name] = obj
	}
	if _, ok := c.unfulfilledNamedValues[name]; ok {
		delete(c.unfulfilledNamedValues, name)
	}
}

// popRemainedValues should be called after all push/pop functions finished.
func (c *injectChecker) popRemainedValues() {
	for _, v := range c.unnamedValues {
		if len(c.unfulfilledUnnamedInterfaces) > 0 || len(c.unfulfilledUnnamedValues) > 0 {
			c.popFulfilledUnnamedValues(v)
		}
	}
	for n, v := range c.namedValues {
		if len(c.unfulfilledNamedValues) > 0 {
			c.popFulfilledNamedValues(n, v)
		}
	}
}

func (c *injectChecker) getUnfulfilledUnnamedValues() map[reflect.Type]reflect.Value {
	ret := make(map[reflect.Type]reflect.Value)
	for k, v := range c.unfulfilledUnnamedInterfaces {
		ret[k] = v
	}
	for k, v := range c.unfulfilledUnnamedValues {
		ret[k] = v
	}
	return ret
}

func (c *injectChecker) getUnfulfilledNamedValues() map[string]reflect.Value {
	return c.unfulfilledNamedValues
}

func (c *injectChecker) isAllFulfilled() bool {
	return len(c.unfulfilledNamedValues) == 0 &&
		len(c.unfulfilledUnnamedValues) == 0 &&
		len(c.unfulfilledUnnamedInterfaces) == 0
}

type fieldValueMap map[reflect.Type]reflect.Value

func (fm fieldValueMap) prettify() string {
	s := make([]string, 0, len(fm))
	for tp, v := range fm {
		s = append(s, fmt.Sprintf("(%v).{%v}", v.Type(), tp))
	}
	return strings.Join(s, " ")
}

type namedValueMap map[string]reflect.Value

func (nm namedValueMap) prettify() string {
	s := make([]string, 0, len(nm))
	for name, v := range nm {
		s = append(s, fmt.Sprintf("(%v):%s", v.Type(), name))
	}
	return strings.Join(s, " ")
}
