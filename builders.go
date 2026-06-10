// SPDX-License-Identifier: MIT

// Package skrub provides the public facade and core entry points for the validation library.
// It re-exports essential interfaces, types, and configurations from the core package
// to simplify imports for the end user.
package skrub

import (
	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// -----------------------------------------------------------------------------
// Bound Chain Constructors
// -----------------------------------------------------------------------------

// String creates a stateful validator chain bound to a string target.
// The target must be a *string or an object implementing the core.Valuer interface.
//
// Example:
//
//	skrub.String(&user.Name, "name").Min(5).Email().Validate(ctx)
func String(target any, name string) *chains.StringChain {
	return chains.NewStringChain(target, name)
}

// Slice creates a stateful validator chain bound to a slice target.
// The target must be a pointer to a slice (*[]T) or a slice ([]T).
// The chain uses reflection to handle element iteration and recursion.
//
// Example:
//
//	skrub.Slice(&user.Tags, "tags").MinLen(1).Elements(skrub.DefString().Max(20)).Validate(ctx)
func Slice(target any, name string) *chains.SliceChain {
	return chains.NewSliceChain(target, name)
}

// -----------------------------------------------------------------------------
// Unbound Template Constructors (Definitions)
// -----------------------------------------------------------------------------

// DefString creates an unbound template for string validation.
// This template can be reused and bound to multiple targets later.
func DefString() *defs.StringDef {
	return defs.NewStringDef()
}

// DefInt creates an unbound template for integer validation.
// This template can be reused and bound to multiple targets later.
func DefInt() *defs.IntDef {
	return defs.NewIntDef()
}

// DefSlice creates an unbound template for slice validation.
// This template can be reused and bound to multiple targets later.
func DefSlice() *defs.SliceDef {
	return defs.NewSliceDef()
}

// DefStruct creates a new StructDef for explicit field-level validation.
// Use Field() to register validators and Bind() to produce a StructChain.
//
// Example:
//
//	rule := skrub.DefStruct().
//	    Field("Name", skrub.DefString().NotEmpty()).
//	    Field("Age", skrub.DefInt().Min(18)).
//	    Bind(&user)
//	skrub.Validate(&user, rule)
func DefStruct() *defs.StructDef {
	return defs.NewStructDef()
}

// DefMatrix creates a recursive template that automatically nests SliceDef templates
// to validate N-dimensional slices (matrices).
//
// Example:
//
//	// Validates a 3D slice ([][][]int) where every integer must be >= 0.
//	matrixTemplate := skrub.DefMatrix(3, skrub.DefInt().Min(0))
//	matrixTemplate.Bind(&data, "matrix").Validate(ctx)
func DefMatrix(dimensions int, template core.Template) core.Template {
	return defs.NewMatrixDef(dimensions, template)
}
