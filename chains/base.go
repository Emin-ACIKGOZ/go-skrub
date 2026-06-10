// SPDX-License-Identifier: MIT

// Package chains implements bound validation rules that hold state and target specific memory addresses.
//
// These chains are designed to be bound to a specific Go value (via a target address)
// at creation time and apply a set of validation rules against that value.
package chains

import (
	"reflect"
	"sync/atomic"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// BaseChain provides atomic state management and concurrency guards for validation chains.
// It is intended to be embedded in specific chain implementations to enforce safe,
// non-concurrent execution of the Validate method.
type BaseChain struct {
	// Name is the field or variable name this chain is bound to, used for
	// constructing the final validation error path.
	Name string

	// state tracks the active status of the chain (0 = Idle, 1 = Busy).
	state int32
}

// Acquire attempts to lock the chain for the current validation run using an atomic swap.
//
// If the chain is already in use (Busy), Acquire returns core.ErrConcurrencyViolation,
// guaranteeing zero-panic safety against concurrent misuse.
func (b *BaseChain) Acquire() error {
	if !atomic.CompareAndSwapInt32(&b.state, 0, 1) {
		return core.ErrConcurrencyViolation
	}
	return nil
}

// Release unlocks the chain by atomically setting the state to Idle.
// This method must be called via defer immediately after a successful Acquire call
// in the embedding chain's Validate method.
func (b *BaseChain) Release() {
	atomic.StoreInt32(&b.state, 0)
}

// Reset clears the internal state, making the chain ready for reuse in a pool.
// It sets Name to an empty string and atomically resets the state to Idle (0).
func (b *BaseChain) Reset() {
	b.Name = ""
	atomic.StoreInt32(&b.state, 0)
}

// emitError constructs a FieldError and either records it in the context
// (accumulate mode) or returns it directly (short-circuit mode).
// Returns nil when the error was accumulated, signaling the caller to continue.
// This replaces direct calls to Fail in Validate hot paths.
func (b *BaseChain) emitError(ctx *core.Context, value any, reason string) error {
	fe := b.Fail(ctx, value, reason)
	if ctx != nil && ctx.IsAccumulating() {
		if fe, ok := fe.(*core.FieldError); ok {
			ctx.RecordError(fe)
		}
		return nil
	}
	return fe
}

// Fail constructs a core.FieldError using the provided validation value and reason,
// correctly resolving the full error Path based on the current core.Context and the chain's Name.
//
// The resolution logic ensures nested paths are correctly delimited (e.g., "User.Address.City").
// Since all chains now push their name to the context stack via Validate(), the complete
// path is available from ctx.String() directly.
func (b *BaseChain) Fail(ctx *core.Context, value any, reason string) error {
	// Initialize path to the chain's local name as a fallback.
	path := b.Name

	// If the context exists and has path segments, use it as the canonical path source.
	// The chain's name is already pushed to the context stack by the embedding chain's
	// Validate method, so ctx.String() returns the complete qualified path.
	if ctx != nil {
		ctxPath := ctx.String()
		if ctxPath != "" {
			path = ctxPath
		}
	}

	return &core.FieldError{
		Path:   path,
		Value:  value,
		Reason: reason,
	}
}

// resolveValuerIndirect uses reflection to unwrap pointer-to-Valuer types.
// It handles cases like *T where T implements core.Valuer, which the type switch
// in resolveTarget cannot match directly.
func resolveValuerIndirect(target any) (any, bool, error) {
	val := reflect.ValueOf(target)
	// Unwrap pointers until we find a Valuer or run out of indirections.
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, true, nil
		}
		val = val.Elem()
	}
	// Check if the dereferenced value implements Valuer.
	if valuer, ok := val.Interface().(core.Valuer); ok {
		return valuer.Unwrap(), false, nil
	}
	return nil, false, core.ErrMisuse
}
