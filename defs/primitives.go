// SPDX-License-Identifier: MIT

package defs

import (
	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// StringDef is an unbound template for string validation.
// It stores configuration modifiers to be applied when the template is bound to a target.
type StringDef struct {
	modifiers []func(*chains.StringChain)
}

// NewStringDef creates and returns a new unbound string validation template.
func NewStringDef() *StringDef {
	return &StringDef{
		modifiers: make([]func(*chains.StringChain), 0),
	}
}

// Bind creates a stateful, bound StringChain for the specific target value and
// applies all queued validation rules (modifiers) to the chain.
// Bind returns the resulting core.Rule ready for execution via Validate.
func (d *StringDef) Bind(target any, name string) core.Rule {
	chain := chains.NewStringChain(target, name)

	for _, mod := range d.modifiers {
		mod(chain)
	}

	return chain
}

// Min queues a minimum character length (rune count) check to the template.
// The minimum allowed length is validationMin.
func (d *StringDef) Min(validationMin int) *StringDef {
	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.Min(validationMin)
	})
	return d
}

// Max queues a maximum character length (rune count) check to the template.
// The maximum allowed length is validationMax.
func (d *StringDef) Max(validationMax int) *StringDef {
	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.Max(validationMax)
	})
	return d
}

// Email queues an email format check to the template.
// The check uses a basic structural regex.
func (d *StringDef) Email() *StringDef {
	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.Email()
	})
	return d
}

// UUID queues a standard UUID format check (8-4-4-4-12 hex string) to the template.
func (d *StringDef) UUID() *StringDef {
	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.UUID()
	})
	return d
}

// Pattern queues a regular expression check to the template.
// The string must match the provided pattern.
func (d *StringDef) Pattern(pattern string) *StringDef {
	d.modifiers = append(d.modifiers, func(c *chains.StringChain) {
		c.Pattern(pattern)
	})
	return d
}

// IntDef is an unbound template for integer validation.
// It stores configuration modifiers to be applied when the template is bound to a target.
type IntDef struct {
	modifiers []func(*chains.IntChain)
}

// NewIntDef creates and returns a new unbound integer validation template.
func NewIntDef() *IntDef {
	return &IntDef{
		modifiers: make([]func(*chains.IntChain), 0),
	}
}

// Bind creates a stateful, bound IntChain for the specific target value and
// applies all queued validation rules (modifiers) to the chain.
// Bind returns the resulting core.Rule ready for execution via Validate.
func (d *IntDef) Bind(target any, name string) core.Rule {
	chain := chains.NewIntChain(target, name)

	for _, mod := range d.modifiers {
		mod(chain)
	}

	return chain
}

// Min queues a minimum value check to the template.
// The integer must be greater than or equal to validationMin.
func (d *IntDef) Min(validationMin int) *IntDef {
	d.modifiers = append(d.modifiers, func(c *chains.IntChain) {
		c.Min(validationMin)
	})
	return d
}

// Max queues a maximum value check to the template.
// The integer must be less than or equal to validationMax.
func (d *IntDef) Max(validationMax int) *IntDef {
	d.modifiers = append(d.modifiers, func(c *chains.IntChain) {
		c.Max(validationMax)
	})
	return d
}
