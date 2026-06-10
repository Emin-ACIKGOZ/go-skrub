// SPDX-License-Identifier: MIT

package core

// ChainConfig holds the immutable, goroutine-safe portion of a validation chain's
// logic. A single ChainConfig is constructed once per BindStateless call and
// shared across all goroutines using that Rule. All fields are read-only after
// construction.
type ChainConfig struct {
	// Validators is the compiled list of validation functions.
	// Each function returns a *FieldError on failure, or nil on success.
	Validators []func(ctx *Context, val any) *FieldError

	// ElementConfigs holds pre-compiled configs for slice element templates.
	// Populated only for SliceRule; nil for scalar rules.
	ElementConfigs []*ChainConfig
}
