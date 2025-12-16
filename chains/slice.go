// SPDX-License-Identifier: MIT

package chains

import (
	"reflect"
	"strconv"

	safeReflect "github.com/Emin-ACIKGOZ/go-skrub/internal/skrubreflect"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// SliceChain is a reflection-based, bound validator for slice types.
// It iterates over elements using reflection and supports recursive validation
// through nested Templates applied to each element.
type SliceChain struct {
	BaseChain
	target any

	sliceValidators  []func(v reflect.Value) error
	elementTemplates []core.Template
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
// each element using the configured element templates.
//
// If the bound target is a nil pointer or a nil interface, Validate returns nil
// immediately, skipping all further validation.
// Validate returns core.ErrMisuse if the target value is not a slice after
// resolving all pointers and interfaces.
func (c *SliceChain) Validate(ctx *core.Context) error {
	if err := c.Acquire(); err != nil {
		return err
	}
	defer c.Release()

	val := reflect.ValueOf(c.target)

	// Reliably unwrap pointers/interfaces and handle untyped nil.
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil // Allow nil pointers/interfaces to skip validation.
		}
		val = val.Elem()
	}

	// Handle reflect.Invalid kind, which can occur if the resolved element is a nil slice.
	if val.Kind() == reflect.Invalid {
		return nil
	}

	if val.Kind() != reflect.Slice {
		// Target was not a slice when bound, indicating misuse of the chain.
		return core.ErrMisuse
	}

	// Execute slice-level validators (e.g., MinLen, MaxLen).
	for _, fn := range c.sliceValidators {
		if err := fn(val); err != nil {
			return c.Fail(ctx, val.Interface(), err.Error())
		}
	}

	return c.validateElements(ctx, val)
}

// validateElements iterates through slice items and applies recursive templates.
// Returns nil if no element templates are configured or if the slice is empty.
func (c *SliceChain) validateElements(ctx *core.Context, val reflect.Value) error {
	if len(c.elementTemplates) == 0 {
		return nil
	}

	// Ensure a non-nil context is used for pathing and configuration, even if the user
	// did not provide one (e.g., if Validate is called directly).
	if ctx == nil {
		ctx = core.NewContext(core.Config{})
	}

	count := val.Len()
	for i := 0; i < count; i++ {
		elementValue := val.Index(i)

		if c.shouldSkipElement(elementValue) {
			continue
		}

		if err := c.validateElement(ctx, elementValue, i); err != nil {
			return err
		}
	}

	return nil
}

// shouldSkipElement checks if the given element value represents a nil pointer
// or an invalid resolved target that should not be validated recursively.
func (c *SliceChain) shouldSkipElement(elementValue reflect.Value) bool {
	// Skip validation if the element is a nil pointer (e.g., *string in [](*string)).
	if elementValue.Kind() == reflect.Ptr && elementValue.IsNil() {
		return true
	}

	// Resolve the value to its underlying concrete type.
	// If the resolved value is invalid (e.g., a nil interface after Elem()), skip it.
	bindTargetValue := reflect.ValueOf(safeReflect.ResolveValue(elementValue))
	return !bindTargetValue.IsValid()
}

// validateElement handles context creation, binding, and validation for a single slice item.
func (c *SliceChain) validateElement(ctx *core.Context, elementValue reflect.Value, index int) error {
	indexStr := "[" + strconv.Itoa(index) + "]"

	childCtx := c.createChildContext(ctx, indexStr)

	// The target for binding is the resolved interface value, ensuring Templates
	// operate on the concrete type, not the container (e.g., an interface or pointer).
	bindTarget := safeReflect.ResolveValue(elementValue)

	for _, tmpl := range c.elementTemplates {
		// The inner rule is bound to the element, not the chain target.
		rule := tmpl.Bind(bindTarget, "")
		if err := rule.Validate(childCtx); err != nil {
			return err
		}
	}
	return nil
}

// createChildContext handles the logic for path resolution and context depth checking.
// It ensures that array index paths (e.g., [0]) are correctly nested within the
// parent chain's path.
func (c *SliceChain) createChildContext(ctx *core.Context, indexStr string) *core.Context {
	// If the parent context path is empty and the chain has a name, initialize the path.
	// This ensures a correct root path like "MySlice[0]" instead of "[0]".
	if ctx.Path == "" && c.Name != "" {
		effectiveCtx := core.NewContext(ctx.Cfg)
		effectiveCtx.Path = c.Name
		ctx = effectiveCtx
	}

	// Use the effective context to enter the index path.
	// The potential core.ErrRecursionDepth is propagated from the child rule's Validate call.
	childCtx, _ := ctx.Enter(indexStr)
	return childCtx
}

// Reset clears the SliceChain's state, returning it to the state of a freshly
// initialized chain for object pooling reuse.
func (c *SliceChain) Reset() {
	c.BaseChain.Reset()
	c.target = nil
	c.sliceValidators = nil
	c.elementTemplates = nil
}

// MinLen enforces a minimum number of elements in the slice.
// If the slice length is less than validationMin, validation fails.
func (c *SliceChain) MinLen(validationMin int) *SliceChain {
	c.sliceValidators = append(c.sliceValidators, func(v reflect.Value) error {
		if v.Len() < validationMin {
			return core.NewFieldError("", v.Len(), "slice length is less than minimum")
		}
		return nil
	})
	return c
}

// MaxLen enforces a maximum number of elements in the slice.
// If the slice length exceeds validationMax, validation fails.
func (c *SliceChain) MaxLen(validationMax int) *SliceChain {
	c.sliceValidators = append(c.sliceValidators, func(v reflect.Value) error {
		if v.Len() > validationMax {
			return core.NewFieldError("", v.Len(), "slice length exceeds maximum")
		}
		return nil
	})
	return c
}

// Elements applies the provided Templates to every item in the bound slice.
// This enables recursive validation of complex or nested structures within the slice.
// Validation on an element is skipped if the element is a nil pointer or resolves to nil.
func (c *SliceChain) Elements(templates ...core.Template) *SliceChain {
	c.elementTemplates = append(c.elementTemplates, templates...)
	return c
}
