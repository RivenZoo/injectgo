package injectgo_test

import (
	"fmt"
	"testing"

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

	type NamedA struct {
		B *B `inject:"NameB"`
	}
	c = injectgo.NewContainer()
	na := &NamedA{}

	c.Provide(na, b)
	assert.Panics(t, func() {
		c.Populate(nil)
	}, "should panic because no named value provided")

	c = injectgo.NewContainer()
	c.Provide(na)
	c.ProvideByName("NameB", b)
	c.Populate(nil)
	assert.Equal(t, b, na.B)
}
