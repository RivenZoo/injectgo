package injectgo

import (
	"fmt"
	"reflect"
	"strings"
)

type injectField struct {
	value       reflect.Value
	fieldType   reflect.Type
	tagName     string // eg `inject: "myfield"`, set tagName to "myfield".
	isSatisfied bool   // if field is fulfilled, set it to true.
}

func (f injectField) String() string {
	return fmt.Sprintf("{%s: %v, tag: \"%s\", satisfied: %t}",
		f.fieldType, f.value, f.tagName, f.isSatisfied)
}

type injectObject struct {
	value             reflect.Value
	fields            []injectField
	unfulfilledNum    int
	isComplete        bool // true if unfulfilledNum == 0
	isMethodCallAdded bool
}

func (o *injectObject) String() string {
	s := []string{fmt.Sprintf("%s: %v", o.value.Type(), o.value)}

	fStrs := make([]string, 0, len(o.fields))
	for i := range o.fields {
		fStrs = append(fStrs, fmt.Sprintf("%s", o.fields[i]))
	}
	s = append(s, fmt.Sprintf("[%s]", strings.Join(fStrs, ", ")))
	s = append(s, fmt.Sprintf("isComplete: %t", o.isComplete))
	return fmt.Sprintf("{%s}", strings.Join(s, ", "))
}

func scanInjectFields(v reflect.Value) []injectField {
	ret := make([]injectField, 0)
	rawV := reflect.Indirect(v)
	t := rawV.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if inj, ok := field.Tag.Lookup(injectTag); ok {
			switch field.Type.Kind() {
			case reflect.Interface, reflect.Ptr:
				ret = append(ret, injectField{
					value:       rawV.Field(i),
					fieldType:   field.Type,
					tagName:     inj,
					isSatisfied: false,
				})
			default:
				panic(fmt.Errorf("field %v inject tag %s of object %v wrong type", field, inj, v))
			}
		}
	}
	return ret
}

func newInjectObject(v reflect.Value) *injectObject {
	o := &injectObject{
		value:      v,
		isComplete: false,
	}
	o.fields = scanInjectFields(v)
	o.unfulfilledNum = len(o.fields)
	if o.unfulfilledNum == 0 {
		o.isComplete = true
	}
	return o
}

func (o *injectObject) UnfulfilledFields() []injectField {
	if o.isComplete {
		return nil
	}
	return o.fields
}

func (o *injectObject) SetField(v reflect.Value, field *injectField) {
	if field.isSatisfied {
		panic(fmt.Errorf("field %v of %v is satisfied", *field, o.value.Interface()))
	}

	field.value.Set(v)
	field.isSatisfied = true
	o.unfulfilledNum--
	if o.unfulfilledNum <= 0 {
		o.isComplete = true
	}
}

type objectGraph struct {
	unnamedObjects map[reflect.Type]*injectObject
	namedObjects   map[string]*injectObject

	fulfilledUnnamedObjects map[reflect.Type]*injectObject
	fulfilledNamedObjects   map[string]*injectObject

	addedObjectsPtr map[uintptr]bool
	initObjects     []Initializable // objects need to be initialized
	closeObjects    []Closable      // objects need to be closed
}

func newObjectGraph() *objectGraph {
	return &objectGraph{
		unnamedObjects:          map[reflect.Type]*injectObject{},
		namedObjects:            map[string]*injectObject{},
		fulfilledUnnamedObjects: map[reflect.Type]*injectObject{},
		fulfilledNamedObjects:   map[string]*injectObject{},
		initObjects:             make([]Initializable, 0),
		closeObjects:            make([]Closable, 0),
		addedObjectsPtr:         make(map[uintptr]bool),
	}
}

func (g *objectGraph) addObjectCall(obj *injectObject) {
	if obj.isMethodCallAdded {
		return
	}
	obj.isMethodCallAdded = true

	if obj.value.Type().Kind() == reflect.Ptr {
		ptr := obj.value.Pointer()
		if _, ok := g.addedObjectsPtr[ptr]; ok {
			return
		}
		g.addedObjectsPtr[ptr] = true
	}

	i := obj.value.Interface()
	if initObj, ok := i.(Initializable); ok {
		g.initObjects = append(g.initObjects, initObj)
	}
	if closeObj, ok := i.(Closable); ok {
		g.closeObjects = append(g.closeObjects, closeObj)
	}
}

func (g *objectGraph) ProvideObj(obj reflect.Value) {
	injObj := newInjectObject(obj)
	if injObj.isComplete {
		g.fulfilledUnnamedObjects[obj.Type()] = injObj
		g.addObjectCall(injObj)
	} else {
		g.unnamedObjects[obj.Type()] = injObj
	}
}

func (g *objectGraph) ProvideNamedObj(name string, obj reflect.Value) {
	injObj := newInjectObject(obj)
	if injObj.isComplete {
		g.fulfilledNamedObjects[name] = injObj
		g.addObjectCall(injObj)
	} else {
		g.namedObjects[name] = injObj
	}
}

func (g *objectGraph) findMatchingObject(field *injectField) *injectObject {
	if field.isSatisfied {
		return nil
	}
	if field.tagName != "" {
		if o, ok := g.fulfilledNamedObjects[field.tagName]; ok {
			return o
		}
		if o, ok := g.namedObjects[field.tagName]; ok {
			return o
		}
		return nil
	}
	return g.findUnnamedObjectByType(field.fieldType)
}

func (g *objectGraph) findUnnamedObjectByType(tp reflect.Type) *injectObject {
	if o, ok := g.fulfilledUnnamedObjects[tp]; ok {
		return o
	}
	if o, ok := g.unnamedObjects[tp]; ok {
		return o
	}
	for t, o := range g.fulfilledUnnamedObjects {
		if t.AssignableTo(tp) {
			return o
		}
	}
	for t, o := range g.unnamedObjects {
		if t.AssignableTo(tp) {
			return o
		}
	}
	return nil
}

func (g *objectGraph) Populate() {
	g.populateNamedObjects()
	g.populateUnnamedObjects()
	g.initAllObjects()
}

func (g *objectGraph) populateNamedObjects() {
	for _, injObj := range g.namedObjects {
		g.populateObject(injObj, 0)
	}
}

func (g *objectGraph) populateUnnamedObjects() {
	for _, injObj := range g.unnamedObjects {
		g.populateObject(injObj, 0)
	}
}

const maxCallDepth = 1048576

func (g *objectGraph) populateObject(obj *injectObject, depth int) {
	if depth > maxCallDepth {
		panic(fmt.Errorf("object %s call stack overflow, depth %d", obj, depth))
	}
	fields := obj.UnfulfilledFields()
	for i := range fields {
		field := &fields[i]
		injObj := g.findMatchingObject(field)
		if injObj == nil {
			panic(fmt.Errorf("field (%s) of %s has no matching object", field.fieldType, obj.value))
		}

		depth++
		g.populateObject(injObj, depth)
		if injObj.isComplete {
			obj.SetField(injObj.value, field)
		}
	}
	if obj.isComplete {
		g.addObjectCall(obj)
		return
	}
	panic(fmt.Errorf("object %s not complete", obj))
}

func (g *objectGraph) initAllObjects() {
	for i := range g.initObjects {
		if err := g.initObjects[i].Init(); err != nil {
			panic(err)
		}
	}
}

func (g *objectGraph) Close() {
	// call Close method in inverse order
	for i := len(g.closeObjects) - 1; i >= 0; i-- {
		if err := g.closeObjects[i].Close(); err != nil {
			panic(err)
		}
	}
}
