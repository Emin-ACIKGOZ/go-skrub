// SPDX-License-Identifier: MIT

package skrub

import (
	"github.com/Emin-ACIKGOZ/go-skrub/internal/pool"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

const (
	// DefaultPoolCapacity is sufficient for most high-concurrency environments.
	// It limits the number of Context instances to prevent unbounded memory growth.
	DefaultPoolCapacity = 128
)

var (
	// globalCtxPool manages reusable Context instances to minimize allocation churn.
	// It is configured to block when empty, providing backpressure under heavy load.
	globalCtxPool = pool.NewSafePool(pool.Config{
		Factory: func() any {
			return core.NewContext(core.Config{})
		},
		Capacity:    DefaultPoolCapacity,
		NonBlocking: false, // Blocking ensures validation requests wait for a context.
	})
)

// Validate executes a list of validation rules against a target value using default configuration.
//
// Defaults: MaxDepth is 100, and recursion warnings are disabled.
func Validate(_ any, rules ...Rule) error {
	return ValidateWithConfig(core.Config{}, rules...)
}

// ValidateWithConfig executes validation rules using a custom configuration.
// This is the primary entry point for using features like OnWarning, MaxDepth,
// and error accumulation.
//
// It utilizes a SafePool to retrieve a thread-local Context, ensuring high performance
// and isolation across concurrent goroutines.
func ValidateWithConfig(cfg core.Config, rules ...Rule) error {
	// 1. Acquire a thread-local context from the global pool.
	item, err := globalCtxPool.Get()
	if err != nil {
		return err
	}
	ctx := item.(*core.Context)

	// 2. Apply the provided configuration.
	// If the user provides a custom MaxDepth, we override the pooled context's config.
	if cfg.MaxDepth != 0 {
		ctx.SetMaxDepth(cfg.MaxDepth)
	}
	ctx.SetWarningThreshold(cfg.WarningThreshold, cfg.OnWarning)
	ctx.SetAccumulateErrors(cfg.AccumulateErrors)

	// 3. Ensure the context is returned to the pool after execution.
	// The Put method will automatically call ctx.Reset() via the Resetter interface.
	defer globalCtxPool.Put(ctx)

	// Execute all bound rules
	for _, rule := range rules {
		if err := rule.Validate(ctx); err != nil {
			if !cfg.AccumulateErrors {
				return err
			}
			// In accumulate mode, chains call emitError which calls RecordError
			// and returns nil. If a non-nil error is returned here, it means
			// the error was NOT emitted through emitError (e.g., ErrConcurrencyViolation
			// from Acquire). Wrap and accumulate it.
			if fe, ok := err.(*core.FieldError); ok {
				ctx.RecordError(fe)
			} else {
				ctx.RecordError(&core.FieldError{
					Path:   "",
					Value:  nil,
					Reason: err.Error(),
				})
			}
		}
	}

	if cfg.AccumulateErrors && ctx.HasErrors() {
		return ctx.Errors()
	}
	return nil
}
