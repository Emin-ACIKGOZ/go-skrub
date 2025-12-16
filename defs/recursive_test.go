// SPDX-License-Identifier: MIT

package defs_test

import (
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

func TestNewMatrixDef(t *testing.T) {
	t.Parallel()

	defaultCtx := core.NewContext(core.Config{})

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

	t.Run("OneDimensionReturnsSingleSliceDef", func(t *testing.T) {
		t.Parallel()
		// Should return DefSlice().Elements(innerIntRule).
		template := defs.NewMatrixDef(1, innerIntRule)

		// Target: []int.
		data := []int{-1} // Should fail Min(0).
		rule := template.Bind(&data, "vector")

		err := rule.Validate(defaultCtx)
		expectedPath := "vector[0]"

		if fe, ok := err.(*core.FieldError); !ok || fe.Path != expectedPath {
			t.Fatalf("Expected failure at '%s', got: %v", expectedPath, err)
		}
	})

	t.Run("ThreeDimensionsRecursiveValidation", func(t *testing.T) {
		t.Parallel()

		// Should recursively nest three SliceDefs.
		matrixTemplate := defs.NewMatrixDef(3, innerIntRule)

		// Target: [][][]int with failure at index [0][1][0].
		matrix := [][][]int{
			{
				{100},
				{-5}, // Fails Min(0).
			},
		}

		// Bind to the pointer of the data slice.
		var matrixPtr any = &matrix
		rule := matrixTemplate.Bind(matrixPtr, "matrix")

		err := rule.Validate(defaultCtx)

		expectedPath := "matrix[0][1][0]"

		if fe, ok := err.(*core.FieldError); !ok || fe.Path != expectedPath {
			t.Fatalf("Expected deep path '%s', got: %s (Reason: %s)", expectedPath, fe.Path, fe.Reason)
		}
	})
}
