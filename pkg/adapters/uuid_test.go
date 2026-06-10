// SPDX-License-Identifier: MIT

package adapters_test

import (
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/adapters"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// MockUUID implements the UUIDStringer interface for testing.
type MockUUID string

func (m MockUUID) String() string {
	return string(m)
}

// MockStringer implements fmt.Stringer but not adapters.UUIDStringer directly.
type MockStringer string

func (m MockStringer) String() string {
	return string(m)
}

// Define test constants.
const validUUID = "a1b2c3d4-e5f6-4000-8000-1234567890ab"
const invalidUUID = "not-a-uuid"

func TestUUIDAdapterUnwrap(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		uuidVal := MockUUID(validUUID)
		adapter := adapters.UUID(uuidVal)

		unwrapped := adapter.Unwrap()

		if s, ok := unwrapped.(string); !ok || s != validUUID {
			t.Fatalf("Expected Unwrap to return string '%s', got '%v'", validUUID, unwrapped)
		}
	})

	t.Run("NilValue", func(t *testing.T) {
		t.Parallel()
		adapter := adapters.UUID(MockUUID(""))
		adapter.Val = nil // Force the internal Val to nil.

		unwrapped := adapter.Unwrap()

		if s, ok := unwrapped.(string); !ok || s != "" {
			t.Fatalf("Expected Unwrap of nil Val to return empty string, got '%v'", unwrapped)
		}
	})
}

func TestUUIDPtrConstructor(t *testing.T) {
	t.Parallel()

	t.Run("NilInput", func(t *testing.T) {
		t.Parallel()
		adapter := adapters.UUIDPtr(nil)

		if adapter.Val != nil {
			t.Errorf("Expected adapter.Val to be nil for nil input, got %v", adapter.Val)
		}
		if adapter.Unwrap() != "" {
			t.Errorf("Expected Unwrap to be empty string for nil input.")
		}
	})

	t.Run("DirectStringerInput", func(t *testing.T) {
		t.Parallel()
		// Pass the concrete type implementing UUIDStringer.
		uuidVal := MockUUID(validUUID)
		adapter := adapters.UUIDPtr(uuidVal)

		if adapter.Unwrap() != validUUID {
			t.Errorf("Expected UUIDPtr to resolve UUIDStringer value.")
		}
	})

	t.Run("FmtStringerFallback", func(t *testing.T) {
		t.Parallel()
		// Pass a type that only implements fmt.Stringer.
		stringerVal := MockStringer("fallback-test")
		adapter := adapters.UUIDPtr(stringerVal)

		if adapter.Unwrap() != "fallback-test" {
			t.Errorf("Expected UUIDPtr to resolve fmt.Stringer fallback.")
		}
	})
}

func TestUUIDAdapterIntegration(t *testing.T) {
	t.Parallel()

	t.Run("ValidationSuccess", func(t *testing.T) {
		t.Parallel()
		uuidVal := MockUUID(validUUID)
		adapter := adapters.UUID(uuidVal)

		rule := skrub.String(adapter, "id").UUID()

		ctx := core.NewContext(core.Config{})
		if err := rule.Validate(ctx); err != nil {
			t.Fatalf("Expected validation success for valid UUID, got error: %v", err)
		}
	})

	t.Run("ValidationFailure", func(t *testing.T) {
		t.Parallel()
		uuidVal := MockUUID(invalidUUID)
		adapter := adapters.UUID(uuidVal)

		rule := skrub.String(adapter, "id").UUID()

		ctx := core.NewContext(core.Config{})
		err := rule.Validate(ctx)

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != "invalid UUID format" {
			t.Fatalf("Expected UUID format failure, got: %v", err)
		}
	})
}
