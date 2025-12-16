// SPDX-License-Identifier: MIT

package adapters

import "fmt"

// UUIDStringer defines the minimal interface required for a UUID type (a String() method).
// This ensures compatibility across various UUID library implementations.
type UUIDStringer interface {
	String() string
}

// UUIDAdapter wraps any UUID-like type to make it compatible with validation chains,
// primarily chains that operate on strings (e.g., chains.StringChain).
//
// Implementing core.Valuer allows the validation engine to "unwrap" the UUID
// value into a string representation for validation.
type UUIDAdapter struct {
	// Val is the underlying UUID-like value. It must implement String().
	Val UUIDStringer
}

// Unwrap satisfies the core.Valuer interface.
//
// Unwrap returns the string representation of the underlying UUID value.
// If the underlying value is nil, Unwrap returns an empty string.
func (u *UUIDAdapter) Unwrap() any {
	if u.Val == nil {
		return ""
	}
	return u.Val.String()
}

// UUID creates and returns a new UUIDAdapter for a UUID value that implements UUIDStringer.
//
// Example:
//
//	skrub.String(adapters.UUID(myID), "id").UUID()
func UUID(u UUIDStringer) *UUIDAdapter {
	return &UUIDAdapter{Val: u}
}

// UUIDPtr creates and returns a new UUIDAdapter for a value or pointer to a value
// that implements UUIDStringer.
//
// It attempts to safely resolve the underlying UUID stringer from a generic interface.
// If the input is nil or cannot be resolved to a stringer, it returns an adapter
// with a nil internal value, which Unwrap translates to an empty string.
//
// Example:
//
//	skrub.String(adapters.UUIDPtr(&ptrToID), "id").UUID()
func UUIDPtr(u interface{}) *UUIDAdapter {
	if u == nil {
		return &UUIDAdapter{Val: nil}
	}

	// Try to assert to the specific UUID Stringer interface.
	if s, ok := u.(UUIDStringer); ok {
		return &UUIDAdapter{Val: s}
	}

	// Fallback check for standard fmt.Stringer interface.
	if s, ok := u.(fmt.Stringer); ok {
		return &UUIDAdapter{Val: s}
	}

	// If resolution fails, return nil adapter. The validator chain handles the empty string.
	return &UUIDAdapter{Val: nil}
}
