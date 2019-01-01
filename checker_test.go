package injectgo

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type ADesc struct {
}

func (d *ADesc) String() string {
	return "this is A"
}
func TestInjectChecker(t *testing.T) {
	c := newInjectChecker()

	type Info struct {
		Name string
	}
	type A struct {
		Info *Info        `inject:""`
		Desc fmt.Stringer `inject:"desc"`
	}

	a := &A{}
	i := &Info{}
	d := &ADesc{}

	va := reflect.ValueOf(a)
	c.popFulfilledUnnamedValues(va)
	c.pushInjectedFields(va)
	c.popRemainedValues()
	assert.False(t, c.isAllFulfilled())
	assert.NotEmpty(t, c.getUnfulfilledNamedValues())
	assert.NotEmpty(t, c.getUnfulfilledUnnamedValues())
	t.Log(c.getUnfulfilledUnnamedValues(), c.getUnfulfilledNamedValues())

	vi := reflect.ValueOf(i)
	c.popFulfilledUnnamedValues(vi)
	c.pushInjectedFields(vi)
	c.popRemainedValues()
	assert.False(t, c.isAllFulfilled())
	assert.Empty(t, c.getUnfulfilledUnnamedValues())
	assert.NotEmpty(t, c.getUnfulfilledNamedValues())

	vd := reflect.ValueOf(d)
	c.popFulfilledNamedValues("desc", vd)
	c.pushInjectedFields(vd)
	c.popRemainedValues()
	assert.True(t, c.isAllFulfilled())
	assert.Empty(t, c.getUnfulfilledUnnamedValues())
	assert.Empty(t, c.getUnfulfilledNamedValues())

	a = &A{}
	i = &Info{}
	d = &ADesc{}

	vi = reflect.ValueOf(i)
	c.popFulfilledUnnamedValues(vi)
	c.pushInjectedFields(vi)
	va = reflect.ValueOf(a)
	c.popFulfilledUnnamedValues(va)
	c.pushInjectedFields(va)
	vd = reflect.ValueOf(d)
	c.popFulfilledNamedValues("desc", vd)
	c.pushInjectedFields(vd)
	c.popRemainedValues()

	assert.True(t, c.isAllFulfilled())
	assert.Empty(t, c.getUnfulfilledUnnamedValues())
	assert.Empty(t, c.getUnfulfilledNamedValues())
}
