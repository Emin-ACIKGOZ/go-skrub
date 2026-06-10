// SPDX-License-Identifier: MIT

package chains

import (
	"reflect"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// MapRule validates map entries by applying separate rules to keys and values.
// Matches go-validator's dive,keys/endkeys behavior.
type MapRule struct {
	keyTpl core.Template // validates keys (tags between keys and endkeys)
	valTpl core.Template // validates values (tags after endkeys)
	target any
	name   string
}

// NewMapRule creates a new MapRule.
func NewMapRule(keyTpl, valTpl core.Template, target any, name string) *MapRule {
	return &MapRule{keyTpl: keyTpl, valTpl: valTpl, target: target, name: name}
}

// Validate iterates map entries and validates each key and value.
//
//nolint:cyclop // Complexity from type dispatch, key/value validation, name push
func (r *MapRule) Validate(ctx *core.Context) error {
	if ctx == nil {
		ctx = core.NewContext(core.Config{})
	}

	val := reflect.ValueOf(r.target)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Map {
		return core.ErrMisuse
	}

	if r.name != "" {
		if err := ctx.Push(r.name); err != nil {
			return err
		}
		defer ctx.Pop()
	}

	for _, key := range val.MapKeys() {
		keyStr := key.String()
		if keyStr == "" {
			keyStr = key.Kind().String()
		}
		if err := ctx.Push(keyStr); err != nil {
			return err
		}

		// Validate key (map keys are not addressable, create a copy)
		if r.keyTpl != nil {
			keyCopy := makeAddressable(key)
			keyRule := r.keyTpl.Bind(keyCopy, "")
			if err := keyRule.Validate(ctx); err != nil {
				ctx.Pop()
				return err
			}
		}

		// Validate value (map values are not addressable, create a copy)
		if r.valTpl != nil {
			valElem := val.MapIndex(key)
			valCopy := makeAddressable(valElem)
			valRule := r.valTpl.Bind(valCopy, "")
			if err := valRule.Validate(ctx); err != nil {
				ctx.Pop()
				return err
			}
		}

		ctx.Pop()
	}
	return nil
}

// makeAddressable creates an addressable copy of a reflect.Value.
// Map keys and values are not addressable in Go, but go-skrub's rule binding
// requires addressable targets. This helper creates a copy that can be addressed.
func makeAddressable(v reflect.Value) any {
	// Create a new pointer to the same type, copy the value
	ptr := reflect.New(v.Type())
	ptr.Elem().Set(v)
	return ptr.Interface()
}
