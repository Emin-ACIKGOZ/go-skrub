// SPDX-License-Identifier: MIT

package skrub_test

import (
	"errors"
	"reflect"
	"testing"

	skrub "github.com/Emin-ACIKGOZ/go-skrub"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// These tests ensure the public facade is correctly linked to the internal core structures.

func TestPublicErrorVariables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		publicError  error
		expectedCore error
	}{
		{
			name:         "ErrMisuse",
			publicError:  skrub.ErrMisuse,
			expectedCore: core.ErrMisuse,
		},
		{
			name:         "ErrConcurrencyViolation",
			publicError:  skrub.ErrConcurrencyViolation,
			expectedCore: core.ErrConcurrencyViolation,
		},
		{
			name:         "ErrPoolExhausted",
			publicError:  skrub.ErrPoolExhausted,
			expectedCore: core.ErrPoolExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Check if the public variable is the *exact same* error instance as the core variable.
			if !errors.Is(tt.publicError, tt.expectedCore) {
				t.Errorf("Public error variable %s is not identical to core.%s", tt.name, tt.name)
			}
			if tt.publicError.Error() != tt.expectedCore.Error() {
				t.Errorf("Error messages do not match. Got '%s', want '%s'", tt.publicError.Error(), tt.expectedCore.Error())
			}
		})
	}
}

func TestPublicErrorTypeAliases(t *testing.T) {
	t.Parallel()

	// 1. Test RecursionError type alias.
	t.Run("RecursionErrorAlias", func(t *testing.T) {
		t.Parallel()

		// Create an instance of the public type alias.
		publicErr := skrub.RecursionError{
			Path:     "deep.field",
			Depth:    101,
			MaxDepth: 100,
		}

		// Use reflection to verify the underlying type is core.RecursionError.
		if reflect.TypeOf(publicErr) != reflect.TypeOf(core.RecursionError{}) {
			t.Errorf("skrub.RecursionError is not an alias of core.RecursionError. Public type: %v, Core type: %v",
				reflect.TypeOf(publicErr), reflect.TypeOf(core.RecursionError{}))
		}

		// Also check if the aliased type behaves correctly (implements Error() from core).
		expectedMsg := "skrub: recursion limit exceeded at deep.field (depth 101 > 100)"
		if publicErr.Error() != expectedMsg {
			t.Errorf("Aliased type Error() method failed. Got '%s', want '%s'", publicErr.Error(), expectedMsg)
		}
	})
}

func TestNewFieldErrorConstructor(t *testing.T) {
	t.Parallel()

	const path = "user.email"
	const reason = "invalid format"
	const value = "bad_email"

	// Test the public constructor function.
	err := skrub.NewFieldError(path, value, reason)

	// 1. Check if the internal fields were set correctly.
	if err.Path != path {
		t.Errorf("Path mismatch. Got %s, want %s", err.Path, path)
	}
	if err.Reason != reason {
		t.Errorf("Reason mismatch. Got %s, want %s", err.Reason, reason)
	}
	if err.Value != value {
		t.Errorf("Value mismatch. Got %s, want %s", err.Value, value)
	}

	// 2. Check the Error() output.
	expectedMsg := "user.email: invalid format"
	if err.Error() != expectedMsg {
		t.Errorf("Error() output mismatch. Got '%s', want '%s'", err.Error(), expectedMsg)
	}
}
