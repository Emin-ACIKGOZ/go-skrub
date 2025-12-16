// SPDX-License-Identifier: MIT

package defs

import (
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// NewMatrixDef creates a recursive, unbound template for validating N-dimensional slices (matrices).
// It automatically nests SliceDef templates to match the requested number of dimensions.
//
// The recursion terminates when dimensions is 0, returning the original template as the innermost rule.
//
// Example:
//
//	// Validates a 3D matrix ([][][]int) where every integer must be >= 0.
//	skrub.DefMatrix(3, skrub.DefInt().Min(0))
func NewMatrixDef(dimensions int, template core.Template) core.Template {
	// If dimensions are zero or negative, return the inner template as the identity.
	if dimensions <= 0 {
		return template
	}

	// Base Case: 1 Dimension is a SliceDef containing the element template.
	if dimensions == 1 {
		return NewSliceDef().Elements(template)
	}

	// Recursive Step: Create a SliceDef whose elements are defined by a Matrix of (N-1) dimensions.
	return NewSliceDef().Elements(NewMatrixDef(dimensions-1, template))
}
