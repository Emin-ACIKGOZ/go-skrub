// SPDX-License-Identifier: MIT

package chains

import (
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// IntRule is a goroutine-safe validation rule for integer values.
// It holds only immutable state: the ChainConfig (shared across goroutines),
// the target pointer (const after construction), and the field name.
type IntRule struct {
	config *core.ChainConfig
	target any
	name   string
}

// NewIntRule creates an IntRule from the given config, target, and name.
func NewIntRule(config *core.ChainConfig, target any, name string) *IntRule {
	return &IntRule{config: config, target: target, name: name}
}

// Validate executes all validators in the chain config against the bound target.
// It is goroutine-safe: no Acquire/Release guard, no mutable shared state.
func (r *IntRule) Validate(ctx *core.Context) error {
	if ctx == nil {
		ctx = core.NewContext(core.Config{})
	}

	val, isNil, err := r.resolveTarget()
	if err != nil || isNil {
		return err
	}

	if r.name != "" {
		if err := ctx.Push(r.name); err != nil {
			return err
		}
		defer ctx.Pop()
	}

	for _, fn := range r.config.Validators {
		if fe := fn(ctx, val); fe != nil {
			return r.wrapError(ctx, fe)
		}
	}
	return nil
}

func (r *IntRule) resolveTarget() (int, bool, error) {
	switch t := r.target.(type) {
	case *int:
		if t == nil {
			return 0, true, nil
		}
		return *t, false, nil
	case core.Valuer:
		u := t.Unwrap()
		if v, ok := u.(int); ok {
			return v, false, nil
		}
		return 0, false, core.ErrMisuse
	default:
		val, isNil, err := resolveValuerIndirect(r.target)
		if err != nil || isNil {
			return 0, isNil, err
		}
		if v, ok := val.(int); ok {
			return v, false, nil
		}
		return 0, false, core.ErrMisuse
	}
}

func (r *IntRule) wrapError(ctx *core.Context, fe *core.FieldError) error {
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
