// SPDX-License-Identifier: MIT

package defs_test

import (
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

func TestNewMatrixDef(t *testing.T) {
	t.Parallel()

	// Define the innermost rule.
	innerIntRule := defs.NewIntDef().Min(0)

	t.Run("ZeroDimensionsReturnsBaseTemplate", func(t *testing.T) {
		t.Parallel()
		// If dimensions <= 0, it should return the inner rule directly.
		template := defs.NewMatrixDef(0, innerIntRule)

		if template != innerIntRule {
			t.Errorf("Expected NewMatrixDef(0) to return inner rule, got a different template.")
		}
	})

	t.Run("OneDimensionPathing", func(t *testing.T) {
		t.Parallel()
		// Fix: Initialize a local context for each parallel subtest.
		// Sharing a Context across goroutines causes data races on the internal stack.
		ctx := core.NewContext(core.Config{})

		template := defs.NewMatrixDef(1, innerIntRule)

		// Target: []int.
		data := []int{-1} // Should fail Min(0).
		rule := template.Bind(&data, "vector")

		err := rule.Validate(ctx)
		expectedPath := "vector[0]"

		if fe, ok := err.(*core.FieldError); !ok || fe.Path != expectedPath {
			t.Fatalf("Expected path %q, got: %v", expectedPath, err)
		}
	})

	t.Run("ThreeDimensionsRecursiveValidation", func(t *testing.T) {
		t.Parallel()
		// Fix: Initialize a local context for each parallel subtest.
		ctx := core.NewContext(core.Config{})

		// Should recursively nest three SliceDefs.
		matrixTemplate := defs.NewMatrixDef(3, innerIntRule)

		// Target: [][][]int with failure at index [0][1][0].
		matrix := [][][]int{
			{
				{100},
				{-5}, // Fails at [0][1][0]
			},
		}

		rule := matrixTemplate.Bind(&matrix, "matrix")
		err := rule.Validate(ctx)

		expectedPath := "matrix[0][1][0]"

		if fe, ok := err.(*core.FieldError); !ok || fe.Path != expectedPath {
			t.Fatalf("Expected deep path %q, got: %v", expectedPath, err)
		}
	})
}
