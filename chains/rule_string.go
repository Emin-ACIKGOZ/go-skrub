// SPDX-License-Identifier: MIT

package chains

import (
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// StringRule is a goroutine-safe validation rule for string values.
// It holds only immutable state: the ChainConfig (shared across goroutines),
// the target pointer (const after construction), and the field name.
// Unlike StringChain, StringRule has no CAS guard — it can be used
// concurrently by multiple goroutines without ErrConcurrencyViolation.
type StringRule struct {
	config *core.ChainConfig
	target any
	name   string
}

// NewStringRule creates a StringRule from the given config, target, and name.
func NewStringRule(config *core.ChainConfig, target any, name string) *StringRule {
	return &StringRule{config: config, target: target, name: name}
}

// Validate executes all validators in the chain config against the bound target.
// It is goroutine-safe: no Acquire/Release guard, no mutable shared state.
func (r *StringRule) Validate(ctx *core.Context) error {
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

func (r *StringRule) resolveTarget() (string, bool, error) {
	switch t := r.target.(type) {
	case *string:
		if t == nil {
			return "", true, nil
		}
		return *t, false, nil
	case core.Valuer:
		u := t.Unwrap()
		if s, ok := u.(string); ok {
			return s, false, nil
		}
		return "", false, core.ErrMisuse
	default:
		val, isNil, err := resolveValuerIndirect(r.target)
		if err != nil || isNil {
			return "", isNil, err
		}
		if s, ok := val.(string); ok {
			return s, false, nil
		}
		return "", false, core.ErrMisuse
	}
}

func (r *StringRule) wrapError(ctx *core.Context, fe *core.FieldError) error {
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
