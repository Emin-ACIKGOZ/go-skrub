// SPDX-License-Identifier: MIT

// Package core defines the fundamental interfaces, types, errors, and context management
// for the go-skrub validation engine. This package serves as the single source of truth
// to break architectural import cycles.
package core

import (
	"errors"
	"fmt"
	"strings"
)

// =============================================================================
// 1. Errors & Constants
// =============================================================================

var (
	// ErrMisuse indicates the library was used incorrectly (e.g., passing nil pointers).
	ErrMisuse = errors.New("skrub: misuse of API")

	// ErrConcurrencyViolation is returned when a validation chain is accessed concurrently.
	ErrConcurrencyViolation = errors.New("skrub: concurrent access violation detected")

	// ErrPoolExhausted is returned by SafePool when it is empty and configured as NonBlocking.
	ErrPoolExhausted = errors.New("skrub: pool exhausted")
)

// RecursionError is returned when validation exceeds the configured MaxDepth.
type RecursionError struct {
	Path     string
	Depth    int
	MaxDepth int
}

// Error returns a formatted message indicating the recursion limit was exceeded.
func (e *RecursionError) Error() string {
	return fmt.Sprintf("skrub: recursion limit exceeded at %s (depth %d > %d)", e.Path, e.Depth, e.MaxDepth)
}

// FieldError represents a validation failure on a specific field.
type FieldError struct {
	Path   string
	Value  any
	Reason string
	Cause  error
}

// Error returns the formatted path and reason for the validation failure.
// If Path is empty, it returns only the reason.
func (e *FieldError) Error() string {
	if e.Path == "" {
		return e.Reason
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Reason)
}

// Unwrap supports standard Go error wrapping, returning the underlying Cause.
func (e *FieldError) Unwrap() error { return e.Cause }

// NewFieldError creates a simplified FieldError without a wrapped cause.
func NewFieldError(path string, value any, reason string) *FieldError {
	return &FieldError{
		Path:   path,
		Value:  value,
		Reason: reason,
	}
}

// =============================================================================
// 2. Interfaces
// =============================================================================

// Valuer allows custom types (adapters) to expose primitives for validation,
// bypassing complex reflection logic.
type Valuer interface {
	Unwrap() any
}

// Rule is the fundamental interface for a bound, stateful validation chain.
// The Validate method executes the rules against the bound target.
type Rule interface {
	Validate(ctx *Context) error
}

// Resetter allows a Rule to clear its internal state, enabling safe pooling and reuse.
// The Reset method is called before an item is returned to the pool.
type Resetter interface {
	Reset()
}

// Template represents a definition of validation logic (unbound) that can be
// bound to a specific target later.
type Template interface {
	// Bind creates a stateful Rule instance targeted at the provided value.
	Bind(target any, name string) Rule
}

// =============================================================================
// 3. Context & Configuration
// =============================================================================

// Config defines the runtime configuration for validation, controlling safety
// mechanisms like recursion limits and warnings.
type Config struct {
	// MaxDepth is the maximum depth allowed for nested validation (recursion hard stop).
	// The default is 100 if set to zero.
	MaxDepth int
	// WarningThreshold specifies the depth at which the OnWarning hook is executed (soft warning).
	WarningThreshold int
	// OnWarning is an optional callback executed when the WarningThreshold is reached.
	OnWarning func(path string, depth int)
}

// Context maintains the state of a validation request as it traverses the data structure.
// It is used to track the recursion depth and the field path.
type Context struct {
	// Path is the current dot-notation path (e.g., "User.Address[0].City").
	Path string
	// Depth is the current recursion level (0 is root).
	Depth int
	// Cfg holds the immutable runtime configuration.
	Cfg Config
}

// NewContext initializes a root context, applying default configuration if necessary.
func NewContext(cfg Config) *Context {
	if cfg.MaxDepth == 0 {
		cfg.MaxDepth = 100
	}
	return &Context{
		Path:  "",
		Depth: 0,
		Cfg:   cfg,
	}
}

// Enter creates a child context for a nested field or element and increments the recursion depth.
//
// It performs critical safety checks:
// 1. MaxDepth (Hard Stop): Returns a RecursionError if the new depth exceeds Cfg.MaxDepth.
// 2. WarningThreshold (Soft Warning): Executes the Cfg.OnWarning callback if the new depth equals Cfg.WarningThreshold.
func (c *Context) Enter(name string) (*Context, error) {
	newDepth := c.Depth + 1

	// Hard Stop
	if newDepth > c.Cfg.MaxDepth {
		return nil, &RecursionError{
			Path:     c.joinPath(name),
			Depth:    newDepth,
			MaxDepth: c.Cfg.MaxDepth,
		}
	}

	// Soft Warning
	if c.Cfg.WarningThreshold > 0 && newDepth == c.Cfg.WarningThreshold {
		if c.Cfg.OnWarning != nil {
			c.Cfg.OnWarning(c.joinPath(name), newDepth)
		}
	}

	return &Context{
		Path:  c.joinPath(name),
		Depth: newDepth,
		Cfg:   c.Cfg,
	}, nil
}

// joinPath correctly formats the dot-notation path, handling array indices ([0])
// to prevent incorrect paths like "Field.[0]".
func (c *Context) joinPath(name string) string {
	if c.Path == "" {
		return name
	}
	if name == "" {
		return c.Path
	}
	if strings.HasPrefix(name, "[") {
		// Array notation: append directly (e.g., "Items[0]")
		return c.Path + name
	}
	// Field notation: use dot (e.g., "User.Name")
	return c.Path + "." + name
}
