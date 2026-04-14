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

	val, isNil, err := c.resolveTarget()
	if err != nil || isNil {
		return err
	}

	for _, fn := range c.validators {
		if err := fn(val); err != nil {
			// FieldError path is managed by the Context/Recorder stack.
			return c.Fail(ctx, val, err.Error())
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
		return 0, false, core.ErrMisuse
	}
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
func (c *IntChain) MatchString(re *regexp.Regexp) *IntChain {
	c.validators = append(c.validators, func(v int) error {
		str := strconv.Itoa(v)
		if !re.MatchString(str) {
			return core.NewFieldError("", v, core.ReasonPattern)
		}
		return nil
	})
	return c
}
