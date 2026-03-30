// SPDX-License-Identifier: MIT

package defs

import (
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// NewMatrixDef creates a recursive, unbound template for validating N-dimensional slices (matrices).
// It automatically nests SliceDef templates to match the requested number of dimensions.
//
// Every nested layer created by this factory is anonymous; the root name is applied
// when the resulting template is bound to a target.
func NewMatrixDef(dimensions int, template core.Template) core.Template {
	if dimensions <= 0 {
		return template
	}

	// Base Case: 1 Dimension is an anonymous SliceDef containing the element template.
	if dimensions == 1 {
		return NewSliceDef().Elements(template)
	}

	// Recursive Step: Create an anonymous SliceDef whose elements are defined by a (N-1) Matrix.
	return NewSliceDef().Elements(NewMatrixDef(dimensions-1, template))
}
