// SPDX-License-Identifier: MIT

package chains

import (
	"reflect"
	"sync"

	safeReflect "github.com/Emin-ACIKGOZ/go-skrub/internal/skrubreflect"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// flyweightSet holds per-goroutine pre-bound rules for slice element validation.
// Each goroutine gets its own set from the SliceRule's sync.Pool, avoiding
// repeated Bind() calls for every element.
type flyweightSet struct {
	rules      []core.Rule
	rebindable []core.Rebindable
}

// SliceRule is a goroutine-safe validation rule for slice values.
// It uses a sync.Pool of per-goroutine flyweight sets to combine
// goroutine safety with zero-alloc element validation.
type SliceRule struct {
	config           *core.ChainConfig
	target           any
	name             string
	elementTemplates []core.Template
	pool             sync.Pool // stores *flyweightSet
}

// NewSliceRule creates a SliceRule from the given config, target, name,
// and element templates for recursive validation.
func NewSliceRule(config *core.ChainConfig, target any, name string, elementTemplates []core.Template) *SliceRule {
	r := &SliceRule{
		config:           config,
		target:           target,
		name:             name,
		elementTemplates: elementTemplates,
	}
	r.pool = sync.Pool{
		New: func() any {
			return &flyweightSet{}
		},
	}
	return r
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

	// Get or create per-goroutine flyweight set from the pool
	fs := r.pool.Get().(*flyweightSet)

	// Initialize flyweights on first use by this goroutine
	if len(fs.rules) != tmplCount {
		fs.rules = make([]core.Rule, tmplCount)
		fs.rebindable = make([]core.Rebindable, tmplCount)
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
			if fs.rules[j] == nil {
				// First element: Bind to create the flyweight Rule
				fs.rules[j] = tmpl.Bind(bindTarget, "")
				if rb, ok := fs.rules[j].(core.Rebindable); ok {
					fs.rebindable[j] = rb
				}
			} else if fs.rebindable[j] != nil {
				// Subsequent elements: reuse via SetTarget (zero alloc)
				fs.rebindable[j].SetTarget(bindTarget)
			} else {
				// Fallback: rebind if the rule doesn't support Rebindable
				fs.rules[j] = tmpl.Bind(bindTarget, "")
			}

			if err := fs.rules[j].Validate(ctx); err != nil {
				r.pool.Put(fs)
				ctx.Pop()
				return err
			}
		}
		ctx.Pop()
	}

	r.pool.Put(fs)
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
