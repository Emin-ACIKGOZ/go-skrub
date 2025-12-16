// SPDX-License-Identifier: MIT

package core_test

import (
	"errors"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

func TestErrorFormatting(t *testing.T) {
	t.Parallel()

	t.Run("FieldErrorSimple", func(t *testing.T) {
		t.Parallel()
		fe := &core.FieldError{Path: "user.name", Reason: "must not be empty"}
		expected := "user.name: must not be empty"
		if fe.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, fe.Error())
		}
	})

	t.Run("FieldErrorNoPath", func(t *testing.T) {
		t.Parallel()
		fe := &core.FieldError{Reason: "internal failure"}
		expected := "internal failure"
		if fe.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, fe.Error())
		}
	})

	t.Run("FieldErrorWithUnwrap", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("underlying DB error")
		fe := &core.FieldError{Path: "id", Reason: "not found", Cause: cause}

		if fe.Unwrap() != cause {
			t.Error("Unwrap did not return the correct underlying cause.")
		}
	})

	t.Run("RecursionError", func(t *testing.T) {
		t.Parallel()
		re := &core.RecursionError{Path: "data.[5].sub", Depth: 101, MaxDepth: 100}
		expected := "skrub: recursion limit exceeded at data.[5].sub (depth 101 > 100)"
		if re.Error() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, re.Error())
		}
	})
}

func TestNewContextDefaults(t *testing.T) {
	t.Parallel()

	// Default config (MaxDepth=0) initializes MaxDepth to 100.
	ctx := core.NewContext(core.Config{})

	if ctx.Cfg.MaxDepth != 100 {
		t.Errorf("Expected default MaxDepth 100, got %d", ctx.Cfg.MaxDepth)
	}
	if ctx.Path != "" {
		t.Errorf("Expected root path to be empty, got %s", ctx.Path)
	}
	if ctx.Depth != 0 {
		t.Errorf("Expected root depth to be 0, got %d", ctx.Depth)
	}
}

func TestContextJoinPath(t *testing.T) {
	t.Parallel()

	ctx := core.NewContext(core.Config{MaxDepth: 10})

	if ctx.Path != "" {
		t.Fatal("Setup failed: Path must be empty")
	}

	t.Run("FieldNotation", func(t *testing.T) {
		ctx1, _ := ctx.Enter("User")
		if ctx1.Path != "User" {
			t.Errorf("Expected 'User', got '%s'", ctx1.Path)
		}

		ctx2, _ := ctx1.Enter("Address")
		if ctx2.Path != "User.Address" {
			t.Errorf("Expected 'User.Address', got '%s'", ctx2.Path)
		}
	})

	t.Run("ArrayNotation", func(t *testing.T) {
		ctx1, _ := ctx.Enter("Items")

		// Array index should append directly without a preceding dot.
		ctx2, _ := ctx1.Enter("[0]")
		if ctx2.Path != "Items[0]" {
			t.Errorf("Expected 'Items[0]', got '%s'", ctx2.Path)
		}

		// Field inside array should use dot notation.
		ctx3, _ := ctx2.Enter("ID")
		if ctx3.Path != "Items[0].ID" {
			t.Errorf("Expected 'Items[0].ID', got '%s'", ctx3.Path)
		}
	})
}

func TestContextRecursionSafety(t *testing.T) {
	t.Parallel()

	const maxDepth = 5
	const warningThreshold = 3

	var warningCount int

	cfg := core.Config{
		MaxDepth:         maxDepth,
		WarningThreshold: warningThreshold,
		OnWarning: func(path string, depth int) {
			warningCount++
			if depth != warningThreshold {
				t.Errorf("Warning fired at incorrect depth: %d", depth)
			}
			// Expected path must be the full path "L1.L2.L3".
			if path != "L1.L2.L3" {
				t.Errorf("Warning fired at incorrect path. Expected 'L1.L2.L3', got '%s'", path)
			}
		},
	}

	ctx := core.NewContext(cfg)

	// 1. Should succeed and proceed to depth 3 (WarningThreshold).
	ctxL1, _ := ctx.Enter("L1")
	ctxL2, _ := ctxL1.Enter("L2")
	ctxL3, err := ctxL2.Enter("L3") // Warning fires here (Depth 3).

	if err != nil {
		t.Fatalf("Expected success at depth 3, got: %v", err)
	}

	// 2. Warning check: L3 is depth 3 (threshold).
	if warningCount != 1 {
		t.Errorf("Expected 1 warning fired, got %d", warningCount)
	}

	// 3. Should continue past warning up to max depth (L5).
	ctxL4, _ := ctxL3.Enter("L4")
	ctxL5, _ := ctxL4.Enter("L5") // Depth 5 (MaxDepth).

	if ctxL5.Depth != maxDepth {
		t.Errorf("Failed to reach max depth %d", maxDepth)
	}

	// 4. Hard Stop check (L6).
	_, err = ctxL5.Enter("L6") // Depth 6 (Exceeds MaxDepth).

	if _, ok := err.(*core.RecursionError); !ok {
		t.Errorf("Expected RecursionError at depth 6, got: %v", err)
	}

	// Verify error details.
	if re, ok := err.(*core.RecursionError); ok && re.MaxDepth != maxDepth {
		t.Errorf("RecursionError MaxDepth incorrect, got %d", re.MaxDepth)
	}
}
