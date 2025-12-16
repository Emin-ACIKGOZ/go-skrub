// SPDX-License-Identifier: MIT

package chains

import (
	"regexp"
	"unicode/utf8"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// StringChain is a bound validator for string values.
// It supports binding to native *string types and any custom type that
// implements the core.Valuer interface and unwraps to a string.
type StringChain struct {
	BaseChain
	target     any
	validators []func(value string) error
}

// NewStringChain initializes a new StringChain for the given target and validation name.
// The target must be a *string or a core.Valuer that yields a string; otherwise,
// Validate returns core.ErrMisuse upon execution.
func NewStringChain(target any, name string) *StringChain {
	return &StringChain{
		BaseChain: BaseChain{Name: name},
		target:    target,
	}
}

// Validate executes the configured length and pattern validators against the bound string value.
//
// It returns nil immediately if the bound target is a nil *string pointer.
// It returns core.ErrMisuse if the target is not a supported string type (*string or core.Valuer).
func (c *StringChain) Validate(ctx *core.Context) error {
	if err := c.Acquire(); err != nil {
		return err
	}
	defer c.Release()

	var val string
	var isNil bool

	switch t := c.target.(type) {
	case *string:
		if t == nil {
			isNil = true
		} else {
			val = *t
		}
	case core.Valuer:
		unwrapped := t.Unwrap()
		if s, ok := unwrapped.(string); ok {
			val = s
		} else {
			return core.ErrMisuse // Valuer did not return a string.
		}
	default:
		return core.ErrMisuse // Target is not a supported type.
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

// Reset clears the StringChain's state, returning it to the state of a freshly
// initialized chain for object pooling reuse.
func (c *StringChain) Reset() {
	c.BaseChain.Reset()
	c.target = nil
	c.validators = nil
}

// Min enforces a minimum character length (rune count) for the string.
// The minimum allowed length is validationMin.
func (c *StringChain) Min(validationMin int) *StringChain {
	c.validators = append(c.validators, func(v string) error {
		if utf8.RuneCountInString(v) < validationMin {
			return core.NewFieldError("", v, "length is less than required minimum")
		}
		return nil
	})
	return c
}

// Max enforces a maximum character length (rune count) for the string.
// The maximum allowed length is validationMax.
func (c *StringChain) Max(validationMax int) *StringChain {
	c.validators = append(c.validators, func(v string) error {
		if utf8.RuneCountInString(v) > validationMax {
			return core.NewFieldError("", v, "length exceeds maximum limit")
		}
		return nil
	})
	return c
}

// Pattern enforces that the string must match the provided regular expression.
// The pattern is compiled during the call to Pattern; for optimal performance
// in critical loops, pre-compile the regex outside the chain definition.
func (c *StringChain) Pattern(pattern string) *StringChain {
	re := regexp.MustCompile(pattern)
	c.validators = append(c.validators, func(v string) error {
		if !re.MatchString(v) {
			return core.NewFieldError("", v, "value does not match required pattern")
		}
		return nil
	})
	return c
}

// Email enforces a basic email format using a standard, non-RFC-compliant regex.
// It checks for the general structure: text@text.text.
func (c *StringChain) Email() *StringChain {
	// A simple, generally useful regex for basic email format validation.
	re := regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	c.validators = append(c.validators, func(v string) error {
		if !re.MatchString(v) {
			return core.NewFieldError("", v, "invalid email format")
		}
		return nil
	})
	return c
}

// UUID enforces the standard 8-4-4-4-12 hexadecimal string UUID format (version 1-5).
func (c *StringChain) UUID() *StringChain {
	// Standard UUID format regex, case-insensitive.
	re := regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	c.validators = append(c.validators, func(v string) error {
		if !re.MatchString(v) {
			return core.NewFieldError("", v, "invalid UUID format")
		}
		return nil
	})
	return c
}
