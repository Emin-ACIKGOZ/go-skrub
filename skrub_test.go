// SPDX-License-Identifier: MIT

package skrub_test

import (
	"errors"
	"testing"

	skrub "github.com/Emin-ACIKGOZ/go-skrub"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// ErrMockRuleFailed is the canonical error returned by a failing mock rule.
var ErrMockRuleFailed = errors.New("mock rule validation failed")

// MockRule tracks validation attempts and returns a configured error.
type MockRule struct {
	ValidateFn func(ctx *core.Context) error
	CallCount  int
}

// Verify MockRule implements core.Rule.
var _ core.Rule = (*MockRule)(nil)

// Validate executes the mock function and increments the call count.
func (m *MockRule) Validate(ctx *core.Context) error {
	m.CallCount++
	if m.ValidateFn != nil {
		return m.ValidateFn(ctx)
	}
	return nil
}

// TestValidate_SuccessAndDefaultConfig verifies basic successful validation
// and ensures every rule provided is actually executed.
func TestValidate_SuccessAndDefaultConfig(t *testing.T) {
	t.Parallel()

	rule1 := &MockRule{}
	rule2 := &MockRule{}

	// Target struct is ignored by the engine but required by the signature.
	err := skrub.Validate(struct{}{}, rule1, rule2)

	if err != nil {
		t.Fatalf("Validate failed unexpectedly: %v", err)
	}
	if rule1.CallCount != 1 || rule2.CallCount != 1 {
		t.Errorf("Rules were not called exactly once. Rule1: %d, Rule2: %d", rule1.CallCount, rule2.CallCount)
	}
}

// TestValidate_ErrorHandling verifies that validation short-circuits
// upon encountering the first validation error.
func TestValidate_ErrorHandling(t *testing.T) {
	t.Parallel()

	rule1 := &MockRule{}
	rule2 := &MockRule{ValidateFn: func(_ *core.Context) error { return ErrMockRuleFailed }}
	rule3 := &MockRule{} // Should not be called

	err := skrub.Validate(struct{}{}, rule1, rule2, rule3)

	if !errors.Is(err, ErrMockRuleFailed) {
		t.Fatalf("Expected %v, got %v", ErrMockRuleFailed, err)
	}

	// Logic Validation: Prove that Rule 1 and 2 ran, but Rule 3 was skipped.
	if rule1.CallCount != 1 || rule2.CallCount != 1 {
		t.Errorf("Pre-error rules failed to execute. R1: %d, R2: %d", rule1.CallCount, rule2.CallCount)
	}
	if rule3.CallCount != 0 {
		t.Errorf("Regression: Engine failed to short-circuit. Rule 3 called %d times", rule3.CallCount)
	}
}

// TestValidateWithConfig_ConfigurationDelegation verifies that core.Config settings
// are correctly enforced and that the Mutable Stack provides correct path strings.
func TestValidateWithConfig_ConfigurationDelegation(t *testing.T) {
	t.Parallel()

	const testMaxDepth = 5
	var warningFired bool
	var warningDepth int
	var capturedPath string

	// Rule that deliberately triggers the recursion warning and hard stop limits.
	recursionTestRule := &MockRule{
		ValidateFn: func(ctx *core.Context) error {
			// Trigger a warning check at Threshold 3
			_ = ctx.Push("L1")
			_ = ctx.Push("L2")
			_ = ctx.Push("L3") // Warning fires here

			// Trigger a hard stop check at MaxDepth + 1
			_ = ctx.Push("L4")
			_ = ctx.Push("L5")
			return ctx.Push("L6") // Depth 6: Hard stop
		},
	}

	cfg := core.Config{
		MaxDepth:         testMaxDepth,
		WarningThreshold: 3,
		OnWarning: func(path string, depth int) {
			warningFired = true
			warningDepth = depth
			capturedPath = path
		},
	}

	err := skrub.ValidateWithConfig(cfg, recursionTestRule)

	// 1. Verify Hard Stop logic
	re, ok := err.(*core.RecursionError)
	if !ok {
		t.Fatalf("Expected *core.RecursionError, got %v", err)
	}
	if re.MaxDepth != testMaxDepth {
		t.Errorf("RecursionError reported incorrect MaxDepth: %d", re.MaxDepth)
	}

	// 2. Verify Warning Logic and Path Integrity
	if !warningFired {
		t.Error("OnWarning callback was never fired.")
	}
	if warningDepth != 3 {
		t.Errorf("OnWarning fired at depth %d, want 3", warningDepth)
	}
	// This proves the Mutable Stack produces correct paths during validation tree traversal.
	if capturedPath != "L1.L2.L3" {
		t.Errorf("Warning path integrity failure. Got %q, want %q", capturedPath, "L1.L2.L3")
	}
}

// TestValidateWithConfig_NoTargetParameter ensures validation works without a target parameter.
func TestValidateWithConfig_NoTargetParameter(t *testing.T) {
	t.Parallel()

	t.Run("Direct Call", func(t *testing.T) {
		rule := &MockRule{}
		err := skrub.ValidateWithConfig(core.Config{}, rule)
		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}
		if rule.CallCount != 1 {
			t.Error("Rule was not called.")
		}
	})
}

func TestValidate_GlobalPoolPanicRecovery(t *testing.T) {
	badRule := &MockRule{
		ValidateFn: func(_ *core.Context) error {
			panic("engine crash")
		},
	}

	// If the pool fails to recover contexts during panics, it will deadlock.
	for i := 0; i < 150; i++ {
		func() {
			defer func() { _ = recover() }()
			_ = skrub.Validate(nil, badRule)
		}()
	}

	successRule := &MockRule{}
	err := skrub.Validate(nil, successRule)
	if err != nil {
		t.Errorf("Global pool deadlocked after panics: %v", err)
	}
}
