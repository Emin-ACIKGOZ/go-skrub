// SPDX-License-Identifier: MIT

package chains

import (
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// IntChain is a bound validator for integer values.
// It supports both native *int and custom types implementing the Valuer interface.
type IntChain struct {
	BaseChain
	target     any
	validators []func(value int) error
}

// NewIntChain initializes a new IntChain for the given target.
func NewIntChain(target any, name string) *IntChain {
	return &IntChain{
		BaseChain: BaseChain{Name: name},
		target:    target,
	}
}

// Validate executes the configured validation logic.
// It safely handles nil pointers and type mismatches without panicking.
func (c *IntChain) Validate(ctx *core.Context) error {
	if err := c.Acquire(); err != nil {
		return err
	}
	defer c.Release()

	var val int
	var isNil bool

	switch t := c.target.(type) {
	case *int:
		if t == nil {
			isNil = true
		} else {
			val = *t
		}
	case core.Valuer:
		unwrapped := t.Unwrap()
		if v, ok := unwrapped.(int); ok {
			val = v
		} else {
			return core.ErrMisuse
		}
	default:
		return core.ErrMisuse
	}

	if isNil {
		return nil
	}

	for _, fn := range c.validators {
		if err := fn(val); err != nil {
			return c.Fail(ctx, val, err.Error())
		}
	}

	return nil
}

// Reset clears the chain state for pooling reuse.
func (c *IntChain) Reset() {
	c.BaseChain.Reset()
	c.target = nil
	c.validators = nil
}

// Min enforces that the value is greater than or equal to the specified minimum.
func (c *IntChain) Min(validationMin int) *IntChain {
	c.validators = append(c.validators, func(v int) error {
		if v < validationMin {
			return core.NewFieldError("", v, "value is less than minimum")
		}
		return nil
	})
	return c
}

// Max enforces that the value is less than or equal to the specified maximum.
func (c *IntChain) Max(validationMax int) *IntChain {
	c.validators = append(c.validators, func(v int) error {
		if v > validationMax {
			return core.NewFieldError("", v, "value exceeds maximum limit")
		}
		return nil
	})
	return c
}
