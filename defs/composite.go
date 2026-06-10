// SPDX-License-Identifier: MIT

// Package defs defines reusable, unbound validation templates (Definitions).
// These templates hold a set of validation rules (modifiers) that are applied
// to a specific target value only when the Bind method is called, creating a
// stateful core.Rule (a chain).
package defs

import (
	"sync"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// SliceDef is an unbound template for slice validation.
// It stores structural rules (length) and recursive element templates.
type SliceDef struct {
	mu               sync.Mutex
	modifiers        []func(*chains.SliceChain)
	elementTemplates []core.Template
}

// NewSliceDef creates and returns a new unbound slice validation template.
func NewSliceDef() *SliceDef {
	return &SliceDef{
		modifiers:        make([]func(*chains.SliceChain), 0),
		elementTemplates: make([]core.Template, 0),
	}
}

// Bind creates a lightweight SliceChain bound to the target.
// It applies length constraints and registers element templates for recursive validation.
//
// Deprecated: Use BindStateless for a goroutine-safe Rule. Bind returns a
// *SliceChain with CAS guards and will be removed in v0.6.0.
func (d *SliceDef) Bind(target any, name string) core.Rule {
	return d.BindStateless(target, name)
}

// BindStateless creates a goroutine-safe SliceRule bound to the target.
// The returned Rule can be shared across goroutines without synchronization.
func (d *SliceDef) BindStateless(target any, name string) *chains.SliceRule {
	d.mu.Lock()
	modifiers := d.modifiers
	elements := d.elementTemplates
	d.mu.Unlock()

	config := chains.CompileSliceConfig(modifiers)
	return chains.NewSliceRule(config, target, name, elements)
}

// MinLen enforces a minimum slice length.
func (d *SliceDef) MinLen(vMin int) *SliceDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.SliceChain) {
		// Using the chain's built-in MinLen method ensures consistent error reporting.
		c.MinLen(vMin)
	})
	return d
}

// MaxLen enforces a maximum slice length.
func (d *SliceDef) MaxLen(vMax int) *SliceDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.SliceChain) {
		// Using the chain's built-in MaxLen method ensures consistent error reporting.
		c.MaxLen(vMax)
	})
	return d
}

// NotEmpty enforces that the slice is not empty.
func (d *SliceDef) NotEmpty() *SliceDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.modifiers = append(d.modifiers, func(c *chains.SliceChain) {
		c.NotEmpty()
	})
	return d
}

// Elements registers recursive validation templates for slice items.
// These templates are stored in the definition and passed to every new chain during Bind.
func (d *SliceDef) Elements(templates ...core.Template) *SliceDef {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.elementTemplates = append(d.elementTemplates, templates...)
	return d
}

// GetElementTemplate returns the first configured element template.
// This is primarily used for introspection during testing (e.g., Matrix definition verification).
func (d *SliceDef) GetElementTemplate() core.Template {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.elementTemplates) > 0 {
		return d.elementTemplates[0]
	}
	return nil
}
