// SPDX-License-Identifier: MIT

package chains

import (
	"reflect"

	safeReflect "github.com/Emin-ACIKGOZ/go-skrub/internal/skrubreflect"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// SliceRule is a goroutine-safe validation rule for slice values.
// It holds only immutable state and allocates flyweights per-call
// instead of caching them on the rule (safe for concurrent use).
type SliceRule struct {
	config           *core.ChainConfig
	target           any
	name             string
	elementTemplates []core.Template
}

// NewSliceRule creates a SliceRule from the given config, target, name,
// and element templates for recursive validation.
func NewSliceRule(config *core.ChainConfig, target any, name string, elementTemplates []core.Template) *SliceRule {
	return &SliceRule{
		config:           config,
		target:           target,
		name:             name,
		elementTemplates: elementTemplates,
	}
}

// Validate executes all slice-level validators and recursively validates
// each element. It is goroutine-safe: per-call flyweight allocation.
func (r *SliceRule) Validate(ctx *core.Context) error {
	if ctx == nil {
		ctx = core.NewContext(core.Config{})
	}

	val, isNil, err := r.resolveTarget()
	if err != nil || isNil {
		return err
	}

	for _, fn := range r.config.Validators {
		if fe := fn(ctx, val.Interface()); fe != nil {
			return r.wrapError(ctx, fe)
		}
	}

	if r.name != "" {
		if err := ctx.Push(r.name); err != nil {
			return err
		}
		defer ctx.Pop()
	}

	return r.validateElements(ctx, val)
}

func (r *SliceRule) resolveTarget() (reflect.Value, bool, error) {
	val := reflect.ValueOf(r.target)
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return reflect.Value{}, true, nil
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Slice {
		return reflect.Value{}, false, core.ErrMisuse
	}
	return val, false, nil
}

func (r *SliceRule) validateElements(ctx *core.Context, val reflect.Value) error {
	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i)
		if elem.Kind() == reflect.Ptr && elem.IsNil() {
			continue
		}

		if err := ctx.PushIndex(i); err != nil {
			return err
		}

		bindTarget := safeReflect.ResolveValue(elem)
		for _, tmpl := range r.elementTemplates {
			rule := tmpl.Bind(bindTarget, "")
			if err := rule.Validate(ctx); err != nil {
				ctx.Pop()
				// Child errors already contain the correct full path
				// (including all index segments from recursive nesting).
				// Return them as-is without rewrapping.
				return err
			}
		}
		ctx.Pop()
	}
	return nil
}

func (r *SliceRule) wrapError(ctx *core.Context, fe *core.FieldError) error {
	path := r.name
	if ctx != nil {
		if p := ctx.String(); p != "" {
			path = p
		}
	}
	return &core.FieldError{
		Path:   path,
		Value:  fe.Value,
		Reason: fe.Reason,
		Cause:  fe.Cause,
	}
}
