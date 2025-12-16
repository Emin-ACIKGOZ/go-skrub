// SPDX-License-Identifier: MIT

package skrub

import "github.com/Emin-ACIKGOZ/go-skrub/pkg/core"

var (
	// ErrMisuse indicates the library was used incorrectly (e.g., passing nil pointers)
	// or in an unsupported state.
	ErrMisuse = core.ErrMisuse

	// ErrConcurrencyViolation is returned when a validation chain is accessed
	// concurrently by multiple goroutines.
	ErrConcurrencyViolation = core.ErrConcurrencyViolation

	// ErrPoolExhausted is returned by SafePool when it is empty and configured
	// as NonBlocking.
	ErrPoolExhausted = core.ErrPoolExhausted
)

// RecursionError is the error type returned when a cyclical validation definition
// is detected, preventing infinite recursion.
type RecursionError = core.RecursionError

// NewFieldError creates a new validation error specific to a single field.
// It is useful for custom validators to return structured validation errors.
func NewFieldError(path string, value any, reason string) *core.FieldError {
	return &core.FieldError{
		Path:   path,
		Value:  value,
		Reason: reason,
	}
}
