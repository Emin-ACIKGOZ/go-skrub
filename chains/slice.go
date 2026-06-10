// SPDX-License-Identifier: MIT

package chains

import (
	"reflect"

	safeReflect "github.com/Emin-ACIKGOZ/go-skrub/internal/skrubreflect"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// SliceChain is a reflection-based, bound validator for slice types.
// It iterates over elements using reflection and supports recursive validation
// through nested Templates applied to each element using the Flyweight pattern.
type SliceChain struct {
	BaseChain
	target           any
	validators       []func(reflect.Value) error
	elementTemplates []core.Template

	// Caches to prevent allocations during recursive execution.
	flyweights  []core.Rule
	rebindables []core.Rebindable
}

// NewSliceChain initializes a new SliceChain for the given target slice and validation name.
// The target must be a slice or a pointer to a slice; otherwise, Validate will return ErrMisuse.
func NewSliceChain(target any, name string) *SliceChain {
	return &SliceChain{
		BaseChain: BaseChain{Name: name},
		target:    target,
	}
}

// Validate executes all length validators on the bound slice and recursively validates
// each element. It utilizes the mutable context stack to avoid allocations during traversal.
func (c *SliceChain) Validate(ctx *core.Context) error {
	if err := c.Acquire(); err != nil {
		return err
	}
	defer c.Release()

	// If ctx is nil, create a temporary context for standalone use.
	if ctx == nil {
		ctx = core.NewContext(core.Config{})
	}

	val, isNil, err := c.resolveTarget()
	if err != nil || isNil {
		return err
	}

	// 1. Execute slice-level validators BEFORE pushing name to stack.
	for _, fn := range c.validators {
		if err := fn(val); err != nil {
			if e := c.emitError(ctx, val.Interface(), err.Error()); e != nil {
				return e
			}
		}
	}

	// 2. Push Name to stack ONLY if we are proceeding to elements.
	if c.Name != "" {
		if err := ctx.Push(c.Name); err != nil {
			return err
		}
		// Register Pop immediately after successful Push to guarantee symmetry.
		defer ctx.Pop()
	}

	return c.validateElements(ctx, val)
}

// resolveTarget unwraps pointers and interfaces to find the underlying slice.
func (c *SliceChain) resolveTarget() (reflect.Value, bool, error) {
	val := reflect.ValueOf(c.target)

	// Reliably unwrap pointers/interfaces and handle untyped nil.
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return reflect.Value{}, true, nil // Allow nil pointers/interfaces to skip validation.
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Slice {
		// Target was not a slice when bound, indicating misuse of the chain.
		return reflect.Value{}, false, core.ErrMisuse
	}

	return val, false, nil
}

// validateElements iterates through slice items and applies recursive templates.
// It uses Flyweight rule instances and rebinds them to each element to eliminate allocation churn.
func (c *SliceChain) validateElements(ctx *core.Context, val reflect.Value) error {
	tmplCount := len(c.elementTemplates)
	if tmplCount == 0 {
		return nil
	}

	// Initialize Flyweight caches ONCE per bound chain lifecycle.
	// This eliminates all array allocations during hot-path execution.
	if len(c.flyweights) != tmplCount {
		c.flyweights = make([]core.Rule, tmplCount)
		c.rebindables = make([]core.Rebindable, tmplCount)
	}

	count := val.Len()
	for i := 0; i < count; i++ {
		elementValue := val.Index(i)

		if c.shouldSkipElement(elementValue) {
			continue
		}

		// 3. Zero-allocation PushIndex.
		if err := ctx.PushIndex(i); err != nil {
			return err
		}

		err := c.executeElementRules(ctx, elementValue)
		ctx.Pop()

		if err != nil {
			return err
		}
	}

	return nil
}

// executeElementRules handles the flyweight binding and execution for a single element.
func (c *SliceChain) executeElementRules(ctx *core.Context, elementValue reflect.Value) error {
	bindTarget := safeReflect.ResolveValue(elementValue)

	for j, tmpl := range c.elementTemplates {
		if c.flyweights[j] == nil {
			// First iteration ever: Initialize the Flyweight.
			c.flyweights[j] = tmpl.Bind(bindTarget, "")
			if rb, ok := c.flyweights[j].(core.Rebindable); ok {
				c.rebindables[j] = rb
			}
		} else if c.rebindables[j] != nil {
			// Subsequent iterations: Reuse the instance via SetTarget (Zero Allocations).
			c.rebindables[j].SetTarget(bindTarget)
		} else {
			// Fallback if rule doesn't support Rebindable.
			c.flyweights[j] = tmpl.Bind(bindTarget, "")
		}

		if err := c.flyweights[j].Validate(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *SliceChain) shouldSkipElement(v reflect.Value) bool {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return true
	}
	return false
}

// CompileSliceConfig applies the given modifiers to a temporary SliceChain
// and extracts the compiled validators into an immutable ChainConfig.
func CompileSliceConfig(modifiers []func(*SliceChain)) *core.ChainConfig {
	tmp := &SliceChain{
		validators: make([]func(reflect.Value) error, 0, len(modifiers)),
	}
	for _, mod := range modifiers {
		mod(tmp)
	}
	return &core.ChainConfig{
		Validators: wrapSliceValidators(tmp.validators),
	}
}

func wrapSliceValidators(vals []func(reflect.Value) error) []func(*core.Context, any) *core.FieldError {
	result := make([]func(*core.Context, any) *core.FieldError, len(vals))
	for i, fn := range vals {
		result[i] = func(_ *core.Context, val any) *core.FieldError {
			if err := fn(reflect.ValueOf(val)); err != nil {
				if fe, ok := err.(*core.FieldError); ok {
					return fe
				}
				return core.NewFieldError("", val, err.Error())
			}
			return nil
		}
	}
	return result
}

// SetTarget updates the validation target, allowing the chain to be reused.
func (c *SliceChain) SetTarget(target any) {
	c.target = target
}

// Reset clears the SliceChain's state for object pooling reuse.
func (c *SliceChain) Reset() {
	c.BaseChain.Reset()
	c.target = nil
	// Clearing slice while keeping capacity reduces future allocations.
	c.validators = c.validators[:0]
	c.elementTemplates = c.elementTemplates[:0]

	// Clear caches while preserving capacity to avoid re-allocation on pool reuse.
	c.flyweights = c.flyweights[:0]
	c.rebindables = c.rebindables[:0]
}

// MinLen enforces a minimum number of elements in the slice.
func (c *SliceChain) MinLen(vMin int) *SliceChain {
	c.validators = append(c.validators, func(v reflect.Value) error {
		if v.Len() < vMin {
			return core.NewFieldError("", v.Interface(), core.ReasonMinLength)
		}
		return nil
	})
	return c
}

// MaxLen enforces a maximum number of elements in the slice.
func (c *SliceChain) MaxLen(vMax int) *SliceChain {
	c.validators = append(c.validators, func(v reflect.Value) error {
		if v.Len() > vMax {
			return core.NewFieldError("", v.Interface(), core.ReasonMaxLength)
		}
		return nil
	})
	return c
}

// NotEmpty validates that the slice is not empty.
func (c *SliceChain) NotEmpty() *SliceChain {
	c.validators = append(c.validators, func(v reflect.Value) error {
		if v.Len() == 0 {
			return core.NewFieldError("", v.Interface(), core.ReasonRequired)
		}
		return nil
	})
	return c
}

// Elements applies the provided Templates to every item in the bound slice.
func (c *SliceChain) Elements(templates ...core.Template) *SliceChain {
	c.elementTemplates = append(c.elementTemplates, templates...)
	return c
}
