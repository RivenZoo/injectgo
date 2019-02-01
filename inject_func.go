package injectgo

import (
	"fmt"
	"reflect"
)

// InjectFunc contains a function to new object and label of the function.
type InjectFunc struct {
	Fn       interface{} // func() T / func() (T, error)
	Label    string      // default selected
	Receiver interface{} // *T, receive object from Fn
}

func (ifn InjectFunc) validate() {
	fn := reflect.Indirect(reflect.ValueOf(ifn.Fn))
	if fn.Type().Kind() != reflect.Func {
		panic(errValueNotFunction)
	}
	t := fn.Type()
	if t.NumIn() != 0 {
		panic(fmt.Errorf("func %v should not accept arguments", ifn))
	}
	if t.NumOut() <= 0 || t.NumOut() > 2 {
		panic(fmt.Errorf("func %v should be at most 2 return values", ifn))
	}
	if t.NumOut() == 2 {
		if t.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			panic(fmt.Errorf("func %v second return value should be error", ifn))
		}
	}
}

func (ifn InjectFunc) create() (reflect.Value, error) {
	fn := reflect.Indirect(reflect.ValueOf(ifn.Fn))
	ret := fn.Call(nil)
	if len(ret) == 1 {
		return ret[0], nil
	}
	if len(ret) == 2 {
		var err error
		if !ret[1].IsNil() {
			err = ret[1].Interface().(error)
		}
		return ret[0], err
	}
	panic(fmt.Errorf("call unsupport function"))
}

func (ifn InjectFunc) setReceiver(obj reflect.Value) {
	if ifn.Receiver != nil {
		reflect.ValueOf(ifn.Receiver).Elem().Set(obj)
	}
}
