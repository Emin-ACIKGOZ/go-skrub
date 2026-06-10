// SPDX-License-Identifier: MIT

// Package core defines the fundamental interfaces, types, errors, and context management
// for the go-skrub validation engine. This package serves as the single source of truth
// to break architectural import cycles.
package core

import (
	"errors"
	"fmt"
	"strings"
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

// StructLevel provides the interface for struct-level validators.
// Implementations receive this in ValidateWith callbacks to access
// field values and report cross-field errors.
type StructLevel interface {
	// FieldValue returns the value of a named field, or an error if
	// the field does not exist or is not addressable.
	FieldValue(name string) (any, error)
	// ReportError records a validation error for the given field path and reason.
	ReportError(path, reason string)
	// Context returns the current validation context.
	Context() *Context
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

// ValidationErrors collects multiple field validation errors.
// It implements the error interface and supports Go 1.20+ multi-error unwrapping
// for compatibility with errors.Is and errors.As.
type ValidationErrors []*FieldError

// Error returns a multiline string of all accumulated errors.
// Format matches go-validator's convention for compatibility.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, fe := range ve {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fe.Error())
	}
	return sb.String()
}

// Unwrap implements the Go 1.20+ interface for multi-error unwrapping.
// This enables errors.Is and errors.As to search within the accumulated errors.
func (ve ValidationErrors) Unwrap() []error {
	result := make([]error, len(ve))
	for i, fe := range ve {
		result[i] = fe
	}
	return result
}

// Is enables targeted error matching across ValidationErrors.
// Returns true if any contained FieldError matches the target.
func (ve ValidationErrors) Is(target error) bool {
	for _, fe := range ve {
		if errors.Is(fe, target) {
			return true
		}
	}
	return false
}
