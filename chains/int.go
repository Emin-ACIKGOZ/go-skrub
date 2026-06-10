// SPDX-License-Identifier: MIT

package chains

import (
	"regexp"
	"strconv"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// IntChain is a bound validator for integer values.
// It supports native *int and custom types implementing core.Valuer.
type IntChain struct {
	BaseChain
	target     any
	validators []func(int) error
}

// NewIntChain initializes a new IntChain for the given target.
func NewIntChain(target any, name string) *IntChain {
	return &IntChain{
		BaseChain: BaseChain{Name: name},
		target:    target,
	}
}

// SetTarget updates the validation target, supporting the Flyweight pattern.
func (c *IntChain) SetTarget(target any) {
	c.target = target
}

// Validate executes the configured validation logic.
func (c *IntChain) Validate(ctx *core.Context) error {
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

	// Push chain name to context stack for proper path tracking.
	if c.Name != "" {
		if err := ctx.Push(c.Name); err != nil {
			return err
		}
		defer ctx.Pop()
	}

	for _, fn := range c.validators {
		if err := fn(val); err != nil {
			if e := c.emitError(ctx, val, err.Error()); e != nil {
				return e
			}
		}
	}

	return nil
}

// resolveTarget extracts the integer value while protecting against nil pointers.
func (c *IntChain) resolveTarget() (int, bool, error) {
	switch t := c.target.(type) {
	case *int:
		if t == nil {
			return 0, true, nil
		}
		return *t, false, nil
	case core.Valuer:
		unwrapped := t.Unwrap()
		if v, ok := unwrapped.(int); ok {
			return v, false, nil
		}
		return 0, false, core.ErrMisuse
	default:
		// Reflection-based fallback: handle *T where T implements core.Valuer
		val, isNil, err := resolveValuerIndirect(c.target)
		if err != nil || isNil {
			return 0, isNil, err
		}
		if v, ok := val.(int); ok {
			return v, false, nil
		}
		return 0, false, core.ErrMisuse
	}
}

// CompileIntConfig applies the given modifiers to a temporary IntChain
// and extracts the compiled validators into an immutable ChainConfig.
func CompileIntConfig(modifiers []func(*IntChain)) *core.ChainConfig {
	tmp := &IntChain{
		validators: make([]func(int) error, 0, len(modifiers)),
	}
	for _, mod := range modifiers {
		mod(tmp)
	}
	return &core.ChainConfig{
		Validators: wrapIntValidators(tmp.validators),
	}
}

func wrapIntValidators(vals []func(int) error) []func(*core.Context, any) *core.FieldError {
	result := make([]func(*core.Context, any) *core.FieldError, len(vals))
	for i, fn := range vals {
		result[i] = func(_ *core.Context, val any) *core.FieldError {
			v, ok := val.(int)
			if !ok {
				return core.NewFieldError("", val, core.ErrMisuse.Error())
			}
			if err := fn(v); err != nil {
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

// Reset clears the chain state, preparing it for SafePool reuse.
func (c *IntChain) Reset() {
	c.BaseChain.Reset()
	c.target = nil
	// Clearing slice while keeping capacity reduces future allocations.
	c.validators = c.validators[:0]
}

// --- Rule Builders ---

// Min enforces that the value is greater than or equal to the specified minimum.
func (c *IntChain) Min(validationMin int) *IntChain {
	c.validators = append(c.validators, func(v int) error {
		if v < validationMin {
			return core.NewFieldError("", v, core.ReasonMinValue)
		}
		return nil
	})
	return c
}

// Max enforces that the value is less than or equal to the specified maximum.
func (c *IntChain) Max(validationMax int) *IntChain {
	c.validators = append(c.validators, func(v int) error {
		if v > validationMax {
			return core.NewFieldError("", v, core.ReasonMaxValue)
		}
		return nil
	})
	return c
}

// NotZero validates that the integer is not zero.
func (c *IntChain) NotZero() *IntChain {
	c.validators = append(c.validators, func(v int) error {
		if v == 0 {
			return core.NewFieldError("", v, core.ReasonRequired)
		}
		return nil
	})
	return c
}

// MatchString validates that the string representation of the integer matches a regex pattern.
// If re is nil, validation always fails with core.ReasonPattern.
func (c *IntChain) MatchString(re *regexp.Regexp) *IntChain {
	c.validators = append(c.validators, func(v int) error {
		if re == nil {
			return core.NewFieldError("", v, core.ReasonPattern)
		}
		str := strconv.Itoa(v)
		if !re.MatchString(str) {
			return core.NewFieldError("", v, core.ReasonPattern)
		}
		return nil
	})
	return c
}
