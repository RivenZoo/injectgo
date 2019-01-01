package injectgo

import (
	"fmt"
	"reflect"
)

type injectChecker struct {
	unfulfilledUnnamedValues     map[reflect.Type]reflect.Value
	unfulfilledUnnamedInterfaces map[reflect.Type]reflect.Value
	unfulfilledNamedValues       map[string]reflect.Value
}

func newInjectChecker() *injectChecker {
	return &injectChecker{
		unfulfilledUnnamedValues:     make(map[reflect.Type]reflect.Value),
		unfulfilledUnnamedInterfaces: make(map[reflect.Type]reflect.Value),
		unfulfilledNamedValues:       make(map[string]reflect.Value),
	}
}

func (c *injectChecker) pushInjectedValues(obj reflect.Value) {
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
	if _, ok := c.unfulfilledNamedValues[name]; ok {
		delete(c.unfulfilledNamedValues, name)
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
