// SPDX-License-Identifier: MIT

package skrub

import (
	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// These variable declarations serve as static assertions.
// They ensure that the concrete chain and definition types correctly implement the
// required core interfaces (core.Rule, core.Resetter, core.Template) at compile time.
// The build will fail if any type does not satisfy its contract.

// --- Bound Chains Verification ---

// Verify StringChain implements core.Rule (for validation logic)
// and core.Resetter (for object pooling).
var (
	_ core.Rule     = (*chains.StringChain)(nil)
	_ core.Resetter = (*chains.StringChain)(nil)
)

// Verify IntChain implements core.Rule and core.Resetter.
var (
	_ core.Rule     = (*chains.IntChain)(nil)
	_ core.Resetter = (*chains.IntChain)(nil)
)

// Verify SliceChain implements core.Rule and core.Resetter.
var (
	_ core.Rule     = (*chains.SliceChain)(nil)
	_ core.Resetter = (*chains.SliceChain)(nil)
)

// --- Unbound Templates Verification ---

// Verify StringDef implements core.Template (allowing it to be bound into a Rule).
var _ core.Template = (*defs.StringDef)(nil)

// Verify IntDef implements core.Template.
var _ core.Template = (*defs.IntDef)(nil)

// Verify SliceDef implements core.Template.
var _ core.Template = (*defs.SliceDef)(nil)
