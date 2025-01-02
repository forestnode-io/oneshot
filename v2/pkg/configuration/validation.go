package configuration

import (
	"reflect"
	"strings"
)

type validator interface {
	Validate() error
}

func validate(v reflect.Value) error {
	// This is a workaround for the fact that reflectwalk panics when calling IsZero on a zero value reflect.Value
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				if strings.Contains(err.Error(), "IsZero") {
					return
				}
			}
			panic(r)
		}
	}()

	if v.IsZero() {
		return nil
	}

	if v.CanInterface() {
		vIface := v.Interface()
		if vIface == nil {
			return nil
		}

		if validator, ok := vIface.(validator); ok {
			return validator.Validate()
		}
	}
	return nil
}

type validationWalker struct{}

// reflectwalk.ArrayWalker interface
func (w validationWalker) Array(v reflect.Value) error {
	return validate(v)
}

func (w validationWalker) ArrayElem(_ int, v reflect.Value) error {
	return validate(v)
}

// reflectwalk.InterfaceWalker interface
func (w validationWalker) Interface(v reflect.Value) error {
	return validate(v)
}

// reflectwalk.MapWalker interface
func (w validationWalker) Map(v reflect.Value) error {
	return validate(v)
}

func (w validationWalker) MapElem(_, _ reflect.Value, v reflect.Value) error {
	return validate(v)
}

// reflectwalk.PrimitiveWalker interface
func (w validationWalker) Primitive(v reflect.Value) error {
	return validate(v)
}

// reflectwalk.SliceWalker interface
func (w validationWalker) Slice(v reflect.Value) error {
	return validate(v)
}

func (w validationWalker) SliceElem(_ int, v reflect.Value) error {
	return validate(v)
}

// reflectwalk.StructWalker interface
func (w validationWalker) Struct(v reflect.Value) error {
	return validate(v)
}

func (w validationWalker) StructField(_ reflect.StructField, v reflect.Value) error {
	return validate(v)
}
