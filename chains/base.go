// SPDX-License-Identifier: MIT

// Package chains implements bound validation rules that hold state and target specific memory addresses.
//
// These chains are designed to be bound to a specific Go value (via a target address)
// at creation time and apply a set of validation rules against that value.
package chains

import (
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

// Fail constructs a core.FieldError using the provided validation value and reason,
// correctly resolving the full error Path based on the current core.Context and the chain's Name.
//
// The resolution logic ensures nested paths are correctly delimited (e.g., "User.Address.City").
func (b *BaseChain) Fail(ctx *core.Context, value any, reason string) error {
	// Initialize path to its default value (the chain's local name).
	path := b.Name

	// Check if the context exists and has a non-empty path.
	if ctx != nil && ctx.Path != "" {
		if b.Name != "" {
			// Case 1: Both context path and chain name are present (e.g., "user.age").
			path = ctx.Path + "." + b.Name
		} else {
			// Case 2: Context path is present, but chain name is empty.
			// The current chain represents a rule on the value pointed to by ctx.Path.
			path = ctx.Path
		}
	}

	return &core.FieldError{
		Path:   path,
		Value:  value,
		Reason: reason,
	}
}
