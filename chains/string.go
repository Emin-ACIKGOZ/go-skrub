// SPDX-License-Identifier: MIT

package chains

import (
	"net"
	"net/url"
	"regexp"
	"unicode/utf8"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// Package-level variables for globally compiled regular expressions.
// This prevents expensive recompilation on every execution of Email() or UUID().
var (
	// EmailRegex provides a generally useful, non-RFC-compliant pattern for basic email format validation.
	EmailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

	// UUIDRegex enforces the standard 8-4-4-4-12 hexadecimal string UUID format (version 1-5), case-insensitive.
	UUIDRegex = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
)

// StringChain is a bound validator for string values.
// It supports binding to native *string types and any custom type that
// implements the core.Valuer interface and unwraps to a string.
type StringChain struct {
	BaseChain
	target     any
	validators []func(string) error
}

// NewStringChain initializes a new StringChain for the given target and validation name.
func NewStringChain(target any, name string) *StringChain {
	return &StringChain{
		BaseChain: BaseChain{Name: name},
		target:    target,
	}
}

// SetTarget updates the validation target, allowing the chain to be reused via Flyweight.
func (c *StringChain) SetTarget(target any) {
	c.target = target
}

// --- Ad-hoc Builders ---

// Min enforces a minimum rune length constraint on the string value.
// If the string contains fewer than vMin runes, validation fails
// with core.ReasonMinLength.
func (c *StringChain) Min(vMin int) *StringChain {
	c.validators = append(c.validators, func(v string) error {
		if utf8.RuneCountInString(v) < vMin {
			return core.NewFieldError("", v, core.ReasonMinLength)
		}
		return nil
	})
	return c
}

// Max enforces a maximum rune length constraint on the string value.
// If the string contains more than vMax runes, validation fails
// with core.ReasonMaxLength.
func (c *StringChain) Max(vMax int) *StringChain {
	c.validators = append(c.validators, func(v string) error {
		if utf8.RuneCountInString(v) > vMax {
			return core.NewFieldError("", v, core.ReasonMaxLength)
		}
		return nil
	})
	return c
}

// Email validates that the string matches a basic email format.
// It uses a precompiled regular expression and returns
// core.ReasonInvalidEmail on failure.
func (c *StringChain) Email() *StringChain {
	c.validators = append(c.validators, func(v string) error {
		if !EmailRegex.MatchString(v) {
			return core.NewFieldError("", v, core.ReasonInvalidEmail)
		}
		return nil
	})
	return c
}

// UUID validates that the string matches the canonical 8-4-4-4-12
// hexadecimal UUID format. On mismatch, it returns
// core.ReasonInvalidUUID.
func (c *StringChain) UUID() *StringChain {
	c.validators = append(c.validators, func(v string) error {
		if !UUIDRegex.MatchString(v) {
			return core.NewFieldError("", v, core.ReasonInvalidUUID)
		}
		return nil
	})
	return c
}

// Pattern enforces that the string must match the provided regular expression.
func (c *StringChain) Pattern(re *regexp.Regexp) *StringChain {
	c.validators = append(c.validators, func(v string) error {
		if !re.MatchString(v) {
			return core.NewFieldError("", v, core.ReasonPattern)
		}
		return nil
	})
	return c
}

// URL validates that the string is a valid HTTP(S) URL.
// It accepts absolute URLs with http:// or https:// schemes.
func (c *StringChain) URL() *StringChain {
	c.validators = append(c.validators, func(v string) error {
		// Parse the URL to ensure it's well-formed
		parsed, err := url.Parse(v)
		if err != nil {
			return core.NewFieldError("", v, core.ReasonInvalidURL)
		}

		// Require http or https scheme
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return core.NewFieldError("", v, core.ReasonInvalidURL)
		}

		// Require a host
		if parsed.Host == "" {
			return core.NewFieldError("", v, core.ReasonInvalidURL)
		}

		return nil
	})
	return c
}

// IP validates that the string is a valid IP address (IPv4 or IPv6).
func (c *StringChain) IP() *StringChain {
	c.validators = append(c.validators, func(v string) error {
		if net.ParseIP(v) == nil {
			return core.NewFieldError("", v, core.ReasonInvalidIP)
		}
		return nil
	})
	return c
}

// IPv4 validates that the string is a valid IPv4 address.
func (c *StringChain) IPv4() *StringChain {
	c.validators = append(c.validators, func(v string) error {
		ip := net.ParseIP(v)
		if ip == nil || ip.To4() == nil {
			return core.NewFieldError("", v, core.ReasonInvalidIPv4)
		}
		return nil
	})
	return c
}

// IPv6 validates that the string is a valid IPv6 address.
// Accepts both pure IPv6 (e.g., ::1) and IPv4-mapped IPv6 (e.g., ::ffff:192.0.2.1).
// Rejects pure IPv4 addresses (e.g., 192.168.1.1).
func (c *StringChain) IPv6() *StringChain {
	c.validators = append(c.validators, func(v string) error {
		ip := net.ParseIP(v)
		if ip == nil {
			return core.NewFieldError("", v, core.ReasonInvalidIPv6)
		}

		// Reject pure IPv4 notation (no colons in input).
		// Accept IPv6 notation (contains colons), including IPv4-mapped (::ffff:x.x.x.x).
		// This validates input format, preventing IPv4 addresses from passing as IPv6.
		if !containsColon(v) && ip.To4() != nil {
			return core.NewFieldError("", v, core.ReasonInvalidIPv6)
		}

		return nil
	})
	return c
}

// containsColon reports whether the string contains at least one colon character.
// Used to distinguish IPv6 notation (contains colons) from IPv4 notation (dots only).
func containsColon(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return true
		}
	}
	return false
}

// NotEmpty validates that the string is not empty.
func (c *StringChain) NotEmpty() *StringChain {
	c.validators = append(c.validators, func(v string) error {
		if v == "" {
			return core.NewFieldError("", v, core.ReasonRequired)
		}
		return nil
	})
	return c
}

// Validate executes all registered validators against the bound target.
// It returns the first encountered validation error or nil if validation succeeds.
// Concurrency violations and misuse are returned as explicit errors.
func (c *StringChain) Validate(ctx *core.Context) error {
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
			return c.Fail(ctx, val, err.Error())
		}
	}
	return nil
}

func (c *StringChain) resolveTarget() (string, bool, error) {
	switch t := c.target.(type) {
	case *string:
		if t == nil {
			return "", true, nil
		}
		return *t, false, nil
	case core.Valuer:
		unwrapped := t.Unwrap()
		if s, ok := unwrapped.(string); ok {
			return s, false, nil
		}
		return "", false, core.ErrMisuse
	default:
		return "", false, core.ErrMisuse
	}
}

// Reset clears the chain state, including its target and rules,
// allowing the instance to be safely reused by the pool.
func (c *StringChain) Reset() {
	c.BaseChain.Reset()
	c.target = nil
	// Clearing slice while keeping capacity reduces future allocations.
	c.validators = c.validators[:0]
}
