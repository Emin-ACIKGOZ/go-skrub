// SPDX-License-Identifier: MIT

package skrubreflect_test

import (
	stdreflect "reflect" // Alias standard reflect
	"testing"

	sreflect "github.com/Emin-ACIKGOZ/go-skrub/internal/skrubreflect" // Alias internal reflect
)

// simpleStruct is used to test addressable vs. non-addressable structs.
type simpleStruct struct {
	A int
}

func TestResolveValue(t *testing.T) {
	t.Parallel()

	// Helper function now uses the standard library alias.
	isPointer := func(v any) bool {
		return stdreflect.TypeOf(v).Kind() == stdreflect.Ptr
	}

	t.Run("Case1_AlreadyPointer", func(t *testing.T) {
		t.Parallel()
		i := 100
		v := stdreflect.ValueOf(&i) // v is a stdreflect.Value of type *int

		result := sreflect.ResolveValue(v)

		if !isPointer(result) {
			t.Errorf("Expected result to be a pointer (*int), got %T", result)
		}
	})

	t.Run("Case2_Interface", func(t *testing.T) {
		t.Parallel()
		var i any = &simpleStruct{A: 1}
		v := stdreflect.ValueOf(i) // v is a stdreflect.Value of type interface{} (holds *simpleStruct)

		result := sreflect.ResolveValue(v)

		if !isPointer(result) {
			t.Errorf("Expected result to be a pointer, got %T", result)
		}
	})

	t.Run("Case3_AddressableValue", func(t *testing.T) {
		t.Parallel()

		s := []int{10}
		v := stdreflect.ValueOf(s).Index(0) // v is stdreflect.Value of int, CanAddr() is true

		if !v.CanAddr() {
			t.Fatalf("Test setup error: Value was expected to be addressable but was not.")
		}

		result := sreflect.ResolveValue(v)

		if !isPointer(result) {
			t.Errorf("Expected addressable value to be returned as a pointer (*int), got %T", result)
		}
	})

	t.Run("Case4_NonAddressableValue", func(t *testing.T) {
		t.Parallel()

		v := stdreflect.ValueOf(100) // v is stdreflect.Value of int, CanAddr() is false

		if v.CanAddr() {
			t.Fatalf("Test setup error: Value was expected to be non-addressable but was.")
		}

		result := sreflect.ResolveValue(v)

		// Result should be the value itself (int), not a pointer (*int).
		if isPointer(result) {
			t.Errorf("Expected result to be a value (int), got pointer %T", result)
		}
		if _, ok := result.(int); !ok {
			t.Errorf("Expected result type int, got %T", result)
		}
	})
}
