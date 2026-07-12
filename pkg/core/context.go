// SPDX-License-Identifier: MIT

package core

import (
	"errors"
	"strconv"
	"strings"
)

const (
	defaultStackCapacity     = 8
	averagePathSegmentLength = 5
	defaultMaxDepth          = 100
)

var (
	// ErrMisuse indicates incorrect usage of the skrub API.
	ErrMisuse = errors.New("skrub: misuse of API")

	// ErrConcurrencyViolation indicates that a validation chain
	// was used concurrently in violation of its execution guard.
	ErrConcurrencyViolation = errors.New("skrub: concurrent access violation detected")

	// ErrPoolExhausted indicates that an internal object pool
	// could not provide a reusable instance.
	ErrPoolExhausted = errors.New("skrub: pool exhausted")
)

// Config defines runtime behavior for validation Context execution.
type Config struct {
	// MaxDepth defines the maximum allowed recursion depth.
	MaxDepth int

	// WarningThreshold triggers OnWarning when depth reaches this value.
	WarningThreshold int

	// OnWarning is called when WarningThreshold is reached.
	OnWarning func(path string, depth int)

	// AccumulateErrors enables collection of all validation errors instead
	// of short-circuiting on the first failure. When true, ValidateWithConfig
	// returns a ValidationErrors containing every field error.
	AccumulateErrors bool
}

type pathSegment struct {
	Key   string
	Index int
	IsIdx bool
}

// Context tracks validation traversal state, including path and depth.
// It enforces recursion limits, error accumulation, and supports
// structured path generation.
type Context struct {
	stack  []pathSegment
	depth  int
	cfg    Config
	errors []*FieldError
}

// NewContext creates a new validation Context using the provided configuration.
// If MaxDepth is zero, it defaults to 100.
func NewContext(cfg Config) *Context {
	if cfg.MaxDepth == 0 {
		cfg.MaxDepth = defaultMaxDepth
	}
	return &Context{
		stack: make([]pathSegment, 0, defaultStackCapacity),
		depth: 0,
		cfg:   cfg,
	}
}

// Push appends a field key to the current validation path.
// It returns an error if the recursion depth exceeds MaxDepth.
func (c *Context) Push(key string) error {
	c.depth++
	if err := c.checkDepth(key); err != nil {
		c.depth--
		return err
	}
	c.stack = append(c.stack, pathSegment{Key: key, IsIdx: false})
	return nil
}

// PushIndex appends an index segment to the current validation path.
// It returns an error if the recursion depth exceeds MaxDepth.
func (c *Context) PushIndex(index int) error {
	c.depth++
	if err := c.checkDepth(""); err != nil {
		c.depth--
		return err
	}
	c.stack = append(c.stack, pathSegment{Index: index, IsIdx: true})
	return nil
}

// Pop removes the most recently pushed path segment.
// It includes defensive boundary checks to prevent slice underflow panics
// and negative depth counters.
func (c *Context) Pop() {
	if len(c.stack) == 0 {
		return
	}
	c.stack = c.stack[:len(c.stack)-1]
	if c.depth > 0 {
		c.depth--
	}
}

// Reset clears the context state for reuse in object pools.
// It resets the stack, depth, errors, and configuration to zero values
// to prevent state leaking between pooled contexts.
func (c *Context) Reset() {
	c.stack = c.stack[:0]
	c.depth = 0
	c.errors = c.errors[:0]
	c.cfg = Config{}
}

// IsAccumulating reports whether this context is configured to collect
// all validation errors instead of short-circuiting.
func (c *Context) IsAccumulating() bool {
	return c.cfg.AccumulateErrors
}

// RecordError appends a field error to the context's accumulator.
// In accumulate mode, validators call this instead of returning immediately.
func (c *Context) RecordError(fe *FieldError) {
	c.errors = append(c.errors, fe)
}

// HasErrors reports whether any errors have been accumulated.
func (c *Context) HasErrors() bool {
	return len(c.errors) > 0
}

// SetAccumulateErrors enables or disables error accumulation mode.
func (c *Context) SetAccumulateErrors(acc bool) {
	c.cfg.AccumulateErrors = acc
}

// Errors returns all accumulated field errors as a ValidationErrors collection.
func (c *Context) Errors() ValidationErrors {
	return ValidationErrors(c.errors)
}

// SetMaxDepth updates the recursion limit for the current context.
func (c *Context) SetMaxDepth(d int) {
	if d == 0 {
		d = defaultMaxDepth
	}
	c.cfg.MaxDepth = d
}

// SetWarningThreshold updates the threshold and callback for recursion warnings.
func (c *Context) SetWarningThreshold(t int, fn func(string, int)) {
	c.cfg.WarningThreshold = t
	c.cfg.OnWarning = fn
}

func (c *Context) checkDepth(currentKeyHint string) error {
	if c.depth > c.cfg.MaxDepth {
		return &RecursionError{
			Path:     c.String() + buildPath(currentKeyHint),
			Depth:    c.depth,
			MaxDepth: c.cfg.MaxDepth,
		}
	}
	if c.cfg.WarningThreshold > 0 && c.depth == c.cfg.WarningThreshold {
		if c.cfg.OnWarning != nil {
			// Build path only when warning is actually triggered
			c.cfg.OnWarning(c.String()+buildPath(currentKeyHint), c.depth)
		}
	}
	return nil
}

func (c *Context) String() string {
	if len(c.stack) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.Grow(len(c.stack) * averagePathSegmentLength)
	for i, seg := range c.stack {
		if seg.IsIdx {
			sb.WriteString("[")
			sb.WriteString(strconv.Itoa(seg.Index))
			sb.WriteString("]")
		} else {
			if i > 0 {
				sb.WriteString(".")
			}
			sb.WriteString(seg.Key)
		}
	}
	return sb.String()
}

func buildPath(key string) string {
	if key == "" {
		return ""
	}
	if strings.HasPrefix(key, "[") {
		return key
	}
	return "." + key
}
