// SPDX-License-Identifier: MIT

package chains

import (
	"reflect"
	"sync"

	safeReflect "github.com/Emin-ACIKGOZ/go-skrub/internal/skrubreflect"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// SliceRule is a goroutine-safe validation rule for slice values.
// Flyweight rules are pre-allocated once under a mutex and reused
// via SetTarget() for all subsequent element iterations.
type SliceRule struct {
	config           *core.ChainConfig
	target           any
	name             string
	elementTemplates []core.Template
	mu               sync.Mutex
	rules            []core.Rule
	rebindable       []core.Rebindable
	initialized      bool
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
	tmplCount := len(r.elementTemplates)
	if tmplCount == 0 {
		return nil
	}

	// Lazy pre-allocation: initialize flyweight rules once, protected by mutex.
	// Subsequent calls use SetTarget() on the pre-bound rules (zero alloc per element).
	if !r.initialized {
		r.mu.Lock()
		if !r.initialized {
			r.rules = make([]core.Rule, tmplCount)
			r.rebindable = make([]core.Rebindable, tmplCount)
			r.initialized = true
		}
		r.mu.Unlock()
	}

	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i)
		if elem.Kind() == reflect.Ptr && elem.IsNil() {
			continue
		}

		if err := ctx.PushIndex(i); err != nil {
			return err
		}

		bindTarget := safeReflect.ResolveValue(elem)
		for j, tmpl := range r.elementTemplates {
			if r.rules[j] == nil {
				// First element: Bind to create the flyweight Rule
				r.rules[j] = tmpl.Bind(bindTarget, "")
				if rb, ok := r.rules[j].(core.Rebindable); ok {
					r.rebindable[j] = rb
				}
			} else if r.rebindable[j] != nil {
				// Subsequent elements: reuse via SetTarget (zero alloc)
				r.rebindable[j].SetTarget(bindTarget)
			} else {
				r.rules[j] = tmpl.Bind(bindTarget, "")
			}

			if err := r.rules[j].Validate(ctx); err != nil {
				ctx.Pop()
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
