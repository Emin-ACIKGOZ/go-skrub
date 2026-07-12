// SPDX-License-Identifier: MIT

package defs

import (
	"regexp"
	"sync"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// StringDef is an unbound template for string validation.
// It uses closure factories to build the rule graph once and share it across all bound instances.
type StringDef struct {
	mu        sync.Mutex
	modifiers []func(*chains.StringChain)
}

// NewStringDef creates a new unbound string template.
func NewStringDef() *StringDef {
	return &StringDef{
		modifiers: make([]func(*chains.StringChain), 0),
	}
}

// Bind creates a goroutine-safe StringRule bound to the target.
// The returned Rule can be shared across goroutines without synchronization.
// For single-goroutine use with maximum performance, use BindCAS instead.
func (d *StringDef) Bind(target any, name string) core.Rule {
	return d.BindStateless(target, name)
}

// BindStateless creates a goroutine-safe StringRule bound to the target.
// The returned Rule can be shared across goroutines without synchronization.
func (d *StringDef) BindStateless(target any, name string) *chains.StringRule {
	d.mu.Lock()
	modifiers := d.modifiers
	d.mu.Unlock()

	config := chains.CompileStringConfig(modifiers)
	return chains.NewStringRule(config, target, name)
}

// BindCAS creates a StringChain bound to the target with CAS guards.
// Unlike Bind, the returned chain is NOT goroutine-safe — concurrent use
// returns ErrConcurrencyViolation. Use this only when validation happens
// in a single goroutine and maximum throughput is required.
func (d *StringDef) BindCAS(target any, name string) *chains.StringChain {
	d.mu.Lock()
	modifiers := d.modifiers
	d.mu.Unlock()

	chain := chains.NewStringChain(target, name)
	for _, mod := range modifiers {
		mod(chain)
	}
	return chain
}

// Min enforces a minimum character length.
func (d *StringDef) Min(vMin int) *StringDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.Min(vMin)
	})
	return d
}

// Max enforces a maximum character length.
func (d *StringDef) Max(vMax int) *StringDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.Max(vMax)
	})
	return d
}

// Email enforces email format validation.
func (d *StringDef) Email() *StringDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.Email()
	})
	return d
}

// UUID enforces standard UUID format validation.
func (d *StringDef) UUID() *StringDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.UUID()
	})
	return d
}

// Pattern enforces a regex match. The regex is compiled during the Definition Phase,
// preventing recompilation during the Execution Phase.
func (d *StringDef) Pattern(pattern string) *StringDef {
	// Pre-compile regex at definition time to ensure high performance during validation.
	re := regexp.MustCompile(pattern)

	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.Pattern(re)
	})
	return d
}

// URL enforces HTTP(S) URL format validation.
// Validates that the string is a well-formed URL with http:// or https:// scheme.
func (d *StringDef) URL() *StringDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.URL()
	})
	return d
}

// IP enforces IP address format validation (IPv4 or IPv6).
func (d *StringDef) IP() *StringDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.IP()
	})
	return d
}

// IPv4 enforces IPv4 address format validation.
func (d *StringDef) IPv4() *StringDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.IPv4()
	})
	return d
}

// IPv6 enforces IPv6 address format validation.
func (d *StringDef) IPv6() *StringDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.IPv6()
	})
	return d
}

// NotEmpty enforces that the string is not empty.
func (d *StringDef) NotEmpty() *StringDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.NotEmpty()
	})
	return d
}

// IntDef is an unbound template for integer validation.
// It stores configuration modifiers to be applied when the template is bound to a target.
type IntDef struct {
	mu        sync.Mutex
	modifiers []func(*chains.IntChain)
}

// NewIntDef creates a new unbound integer template.
func NewIntDef() *IntDef {
	return &IntDef{
		modifiers: make([]func(*chains.IntChain), 0),
	}
}

// Bind creates a lightweight IntChain bound to the target.
//
// Bind creates a goroutine-safe IntRule bound to the target.
// The returned Rule can be shared across goroutines without synchronization.
// For single-goroutine use with maximum performance, use BindCAS instead.
func (d *IntDef) Bind(target any, name string) core.Rule {
	return d.BindStateless(target, name)
}

// BindStateless creates a goroutine-safe IntRule bound to the target.
func (d *IntDef) BindStateless(target any, name string) *chains.IntRule {
	d.mu.Lock()
	modifiers := d.modifiers
	d.mu.Unlock()

	config := chains.CompileIntConfig(modifiers)
	return chains.NewIntRule(config, target, name)
}

// BindCAS creates an IntChain bound to the target with CAS guards.
// NOT goroutine-safe — use only for single-goroutine validation.
func (d *IntDef) BindCAS(target any, name string) *chains.IntChain {
	d.mu.Lock()
	modifiers := d.modifiers
	d.mu.Unlock()

	chain := chains.NewIntChain(target, name)
	for _, mod := range modifiers {
		mod(chain)
	}
	return chain
}

// Min enforces a minimum value.
func (d *IntDef) Min(vMin int) *IntDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.IntChain) {
		c.Min(vMin)
	})
	return d
}

// Max enforces a maximum value.
func (d *IntDef) Max(vMax int) *IntDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.IntChain) {
		c.Max(vMax)
	})
	return d
}

// NotZero enforces that the integer is not zero.
func (d *IntDef) NotZero() *IntDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.IntChain) {
		c.NotZero()
	})
	return d
}

// MatchString enforces that the string representation of the integer matches a regex pattern.
// The pattern is compiled during the Definition Phase for optimal performance.
func (d *IntDef) MatchString(pattern string) *IntDef {
	// Pre-compile regex at definition time to ensure high performance during validation.
	re := regexp.MustCompile(pattern)

	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.IntChain) {
		c.MatchString(re)
	})
	return d
}
