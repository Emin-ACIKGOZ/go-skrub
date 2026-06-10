// SPDX-License-Identifier: MIT

package chains

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// StructFieldRule binds a field name to its validation Rule.
type StructFieldRule struct {
	Name string
	Rule core.Rule
}

// StructChain is a stateless validator that walks struct fields.
// It is created fresh per Bind() call and does NOT embed BaseChain
// (no CAS guard). Create one StructChain per goroutine.
type StructChain struct {
	target           reflect.Value
	fields           []StructFieldRule
	structValidators []func(core.StructLevel) error
}

// NewStructChain creates a new StructChain.
func NewStructChain(target reflect.Value, fields []StructFieldRule, structValidators []func(core.StructLevel) error) *StructChain {
	return &StructChain{
		target:           target,
		fields:           fields,
		structValidators: structValidators,
	}
}

// Validate walks all registered fields and runs their validation rules.
// In accumulate mode, errors are collected; otherwise, the first error
// short-circuits validation.
func (c *StructChain) Validate(ctx *core.Context) error {
	if ctx == nil {
		ctx = core.NewContext(core.Config{})
	}
	if !c.target.IsValid() {
		return nil
	}

	if err := c.validateFields(ctx); err != nil {
		return err
	}
	if err := c.runStructValidators(ctx); err != nil {
		return err
	}
	if ctx.IsAccumulating() && ctx.HasErrors() {
		return ctx.Errors()
	}
	return nil
}

// validateFields iterates all registered field rules and collects or returns errors.
func (c *StructChain) validateFields(ctx *core.Context) error {
	for _, f := range c.fields {
		if err := ctx.Push(f.Name); err != nil {
			return err
		}

		err := f.Rule.Validate(ctx)
		ctx.Pop()

		if err != nil {
			if ctx.IsAccumulating() {
				c.recordFieldError(ctx, err)
				continue
			}
			return err
		}
	}
	return nil
}

// recordFieldError records a field error in the context for accumulation.
func (c *StructChain) recordFieldError(ctx *core.Context, err error) {
	if fe, ok := err.(*core.FieldError); ok {
		ctx.RecordError(fe)
	} else {
		ctx.RecordError(&core.FieldError{
			Path:   ctx.String(),
			Value:  nil,
			Reason: err.Error(),
		})
	}
}

// runStructValidators executes all struct-level validators.
// Returns the first error in short-circuit mode, or nil if errors were accumulated.
func (c *StructChain) runStructValidators(ctx *core.Context) error {
	if len(c.structValidators) == 0 {
		return nil
	}
	sl := &structLevelImpl{target: c.target, ctx: ctx}
	for _, fn := range c.structValidators {
		if err := fn(sl); err != nil {
			return err
		}
	}
	// In short-circuit mode, surface the first ReportError immediately.
	if !ctx.IsAccumulating() && ctx.HasErrors() {
		return ctx.Errors()[0]
	}
	return nil
}

// structLevelImpl implements core.StructLevel.
type structLevelImpl struct {
	target reflect.Value
	ctx    *core.Context
}

func (sl *structLevelImpl) FieldValue(name string) (any, error) {
	if !sl.target.IsValid() {
		return nil, errors.New("skrub: struct target is nil")
	}
	fv := sl.target.FieldByName(name)
	if !fv.IsValid() {
		return nil, fmt.Errorf("skrub: field %q not found", name)
	}
	return fv.Interface(), nil
}

func (sl *structLevelImpl) ReportError(path, reason string) {
	if sl.ctx != nil {
		sl.ctx.RecordError(&core.FieldError{
			Path:   path,
			Reason: reason,
		})
	}
}

func (sl *structLevelImpl) Context() *core.Context {
	return sl.ctx
}
