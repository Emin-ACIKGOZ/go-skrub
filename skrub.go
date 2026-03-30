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
func Validate(target any, rules ...Rule) error {
	return ValidateWithConfig(target, core.Config{}, rules...)
}

// ValidateWithConfig executes validation rules against a target value using a custom configuration.
// This is the primary entry point for using features like OnWarning or adjusting MaxDepth.
//
// It utilizes a SafePool to retrieve a thread-local Context, ensuring high performance
// and isolation across concurrent goroutines.
func ValidateWithConfig(_ any, cfg core.Config, rules ...Rule) error {
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

	// 3. Ensure the context is returned to the pool after execution.
	// The Put method will automatically call ctx.Reset() via the Resetter interface.
	defer globalCtxPool.Put(ctx)

	// Execute all bound rules
	for _, rule := range rules {
		// Rule.Validate accepts the core.Context alias defined in rule.go.
		if err := rule.Validate(ctx); err != nil {
			return err
		}
	}

	return nil
}
