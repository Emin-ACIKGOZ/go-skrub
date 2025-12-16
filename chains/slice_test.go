// SPDX-License-Identifier: MIT

package chains_test

import (
	"testing"

	skrub "github.com/Emin-ACIKGOZ/go-skrub"
	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// MockTarget is a simple struct to test reflection pathing.
type MockTarget struct {
	Items []int
}

// ptr is a helper to get a pointer to a string.
func ptr(s string) *string { return &s }

// ptrInt is a helper to get a pointer to an int.
func ptrInt(i int) *int { return &i }

func TestSliceChainLength(t *testing.T) {
	t.Parallel()

	defaultCtx := core.NewContext(core.Config{})

	t.Run("MinLenFailure", func(t *testing.T) {
		t.Parallel()
		tags := []string{"a", "b"}
		chain := chains.NewSliceChain(&tags, "tags")

		err := chain.MinLen(3).Validate(defaultCtx)

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != "slice length is less than minimum" {
			t.Errorf("Expected MinLen failure, got: %v", err)
		}
	})

	t.Run("MaxLenSuccess", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3, 4}
		chain := chains.NewSliceChain(&items, "items")

		err := chain.MaxLen(5).Validate(defaultCtx)

		if err != nil {
			t.Errorf("Expected MaxLen success, got: %v", err)
		}
	})
}

func TestSliceChainRecursion(t *testing.T) {
	t.Parallel()

	defaultCtx := core.NewContext(core.Config{})

	// Use DefString() template for string elements.
	strTemplate := skrub.DefString().Min(5)

	t.Run("RecursiveElementFailure", func(t *testing.T) {
		t.Parallel()
		data := []*string{
			ptr("valid"),
			ptr("fail"), // too short
		}
		chain := chains.NewSliceChain(&data, "data")

		err := chain.Elements(strTemplate).Validate(defaultCtx)

		// Check for failure at data[1].
		if fe, ok := err.(*core.FieldError); !ok || fe.Path != "data[1]" {
			t.Errorf("Expected failure path 'data[1]', got: %s (Reason: %s)", fe.Path, fe.Reason)
		}
	})

	t.Run("NilSlicePointerPass", func(t *testing.T) {
		t.Parallel()
		var data []*string
		chain := chains.NewSliceChain(data, "data")

		err := chain.Elements(strTemplate).Validate(defaultCtx)

		if err != nil {
			t.Errorf("Expected nil error for nil slice pointer, got: %v", err)
		}
	})

	t.Run("DeepRecursionPathing", func(t *testing.T) {
		t.Parallel()
		// Structure: []*int (slice of pointers to ints).
		data := []*int{
			ptrInt(100),
			ptrInt(-5), // Failure point.
		}

		// Template for the element type *int, requiring *int >= 0.
		intTemplate := skrub.DefInt().Min(0)

		chain := chains.NewSliceChain(&data, "data")

		// Apply the template for *int to the elements of the []*int slice.
		err := chain.Elements(intTemplate).Validate(defaultCtx)

		// Check for failure at data[1].
		expectedPath := "data[1]"
		if fe, ok := err.(*core.FieldError); !ok || fe.Path != expectedPath {
			t.Fatalf("Expected deep path '%s', got: %s (Reason: %s)", expectedPath, fe.Path, fe.Reason)
		}
	})

	t.Run("RecursionLimitEnforcement", func(t *testing.T) {
		t.Parallel()

		// 1. Configure a strict depth limit (MaxDepth = 2).
		// Depth 0: Root
		// Depth 1: outer[0]
		// Depth 2: outer[0][0]
		// Depth 3: outer[0][0][0] -> Should Fail
		strictCtx := core.NewContext(core.Config{MaxDepth: 2})

		// 2. Create a 3-dimensional nested slice (Matrix).
		// matrix[0][0][0]
		matrix := [][][]int{
			{
				{1},
			},
		}

		// 3. Define a matrix validator (3 dimensions).
		matrixTemplate := skrub.DefMatrix(3, skrub.DefInt())

		// 4. Bind and Validate.
		rule := matrixTemplate.Bind(&matrix, "matrix")
		err := rule.Validate(strictCtx)

		// 5. Assert that a RecursionError occurred.
		if err == nil {
			t.Fatal("Expected RecursionError, got success (infinite recursion exploit possible)")
		}

		re, ok := err.(*core.RecursionError)
		if !ok {
			t.Fatalf("Expected error type *core.RecursionError, got %T: %v", err, err)
		}

		// Check that it failed at the expected depth.
		// Expected path: matrix[0][0][0]
		if re.Depth <= 2 {
			t.Errorf("RecursionError triggered too early at depth %d (MaxDepth 2)", re.Depth)
		}
	})
}

func TestSliceChainNilContext(t *testing.T) {
	t.Parallel()

	t.Run("ValidateWithNilContext", func(t *testing.T) {
		t.Parallel()
		tags := []string{"a", "b"}
		chain := chains.NewSliceChain(&tags, "tags")

		// 1. Pass nil to Validate.
		// This forces the new logic in SliceChain.Validate to create the default context.
		err := chain.MinLen(3).Validate(nil)

		// 2. Verify it still behaves correctly (fails validation).
		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != "slice length is less than minimum" {
			t.Errorf("Expected MinLen failure with nil context, got: %v", err)
		}
	})
}
