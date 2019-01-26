package injectgo

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type cyclicNormal1 struct {
	Name string
}

type cyclicNormal2 struct {
	Name    string
	Normal1 *cyclicNormal1 `inject:""`
}

type cyclicStruct1 struct {
	CS2 *cyclicStruct2 `inject:""`
}

type cyclicStruct2 struct {
	CS3 *cyclicStruct3 `inject:""`
}

type cyclicStruct3 struct {
	CS1      *cyclicStruct1 `inject:""`
	Name     string
	Stringer fmt.Stringer `inject:""`
}

func TestCyclicDetect_CyclicPath(t *testing.T) {
	n1 := &cyclicNormal1{}
	n2 := &cyclicNormal2{}

	p := depPath{reflect.TypeOf(n1), reflect.TypeOf(n2)}
	cyclicPath := p.cyclicPath(reflect.TypeOf(n1))
	assert.NotNil(t, cyclicPath)
	t.Log(cyclicPath.prettify())
}

func TestCyclicDetect_addObject(t *testing.T) {
	detector := newCyclicDetector()
	detector.AddDetectObjects(reflect.ValueOf(&cyclicStruct3{}))
	assert.True(t, len(detector.typeDeps) == 1)
	tp := reflect.TypeOf(&cyclicStruct3{})
	assert.True(t, len(detector.typeDeps[tp.Elem()]) == 1)
	t.Log(detector.typeDeps)
}

func TestCyclicDetect_DetectCyclic(t *testing.T) {
	detector := newCyclicDetector()
	detector.AddDetectObjects(reflect.ValueOf(&cyclicStruct1{}),
		reflect.ValueOf(&cyclicStruct2{}),
		reflect.ValueOf(&cyclicStruct3{}))
	exists, cpath := detector.DetectCyclic()
	assert.True(t, exists)
	assert.NotNil(t, cpath)
	t.Log(cpath.prettify())
	t.Log(detector.typeDeps)

	detector = newCyclicDetector()
	detector.AddDetectObjects(reflect.ValueOf(&cyclicNormal1{}), reflect.ValueOf(&cyclicNormal2{}))
	exists, cpath = detector.DetectCyclic()
	assert.False(t, exists)
	assert.Nil(t, cpath)
}
