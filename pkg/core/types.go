// SPDX-License-Identifier: MIT

// Package core defines the fundamental interfaces, types, errors, and context management
// for the go-skrub validation engine. This package serves as the single source of truth
// to break architectural import cycles.
package core

import (
	"fmt"
)

// =============================================================================
// Interfaces
// =============================================================================

// Valuer allows custom types (adapters) to expose primitives for validation.
type Valuer interface {
	Unwrap() any
}

// Rule is the fundamental interface for a bound, stateful validation chain.
type Rule interface {
	Validate(ctx *Context) error
}

// Rebindable allows a Rule to be reset with a new target value.
// This enables the "Flyweight Pattern" where a single chain instance
// is allocated once and reused across a loop.
type Rebindable interface {
	SetTarget(target any)
}

// Resetter allows a Rule to clear its internal state, enabling safe pooling.
type Resetter interface {
	Reset()
}

// Template represents a definition of validation logic (unbound).
type Template interface {
	Bind(target any, name string) Rule
}

// =============================================================================
// Errors
// =============================================================================

// RecursionError is returned when validation exceeds the configured MaxDepth.
type RecursionError struct {
	Path     string
	Depth    int
	MaxDepth int
}

// Error returns a formatted message indicating the recursion limit was exceeded.
func (e *RecursionError) Error() string {
	return fmt.Sprintf("skrub: recursion limit exceeded at %s (depth %d > %d)", e.Path, e.Depth, e.MaxDepth)
}

// FieldError represents a validation failure on a specific field.
type FieldError struct {
	Path   string
	Value  any
	Reason string
	Cause  error
}

// Error returns the formatted path and reason for the validation failure.
// If Path is empty, it returns only the reason.
func (e *FieldError) Error() string {
	if e.Path == "" {
		return e.Reason
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Reason)
}

// Unwrap supports standard Go error wrapping, returning the underlying Cause.
func (e *FieldError) Unwrap() error { return e.Cause }

// NewFieldError creates a simplified FieldError without a wrapped cause.
func NewFieldError(path string, value any, reason string) *FieldError {
	return &FieldError{
		Path:   path,
		Value:  value,
		Reason: reason,
	}
}
