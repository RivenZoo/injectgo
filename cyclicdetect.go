package injectgo

import (
	"fmt"
	"reflect"
	"strings"
)

type typeSet map[reflect.Type]bool
type depPath []reflect.Type

type cyclicDetector struct {
	typeDeps map[reflect.Type]typeSet
}

func newCyclicDetector() *cyclicDetector {
	return &cyclicDetector{
		typeDeps: map[reflect.Type]typeSet{},
	}
}

func (d *cyclicDetector) AddDetectObjects(v ...reflect.Value) {
	for i := range v {
		d.AddDetectObject(v[i])
	}
}

func (d *cyclicDetector) AddDetectObject(v reflect.Value) {
	t := reflect.Indirect(v).Type()
	// only struct need to detect cyclic
	if t.Kind() != reflect.Struct {
		return
	}
	if _, ok := d.typeDeps[t]; ok {
		return
	}
	ts := typeSet{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if inj, ok := field.Tag.Lookup(injectTag); ok {
			switch field.Type.Kind() {
			case reflect.Interface:
				// interface no need to detect cyclic
			case reflect.Ptr:
				ts[field.Type.Elem()] = true
			default:
				panic(fmt.Errorf("field %v inject tag %s of object %v wrong type", field, inj, v))
			}
		}
	}
	d.typeDeps[t] = ts
}

func (d *cyclicDetector) DetectCyclic() (cyclicExists bool, cyclic depPath) {
	for root, ts := range d.typeDeps {
		cyclicExists, cyclic = d.traverseRootType(root, ts)
		if cyclicExists {
			return
		}
	}
	return false, nil
}

func (d *cyclicDetector) traverseRootType(rootType reflect.Type, fieldTypes typeSet) (cyclicExists bool, cyclic depPath) {
	return d.traverseTypePath(rootType, fieldTypes, nil)
}

func (d *cyclicDetector) traverseTypePath(rootType reflect.Type, fieldTypes typeSet, dPath depPath) (cyclicExists bool, cyclicPath depPath) {
	if dPath == nil {
		dPath = depPath{}
	}
	newPath := append(dPath, rootType)
	for fieldType := range fieldTypes {
		cpath := newPath.cyclicPath(fieldType)
		if cpath != nil {
			return true, cpath
		}
		if nextFieldTypes, ok := d.typeDeps[fieldType]; ok {
			cyclicExists, cyclicPath = d.traverseTypePath(fieldType, nextFieldTypes, newPath)
			if cyclicExists {
				return
			}
		}
	}
	return false, nil
}

// cyclicPath return [t1,t2,t3,t1] if t1 already in dependency path.
// return nil if tp not exists
func (p depPath) cyclicPath(tp reflect.Type) depPath {
	if p == nil {
		return p
	}
	for i := range p {
		if p[i] == tp {
			return append(p[i:], tp)
		}
	}
	return nil
}

func (p depPath) prettify() string {
	s := make([]string, 0, len(p))
	for i := range p {
		s = append(s, fmt.Sprintf("%v", p[i]))
	}
	return fmt.Sprintf("[%s]", strings.Join(s, " -> "))
}
