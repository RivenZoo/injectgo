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
	c.pushInjectedValues(va)
	assert.False(t, c.isAllFulfilled())
	assert.NotEmpty(t, c.getUnfulfilledNamedValues())
	assert.NotEmpty(t, c.getUnfulfilledUnnamedValues())
	t.Log(c.getUnfulfilledUnnamedValues(), c.getUnfulfilledNamedValues())

	vi := reflect.ValueOf(i)
	c.popFulfilledUnnamedValues(vi)
	c.pushInjectedValues(vi)
	assert.False(t, c.isAllFulfilled())
	assert.Empty(t, c.getUnfulfilledUnnamedValues())
	assert.NotEmpty(t, c.getUnfulfilledNamedValues())

	vd := reflect.ValueOf(d)
	c.popFulfilledNamedValues("desc", vd)
	c.pushInjectedValues(vd)
	assert.True(t, c.isAllFulfilled())
	assert.Empty(t, c.getUnfulfilledUnnamedValues())
	assert.Empty(t, c.getUnfulfilledNamedValues())
}
