package injectgo

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"reflect"
	"testing"
)

func TestInjectObject(t *testing.T) {
	type testStruct struct {
		Err error         `inject:""`
		Buf *bytes.Buffer `inject:"buf"`
	}
	v := &testStruct{}
	injObj := newInjectObject(reflect.ValueOf(v))

	assert.True(t, len(injObj.fields) == injObj.unfulfilledNum)
	assert.True(t, injObj.unfulfilledNum == 2)
	assert.False(t, injObj.isComplete)

	t.Log(injObj)
	t.Log(injObj.fields[0])
	t.Log(injObj.fields[1])
	t.Log(injObj.fields[0].fieldType)
	t.Log(injObj.fields[1].fieldType)

	values := []reflect.Value{reflect.ValueOf(fmt.Errorf("error msg")),
		reflect.ValueOf(bytes.NewBuffer([]byte("test")))}
	f := injObj.UnfulfilledFields()
	for i := 0; i < len(f); i++ {
		injObj.SetField(values[i], &f[i])
	}
	assert.True(t, injObj.unfulfilledNum == 0)
	assert.True(t, injObj.isComplete)

	t.Log(v)
	t.Log(injObj)
	t.Log(injObj.fields[0])
	t.Log(injObj.fields[1])

	assert.Panics(t, func() {
		injObj.SetField(values[0], &injObj.fields[0])
	}, "should panic because set fulfilled field")
}

func TestInjectObjectGraph(t *testing.T) {
	type testStruct struct {
		Err error         `inject:""`
		Buf *bytes.Buffer `inject:"buf"`
	}
	v := &testStruct{}

	g := newObjectGraph()
	g.ProvideObj(reflect.ValueOf(v))
	g.ProvideObj(reflect.ValueOf(fmt.Errorf("test error")))
	g.ProvideNamedObj("buf", reflect.ValueOf(bytes.NewBuffer([]byte("test buf"))))

	g.Populate()

	assert.True(t, v.Err != nil)
	assert.True(t, v.Buf != nil)

	t.Log(v)

	g.Close()
}

type initCloseCounter struct {
	initCnt  int
	closeCnt int
}

type structA struct {
	cnt initCloseCounter
	B   *structB `inject:""`
}

func (a *structA) Init() error {
	log.Println("init A")
	a.cnt.initCnt++
	return nil
}

func (a *structA) Close() error {
	log.Println("close A")
	a.cnt.closeCnt++
	return nil
}

type structB struct {
	cnt initCloseCounter
	C   *structC `inject:""`
}

func (b *structB) Init() error {
	log.Println("init B")
	b.cnt.initCnt++
	return nil
}

func (b *structB) Close() error {
	log.Println("close B")
	b.cnt.closeCnt++
	return nil
}

type structC struct {
	cnt initCloseCounter
}

func (c *structC) Init() error {
	log.Println("init C")
	c.cnt.initCnt++
	return nil
}

func (c *structC) Close() error {
	log.Println("close C")
	c.cnt.closeCnt++
	return nil
}

func TestObjectGraph_InitClose(t *testing.T) {
	g := newObjectGraph()

	a := &structA{}
	b := &structB{}
	c := &structC{}
	g.ProvideObj(reflect.ValueOf(a))
	g.ProvideObj(reflect.ValueOf(b))
	g.ProvideObj(reflect.ValueOf(c))

	g.Populate()
	assert.True(t, a.cnt.initCnt == 1)
	assert.True(t, b.cnt.initCnt == 1)
	assert.True(t, c.cnt.initCnt == 1)

	t.Log(a, b, c)

	g.Close()
	assert.True(t, a.cnt.closeCnt == 1)
	assert.True(t, b.cnt.closeCnt == 1)
	assert.True(t, c.cnt.closeCnt == 1)

	t.Log(a, b, c)
}
