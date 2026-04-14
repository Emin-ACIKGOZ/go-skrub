// SPDX-License-Identifier: MIT

package chains_test

import (
	"errors"
	"testing"

	skrub "github.com/Emin-ACIKGOZ/go-skrub"
	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// ptr is a helper to get a pointer to a string.
func ptr(s string) *string { return &s }

// ptrInt is a helper to get a pointer to an int.
func ptrInt(i int) *int { return &i }

func TestSliceChainLength(t *testing.T) {
	t.Parallel()

	t.Run("MinLenFailure", func(t *testing.T) {
		t.Parallel()
		tags := []string{"a", "b"}
		// FIX: Removed the 3rd 'nil' argument
		chain := chains.NewSliceChain(&tags, "tags")

		err := chain.MinLen(3).Validate(nil)

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != core.ReasonMinLength {
			t.Errorf("Expected MinLen failure, got: %v", err)
		}
	})

	t.Run("MaxLenSuccess", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3, 4}
		// FIX: Removed the 3rd 'nil' argument
		chain := chains.NewSliceChain(&items, "items")

		err := chain.MaxLen(5).Validate(nil)

		if err != nil {
			t.Errorf("Expected MaxLen success, got: %v", err)
		}
	})
}

func TestSliceChainRecursion(t *testing.T) {
	t.Parallel()

	// Use DefString() template for string elements.
	strTemplate := skrub.DefString().Min(5)

	t.Run("RecursiveElementFailure", func(t *testing.T) {
		t.Parallel()
		data := []*string{
			ptr("valid"),
			ptr("fail"), // too short
		}
		// FIX: Removed the 3rd 'nil' argument
		chain := chains.NewSliceChain(&data, "data")

		err := chain.Elements(strTemplate).Validate(nil)

		// Check for failure at data[1].
		if fe, ok := err.(*core.FieldError); !ok || fe.Path != "data[1]" {
			t.Errorf("Expected failure path 'data[1]', got: %s", fe.Path)
		}
	})

	t.Run("NilSlicePointerPass", func(t *testing.T) {
		t.Parallel()
		var data []*string
		// FIX: Removed the 3rd 'nil' argument
		chain := chains.NewSliceChain(data, "data")

		err := chain.Elements(strTemplate).Validate(nil)

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

		// FIX: Removed the 3rd 'nil' argument
		chain := chains.NewSliceChain(&data, "data")

		// Apply the template for *int to the elements of the []*int slice.
		err := chain.Elements(intTemplate).Validate(nil)

		if fe, ok := err.(*core.FieldError); !ok || fe.Path != "data[1]" {
			t.Fatalf("Expected deep path 'data[1]', got: %s", fe.Path)
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
		matrix := [][][]int{{{1}}}
		// 3. Define a matrix validator (3 dimensions).
		matrixTemplate := skrub.DefMatrix(3, skrub.DefInt())

		// 4. Bind and Validate.
		rule := matrixTemplate.Bind(&matrix, "matrix")
		err := rule.Validate(strictCtx)

		// 5. Assert that a RecursionError occurred.
		if _, ok := err.(*core.RecursionError); !ok {
			t.Fatalf("Expected RecursionError, got %T: %v", err, err)
		}
	})
}

func TestSliceChainNilContext(t *testing.T) {
	t.Parallel()

	t.Run("ValidateWithNilContext", func(t *testing.T) {
		t.Parallel()
		tags := []string{"a", "b"}
		// FIX: Removed the 3rd 'nil' argument
		chain := chains.NewSliceChain(&tags, "tags")

		// 1. Pass nil to Validate.
		// This forces the new logic in SliceChain.Validate to create the default context.
		err := chain.MinLen(3).Validate(nil)

		// 2. Verify it still behaves correctly (fails validation).
		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != core.ReasonMinLength {
			t.Errorf("Expected MinLen failure with nil context, got: %v", err)
		}
	})
}

func TestSliceChain_HeterogeneousRebinding(t *testing.T) {
	t.Parallel()

	// Data with mixed types.
	data := []any{100, "not-an-int"}

	// Using IntDef on an any slice.
	intTemplate := skrub.DefInt().Min(0)
	chain := chains.NewSliceChain(&data, "mixed")

	err := chain.Elements(intTemplate).Validate(nil)

	// Verify that the IntChain flyweight caught the string type mismatch.
	if !errors.Is(err, core.ErrMisuse) {
		t.Errorf("Expected ErrMisuse for heterogeneous rebinding, got: %v", err)
	}

	// Verify path remains correct for the mismatched element.
	if fe, ok := err.(*core.FieldError); ok && fe.Path != "mixed[1]" {
		t.Errorf("Expected path 'mixed[1]', got %q", fe.Path)
	}
}

func TestSliceChainNotEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		slice     interface{}
		expectErr bool
		reason    string
	}{
		// Valid non-empty slices
		{"SingleElement", []string{"a"}, false, ""},
		{"MultiElement", []int{1, 2, 3}, false, ""},
		{"LongSlice", []string{"a", "b", "c", "d", "e"}, false, ""},
		{"PointerSlice", []*int{ptrInt(1), ptrInt(2)}, false, ""},

		// Invalid empty slices
		{"EmptySlice", []string{}, true, core.ReasonRequired},
		{"EmptyIntSlice", []int{}, true, core.ReasonRequired},
		{"EmptyPointerSlice", []*int{}, true, core.ReasonRequired},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chain := chains.NewSliceChain(&tt.slice, "items")
			err := chain.NotEmpty().Validate(nil)

			if (err != nil) != tt.expectErr {
				t.Errorf("NotEmpty: expected error=%v, got error=%v", tt.expectErr, err)
				return
			}

			if tt.expectErr && err != nil {
				if fe, ok := err.(*core.FieldError); !ok || fe.Reason != tt.reason {
					t.Errorf("NotEmpty: expected reason %q, got %q", tt.reason, fe.Reason)
				}
			}
		})
	}
}

func TestSliceChainNotEmptyWithOtherValidators(t *testing.T) {
	t.Parallel()

	t.Run("NotEmpty_Then_MinLen", func(t *testing.T) {
		t.Parallel()
		items := []string{"a"}
		chain := chains.NewSliceChain(&items, "items").NotEmpty().MinLen(2)
		err := chain.Validate(nil)
		if err == nil {
			t.Error("Expected MinLen failure, got nil")
		}
	})

	t.Run("NotEmpty_Success_With_MaxLen", func(t *testing.T) {
		t.Parallel()
		items := []string{"a", "b"}
		chain := chains.NewSliceChain(&items, "items").NotEmpty().MaxLen(3)
		err := chain.Validate(nil)
		if err != nil {
			t.Errorf("Expected success, got: %v", err)
		}
	})

	t.Run("NotEmpty_With_Elements", func(t *testing.T) {
		t.Parallel()
		tags := []string{"ok"}
		template := skrub.DefString().Min(3)
		chain := chains.NewSliceChain(&tags, "tags").NotEmpty().Elements(template)
		err := chain.Validate(nil)
		if err == nil {
			t.Error("Expected element validation failure (string too short), got nil")
		}
	})
}
