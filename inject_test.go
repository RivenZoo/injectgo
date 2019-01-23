package injectgo

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

type Person struct {
	Name string
}

func (p Person) String() string {
	return fmt.Sprintf("name:%s", p.Name)
}

func TestInjectFields(t *testing.T) {
	c := NewContainer()

	type B struct {
		Name string
	}

	type A struct {
		B        *B           `inject:""`
		Stringer fmt.Stringer `inject:""`
	}

	a := &A{}
	b := &B{"b"}
	c.Provide(a, b, &Person{})

	assert.Panics(t, func() {
		c.Provide(B{})
	}, "should panic because type not match")

	c.Populate(nil)
	assert.Equal(t, b, a.B)
	assert.NotEmpty(t, a.Stringer.String())

	/// test named inject
	type NamedA struct {
		B        *B `inject:"NameB"`
		UnnamedB *B `inject:""`
	}
	c = NewContainer()
	na := &NamedA{}

	c.Provide(na, b, b)
	assert.Panics(t, func() {
		c.Populate(nil)
	}, "should panic because no named value provided")

	type NamedA2 struct {
		B        *B `inject:"NameB"`
		UnnamedB *B `inject:""`
	}

	na = &NamedA{}
	na2 := &NamedA2{}
	c = NewContainer()
	c.Provide(na, b, na2)

	c.ProvideByName("NameB", b)
	c.Populate(nil)
	assert.Equal(t, b, na.B)
	assert.Equal(t, b, na.UnnamedB)
	assert.Equal(t, b, na2.B)
	assert.Equal(t, b, na2.UnnamedB)

	na = &NamedA{}
	c = NewContainer()
	c.Provide(na)
	c.ProvideByName("NameB", b)
	assert.Panics(t, func() {
		defer func() {
			i := recover()
			if e, ok := i.(error); ok {
				t.Log(e)
			}
			panic(i)
		}()
		c.Populate(nil)
	}, "should panic because no unnamed object provided")

	c.Provide(b)
	c.Populate(nil)
	assert.Equal(t, b, na.UnnamedB)
	t.Logf("%v, %v", na.UnnamedB, na.B)
}

func TestInjectFunctions(t *testing.T) {
	c := NewContainer()

	type B struct {
		Name string
	}

	type A struct {
		B        *B           `inject:""`
		Stringer fmt.Stringer `inject:""`
	}

	var a *A
	var b *B

	c.ProvideFunc(InjectFunc{
		Fn: func() (*B, error) {
			b = &B{"generated b"}
			return b, nil
		},
	}, InjectFunc{
		Fn: func() *A { a = &A{}; return a },
	}, InjectFunc{
		Fn: func() *Person { return &Person{Name: "unnamed person"} },
	})
	c.Populate(nil)

	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.Equal(t, b, a.B)
	assert.NotEmpty(t, a.Stringer.String())

	/// test named function inject
	type NamedA struct {
		B *B `inject:"NameB"`
	}

	c = NewContainer()
	na := &NamedA{}
	c.Provide(na)
	c.ProvideFuncByName("NameB", InjectFunc{
		Fn: func() *B {
			b = &B{"generated b"}
			return b
		},
	})
	c.Populate(nil)
	assert.Equal(t, b, na.B)

	/// test function return error
	c = NewContainer()
	assert.Panics(t, func() {
		c.ProvideFunc(InjectFunc{
			Fn: func() (*B, error) {
				b = &B{"generated b"}
				return b, errors.New("unknown error")
			},
		})
		c.Populate(nil)
	})
}

type labelSelector struct {
	labels []string
}

func (s labelSelector) IsLabelAllowed(label string) bool {
	for i := range s.labels {
		if s.labels[i] == label {
			return true
		}
	}
	return false
}

func TestInjectFunctions_LabelSelect(t *testing.T) {
	c := NewContainer()

	type B struct {
		Name string
	}

	type A struct {
		B        *B           `inject:""`
		Stringer fmt.Stringer `inject:""`
	}

	var a *A
	var b1 *B
	var b2 *B

	targetFunc := InjectFunc{
		Fn: func() (*B, error) {
			b2 = &B{"generated b1"}
			return b2, nil
		},
		Label: "b2",
	}
	targetLabel := targetFunc.Label

	c.ProvideFunc(
		InjectFunc{
			Fn: func() (*B, error) {
				b1 = &B{"generated b1"}
				return b1, nil
			},
			Label: "b1"},
		targetFunc,
		InjectFunc{
			Fn: func() *A { a = &A{}; return a },
		},
		InjectFunc{
			Fn: func() *Person { return &Person{Name: "unnamed person"} },
		})

	selector := labelSelector{labels: []string{targetLabel}}
	c.Populate(selector)

	assert.NotNil(t, a)
	assert.Nil(t, b1)
	assert.NotNil(t, b2)
	assert.Equal(t, b2, a.B)
	assert.NotEmpty(t, a.Stringer.String())
}

func TestSetValue(t *testing.T) {
	type B struct {
		Name string
	}
	b := B{"B"}
	var pb *B
	setFunctionReceiver(reflect.ValueOf(&b), reflect.ValueOf(&pb))

	assert.Equal(t, &b, pb)

	setFunctionReceiver(reflect.ValueOf(b), reflect.ValueOf(pb))
	assert.Equal(t, b, *pb)
}

func TestInjectFunctionsReceiver(t *testing.T) {
	c := NewContainer()

	type B struct {
		Name string
	}

	type A struct {
		B        *B           `inject:""`
		Stringer fmt.Stringer `inject:""`
	}

	var a *A
	var b *B

	c.ProvideFunc(InjectFunc{
		Fn: func() (*B, error) {
			return &B{"generated b"}, nil
		},
		Receiver: &b,
	}, InjectFunc{
		Fn:       func() *A { return &A{}; },
		Receiver: &a,
	}, InjectFunc{
		Fn: func() *Person { return &Person{Name: "unnamed person"} },
	})
	c.Populate(nil)

	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.Equal(t, b, a.B)
	assert.NotEmpty(t, a.Stringer.String())

	/// test named function inject
	type NamedA struct {
		B *B `inject:"NameB"`
	}
	b = nil

	c = NewContainer()
	na := &NamedA{}
	c.Provide(na)
	c.ProvideFuncByName("NameB", InjectFunc{
		Fn: func() *B {
			return &B{"generated b"}
		},
		Receiver: &b,
	})
	c.Populate(nil)
	assert.Equal(t, b, na.B)

	/// test function wrong receiver
	b = nil
	c = NewContainer()
	assert.Panics(t, func() {
		c.ProvideFunc(InjectFunc{
			Fn: func() (*B, error) {
				return &B{"generated b"}, nil
			},
			Receiver: b,
		})
		c.Populate(nil)
	}, "should panic because receiver should be &b")
}

type B struct {
	Name string
	A    *A `inject:""`
}

type A struct {
	B        *B           `inject:""`
	Stringer fmt.Stringer `inject:""`
}

func TestInjectFunctions_DependencyCyclic(t *testing.T) {
	c := NewContainer()

	a := &A{}
	b := &B{"b", nil}
	c.Provide(a, b, &Person{})

	assert.Panics(t, func() {
		c.Provide(B{})
	}, "should panic because type not match")

	c.Populate(nil)
	assert.Equal(t, b, a.B)
	assert.NotEmpty(t, a.Stringer.String())
	assert.Equal(t, a, b.A)

	t.Logf("%p,%p, %v, %v", a, b, a, b)
}
