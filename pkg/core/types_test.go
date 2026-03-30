// SPDX-License-Identifier: MIT

package core_test

import (
	"errors"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// pushPaths is a test helper that sequentially pushes keys to the context,
// immediately failing the test if any push operation yields an error.
func pushPaths(t *testing.T, ctx *core.Context, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if err := ctx.Push(p); err != nil {
			t.Fatalf("Expected success pushing %q, got: %v", p, err)
		}
	}
}

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

	// Default config (MaxDepth=0) should implicitly initialize MaxDepth to 100.
	ctx := core.NewContext(core.Config{})

	if ctx.String() != "" {
		t.Errorf("Expected root path to be empty, got %s", ctx.String())
	}

	// Behavioral test: verify we can push exactly 100 times successfully.
	for i := 0; i < 100; i++ {
		if err := ctx.Push("level"); err != nil {
			t.Fatalf("Unexpected recursion error at default depth %d: %v", i+1, err)
		}
	}

	// The 101st push should trigger the default hard stop.
	err := ctx.Push("overflow")
	if err == nil {
		t.Fatal("Expected recursion error at depth 101, got nil")
	}

	re, ok := err.(*core.RecursionError)
	if !ok || re.MaxDepth != 100 {
		t.Errorf("Expected RecursionError with MaxDepth 100, got: %v", err)
	}
}

func TestContextPathing(t *testing.T) {
	t.Parallel()
	ctx := core.NewContext(core.Config{MaxDepth: 10})

	// Test Field Push
	if err := ctx.Push("User"); err != nil {
		t.Fatalf("failed to push 'User': %v", err)
	}
	if ctx.String() != "User" {
		t.Errorf("expected User, got %s", ctx.String())
	}

	// Test Nested Field
	if err := ctx.Push("Address"); err != nil {
		t.Fatalf("failed to push 'Address': %v", err)
	}
	if ctx.String() != "User.Address" {
		t.Errorf("expected User.Address, got %s", ctx.String())
	}

	// Test Index Push
	if err := ctx.PushIndex(0); err != nil {
		t.Fatalf("failed to push index 0: %v", err)
	}
	if ctx.String() != "User.Address[0]" {
		t.Errorf("expected User.Address[0], got %s", ctx.String())
	}

	ctx.Pop() // Remove [0]
	ctx.Pop() // Remove Address

	if ctx.String() != "User" {
		t.Errorf("expected User after pops, got %s", ctx.String())
	}
}

func TestContextRecursionLimit(t *testing.T) {
	t.Parallel()
	ctx := core.NewContext(core.Config{MaxDepth: 2})

	pushPaths(t, ctx, "L1", "L2")

	err := ctx.Push("L3") // Should exceed MaxDepth 2
	if err == nil {
		t.Fatal("expected recursion error, got nil")
	}
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
	pushPaths(t, ctx, "L1", "L2", "L3") // Warning fires at L3

	// 2. Warning check: L3 is depth 3 (threshold).
	if warningCount != 1 {
		t.Errorf("Expected 1 warning fired, got %d", warningCount)
	}

	// 3. Should continue past warning up to max depth (L5).
	pushPaths(t, ctx, "L4", "L5")

	// 4. Hard Stop check (L6).
	err := ctx.Push("L6") // Depth 6 (Exceeds MaxDepth).
	if err == nil {
		t.Fatal("Expected RecursionError at depth 6, got nil")
	}

	// Verify error details.
	re, ok := err.(*core.RecursionError)
	if !ok {
		t.Fatalf("Expected *core.RecursionError, got %T", err)
	}
	if re.MaxDepth != maxDepth {
		t.Errorf("RecursionError MaxDepth incorrect, got %d", re.MaxDepth)
	}
}
