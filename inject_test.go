package injectgo_test

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"

	"github.com/RivenZoo/injectgo"
	"github.com/stretchr/testify/assert"
)

type Person struct {
	Name string
}

func (p Person) String() string {
	return fmt.Sprintf("name:%s", p.Name)
}

func TestInjectFields(t *testing.T) {
	c := injectgo.NewContainer()

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
	c = injectgo.NewContainer()
	na := &NamedA{}

	c.Provide(na, b, b)
	assert.Panics(t, func() {
		c.Populate(nil)
	}, "should panic because no named value provided")

	na = &NamedA{}
	c = injectgo.NewContainer()
	c.Provide(na, b)
	c.ProvideByName("NameB", b)
	c.Populate(nil)
	assert.Equal(t, b, na.B)
	assert.Equal(t, b, na.UnnamedB)

	na = &NamedA{}
	c = injectgo.NewContainer()
	c.Provide(na)
	c.ProvideByName("NameB", b)
	c.Populate(nil)
	// if no unnamed object provided, an empty object will be created
	assert.Equal(t, &B{}, na.UnnamedB)
}

func TestInjectFunctions(t *testing.T) {
	c := injectgo.NewContainer()

	type B struct {
		Name string
	}

	type A struct {
		B        *B           `inject:""`
		Stringer fmt.Stringer `inject:""`
	}

	var a *A
	var b *B

	c.ProvideFunc(injectgo.InjectFunc{
		Fn: func() (*B, error) {
			b = &B{"generated b"}
			return b, nil
		},
	}, injectgo.InjectFunc{
		Fn: func() *A { a = &A{}; return a },
	}, injectgo.InjectFunc{
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

	c = injectgo.NewContainer()
	na := &NamedA{}
	c.Provide(na)
	c.ProvideFuncByName("NameB", injectgo.InjectFunc{
		Fn: func() *B {
			b = &B{"generated b"}
			return b
		},
	})
	c.Populate(nil)
	assert.Equal(t, b, na.B)

	/// test function return error
	c = injectgo.NewContainer()
	assert.Panics(t, func() {
		c.ProvideFunc(injectgo.InjectFunc{
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
	c := injectgo.NewContainer()

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

	targetFunc := injectgo.InjectFunc{
		Fn: func() (*B, error) {
			b2 = &B{"generated b1"}
			return b2, nil
		},
		Label: "b2",
	}
	targetLabel := targetFunc.Label

	c.ProvideFunc(
		injectgo.InjectFunc{
			Fn: func() (*B, error) {
				b1 = &B{"generated b1"}
				return b1, nil
			},
			Label: "b1"},
		targetFunc,
		injectgo.InjectFunc{
			Fn: func() *A { a = &A{}; return a },
		},
		injectgo.InjectFunc{
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
