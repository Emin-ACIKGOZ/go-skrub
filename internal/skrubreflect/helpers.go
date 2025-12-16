// SPDX-License-Identifier: MIT

// Package skrubreflect provides safe helpers for dealing with Go's reflection and memory addressing.
// These helpers are primarily focused on determining the correct interface value (pointer vs. value)
// to be used when binding a value to a validation rule chain.
package skrubreflect

import (
	"reflect"
)

// ResolveValue returns the interface{} value that is best suited for validation binding.
//
// It prioritizes returning the address (*T) of the underlying value over the value (T) itself
// when possible (e.g., if the value is addressable or already a pointer). This ensures
// validation chains can operate on a reference when expected.
// If the value is not addressable (e.g., a simple literal), the value (T) is returned as a copy.
func ResolveValue(v reflect.Value) any {
	// If the reflect.Value already represents a pointer or an interface, return the
	// contained reference directly.
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		return v.Interface()
	}

	// If the value is addressable (e.g., it is a field in a struct or an element in a slice
	// that was obtained from an addressable value), return its address (*T).
	if v.CanAddr() {
		return v.Addr().Interface()
	}

	// Fallback: The value is not addressable (e.g., a temporary or a constant).
	// Return the value itself (T) as a copy. Bound chains must handle this case.
	return v.Interface()
}
