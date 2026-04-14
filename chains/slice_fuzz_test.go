// SPDX-License-Identifier: MIT

package chains_test

import (
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// FuzzSliceChainNotEmpty tests NotEmpty validator against arbitrary slice inputs.
// Verifies that:
// - Validator never panics with any slice size
// - Empty slices are rejected
// - Non-empty slices are accepted
// - Various element types are handled correctly
func FuzzSliceChainNotEmpty(f *testing.F) {
	// Seed with known test cases
	f.Add(0)    // Empty slice
	f.Add(1)    // Single element
	f.Add(10)   // Multiple elements
	f.Add(100)  // Large slice
	f.Add(1000) // Very large slice

	f.Fuzz(func(t *testing.T, size int) {
		// Clamp size to reasonable bounds for fuzzing
		if size < 0 || size > 10000 {
			return
		}

		// Test with string slices
		stringSlice := make([]string, size)
		for i := 0; i < size; i++ {
			stringSlice[i] = "item"
		}

		chain := chains.NewSliceChain(&stringSlice, "items")
		chain.NotEmpty()

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// Verify behavior matches expectation
		if size == 0 && err == nil {
			t.Error("NotEmpty should reject empty slice, got nil error")
		}
		if size > 0 && err != nil {
			t.Errorf("NotEmpty should accept non-empty slice (size %d), got error: %v", size, err)
		}
	})
}

// FuzzSliceChainMinLen tests MinLen validator against arbitrary slice inputs.
// Verifies that:
// - Validator never panics with any slice size
// - Slices smaller than minimum are rejected
// - Slices >= minimum are accepted
func FuzzSliceChainMinLen(f *testing.F) {
	// Seed with known test cases
	f.Add(0, 0)
	f.Add(5, 3)
	f.Add(3, 5)
	f.Add(0, 10)
	f.Add(100, 50)

	f.Fuzz(func(t *testing.T, size int, minLen int) {
		// Clamp to reasonable bounds
		if size < 0 || size > 1000 || minLen < 0 || minLen > 1000 {
			return
		}

		// Test with string slices
		stringSlice := make([]string, size)
		for i := 0; i < size; i++ {
			stringSlice[i] = "item"
		}

		chain := chains.NewSliceChain(&stringSlice, "items")
		chain.MinLen(minLen)

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// Verify behavior matches expectation
		if size >= minLen && err != nil {
			t.Errorf("MinLen(%d) should accept slice of size %d, got error: %v", minLen, size, err)
		}
		if size < minLen && err == nil {
			t.Errorf("MinLen(%d) should reject slice of size %d, got nil error", minLen, size)
		}
	})
}

// FuzzSliceChainMaxLen tests MaxLen validator against arbitrary slice inputs.
// Verifies that:
// - Validator never panics with any slice size
// - Slices larger than maximum are rejected
// - Slices <= maximum are accepted
func FuzzSliceChainMaxLen(f *testing.F) {
	// Seed with known test cases
	f.Add(0, 0)
	f.Add(5, 10)
	f.Add(10, 5)
	f.Add(0, 100)
	f.Add(50, 100)

	f.Fuzz(func(t *testing.T, size int, maxLen int) {
		// Clamp to reasonable bounds
		if size < 0 || size > 1000 || maxLen < 0 || maxLen > 1000 {
			return
		}

		// Test with string slices
		stringSlice := make([]string, size)
		for i := 0; i < size; i++ {
			stringSlice[i] = "item"
		}

		chain := chains.NewSliceChain(&stringSlice, "items")
		chain.MaxLen(maxLen)

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// Verify behavior matches expectation
		if size <= maxLen && err != nil {
			t.Errorf("MaxLen(%d) should accept slice of size %d, got error: %v", maxLen, size, err)
		}
		if size > maxLen && err == nil {
			t.Errorf("MaxLen(%d) should reject slice of size %d, got nil error", maxLen, size)
		}
	})
}
