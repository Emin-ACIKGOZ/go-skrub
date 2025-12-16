// SPDX-License-Identifier: MIT

package skrub

import "github.com/Emin-ACIKGOZ/go-skrub/pkg/core"

// Validate executes a list of validation rules against a target value using default configuration.
//
// Defaults: MaxDepth is 100, and recursion warnings are disabled.
func Validate(target any, rules ...Rule) error {
	return ValidateWithConfig(target, Config{}, rules...)
}

// ValidateWithConfig executes validation rules against a target value using a custom configuration.
// This is the primary entry point for using features like OnWarning or adjusting MaxDepth.
func ValidateWithConfig(_ any, cfg Config, rules ...Rule) error {
	// Initialize root context using the core engine logic
	ctx := core.NewContext(cfg)

	// Execute all bound rules
	for _, rule := range rules {
		// Rule.Validate accepts the core.Context alias defined in rule.go.
		if err := rule.Validate(ctx); err != nil {
			return err
		}
	}

	return nil
}
