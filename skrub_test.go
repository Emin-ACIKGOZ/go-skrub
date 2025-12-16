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

// NewMockRule creates a MockRule that returns the specified error.
// A nil error signifies successful validation.
func NewMockRule(errToReturn error) *MockRule {
	return &MockRule{
		ValidateFn: func(_ *core.Context) error {
			return errToReturn
		},
	}
}

// Validate executes the mock function and increments the call count.
func (m *MockRule) Validate(ctx *core.Context) error {
	m.CallCount++
	return m.ValidateFn(ctx)
}

// TestValidate_SuccessAndDefaultConfig verifies basic successful validation
// and implicit use of the default core.Config.
func TestValidate_SuccessAndDefaultConfig(t *testing.T) {
	t.Parallel()

	rule1 := NewMockRule(nil)
	rule2 := NewMockRule(nil)

	err := skrub.Validate(struct{}{}, rule1, rule2)

	if err != nil {
		t.Fatalf("Validate failed unexpectedly on successful rules: %v", err)
	}
	if rule1.CallCount != 1 || rule2.CallCount != 1 {
		t.Errorf("Expected both rules to be called once. Rule1: %d, Rule2: %d", rule1.CallCount, rule2.CallCount)
	}
}

// TestValidate_ErrorHandling verifies that validation short-circuits
// upon encountering the first validation error.
func TestValidate_ErrorHandling(t *testing.T) {
	t.Parallel()

	rule1 := NewMockRule(nil)
	rule2 := NewMockRule(ErrMockRuleFailed)
	rule3 := NewMockRule(nil) // Should not be called

	err := skrub.Validate(struct{}{}, rule1, rule2, rule3)

	if !errors.Is(err, ErrMockRuleFailed) {
		t.Fatalf("Expected %v, got %v", ErrMockRuleFailed, err)
	}

	if rule1.CallCount != 1 || rule2.CallCount != 1 {
		t.Errorf("Rule 1 and 2 must be called once. Rule1: %d, Rule2: %d", rule1.CallCount, rule2.CallCount)
	}
	if rule3.CallCount != 0 {
		t.Errorf("Rule 3 must NOT be called due to short-circuiting. Got %d", rule3.CallCount)
	}
}

// TestValidateWithConfig_ConfigurationDelegation verifies that core.Config settings
// (MaxDepth, WarningThreshold, OnWarning) are correctly delegated and enforced.
func TestValidateWithConfig_ConfigurationDelegation(t *testing.T) {
	t.Parallel()

	const testMaxDepth = 5
	var warningFired bool
	var warningDepth int

	// Rule that deliberately triggers the recursion warning and hard stop limits.
	recursionTestRule := &MockRule{
		ValidateFn: func(ctx *core.Context) error {
			// Trigger a warning check at WarningThreshold (MaxDepth - 2 = 3)
			ctxL1, _ := ctx.Enter("L1")
			ctxL2, _ := ctxL1.Enter("L2")
			ctxL3, _ := ctxL2.Enter("L3") // Depth 3: Should trigger warning

			// Trigger a hard stop check at MaxDepth + 1 = 6
			ctxL4, _ := ctxL3.Enter("L4")
			ctxL5, _ := ctxL4.Enter("L5")
			_, err := ctxL5.Enter("L6") // Depth 6: Should fail

			return err
		},
	}

	cfg := core.Config{
		MaxDepth:         testMaxDepth,
		WarningThreshold: testMaxDepth - 2,
		OnWarning: func(_ string, depth int) { // path is unused
			warningFired = true
			warningDepth = depth
		},
	}

	err := skrub.ValidateWithConfig(nil, cfg, recursionTestRule)

	// 1. Check for the Recursion hard stop error
	re, ok := err.(*core.RecursionError)
	if !ok {
		t.Fatalf("Expected *core.RecursionError, got %v", err)
	}
	if re.MaxDepth != testMaxDepth {
		t.Errorf("RecursionError reported incorrect MaxDepth. Got %d, want %d", re.MaxDepth, testMaxDepth)
	}

	// 2. Check the warning callback execution
	if !warningFired {
		t.Error("OnWarning callback was not fired.")
	}
	if warningDepth != testMaxDepth-2 {
		t.Errorf("OnWarning fired at incorrect depth. Got %d, want %d", warningDepth, testMaxDepth-2)
	}
}

// TestValidateWithConfig_TargetIgnored ensures that the target parameter,
// regardless of its type (nil, pointer, value), is correctly handled by ValidateWithConfig.
func TestValidateWithConfig_TargetIgnored(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target any
	}{
		{"Nil Target", nil},
		{"Pointer Target", &struct{ Name string }{}},
		{"Value Target", 42},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a fresh rule for every parallel sub-test.
			rule := NewMockRule(nil)

			err := skrub.ValidateWithConfig(tt.target, core.Config{}, rule)

			if err != nil {
				t.Fatalf("Validation failed for target %s: %v", tt.name, err)
			}
			if rule.CallCount != 1 {
				t.Errorf("Rule was not called exactly once.")
			}
		})
	}
}
