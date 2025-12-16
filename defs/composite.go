// SPDX-License-Identifier: MIT

// Package defs defines reusable, unbound validation templates (Definitions).
// These templates hold a set of validation rules (modifiers) that are applied
// to a specific target value only when the Bind method is called, creating a
// stateful core.Rule (a chain).
package defs

import (
	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// SliceDef is an unbound template for slice validation.
// It stores rules for the slice itself (length) and rules for its elements
// (recursive validation).
type SliceDef struct {
	// modifiers stores closures that apply configuration (rules) to the SliceChain
	// when Bind is called.
	modifiers []func(*chains.SliceChain)

	// elementTemplate holds the first core.Template provided to Elements.
	// This is maintained for external introspection of the definition's structure.
	elementTemplate core.Template
}

// NewSliceDef creates and returns a new unbound slice validation template.
func NewSliceDef() *SliceDef {
	return &SliceDef{
		modifiers: make([]func(*chains.SliceChain), 0),
	}
}

// Bind creates a stateful, bound SliceChain for the specific target value and
// applies all queued validation rules (modifiers) to the chain.
// Bind returns the resulting core.Rule ready for execution via Validate.
func (d *SliceDef) Bind(target any, name string) core.Rule {
	chain := chains.NewSliceChain(target, name)

	for _, mod := range d.modifiers {
		mod(chain)
	}

	return chain
}

// MinLen queues a validator that enforces a minimum character length (rune count)
// for the slice. The minimum length is validationMin.
func (d *SliceDef) MinLen(validationMin int) *SliceDef {
	d.modifiers = append(d.modifiers, func(c *chains.SliceChain) {
		c.MinLen(validationMin)
	})
	return d
}

// MaxLen queues a validator that enforces a maximum character length (rune count)
// for the slice. The maximum length is validationMax.
func (d *SliceDef) MaxLen(validationMax int) *SliceDef {
	d.modifiers = append(d.modifiers, func(c *chains.SliceChain) {
		c.MaxLen(validationMax)
	})
	return d
}

// Elements queues validation rules (Templates) for the items inside the slice.
// This enables recursive validation of complex or nested structures.
// If multiple templates are provided, only the first one is stored in
// elementTemplate for introspection; all provided templates are queued for validation.
func (d *SliceDef) Elements(templates ...core.Template) *SliceDef {
	if len(templates) > 0 {
		d.elementTemplate = templates[0]
	}

	// Queue the functional instruction for the bound chain.
	d.modifiers = append(d.modifiers, func(c *chains.SliceChain) {
		c.Elements(templates...)
	})
	return d
}

// GetElementTemplate returns the core.Template used to validate elements within the slice.
// This template is primarily used for external inspection or testing of the definition's structure.
// If no element template was set via Elements, it returns nil.
func (d *SliceDef) GetElementTemplate() core.Template {
	return d.elementTemplate
}
