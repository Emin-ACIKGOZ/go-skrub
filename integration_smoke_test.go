// SPDX-License-Identifier: MIT

package skrub_test

import (
	"sync"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// TestIntegrationSmoke combines struct validation, cross-field ValidateWith,
// accumulate mode, and concurrent access in a single race-detected test.
func TestIntegrationSmoke(t *testing.T) {
	type PasswordPair struct {
		Password        string `validate:"required,min=8"`
		ConfirmPassword string `validate:"required"`
	}

	// Test data
	validPair := PasswordPair{Password: "secret123", ConfirmPassword: "secret123"}
	mismatchPair := PasswordPair{Password: "secret123", ConfirmPassword: "different"}
	// Build the struct validation rule once
	buildRule := func(p *PasswordPair) core.Rule {
		return skrub.DefStruct().
			Field("Password", skrub.DefString().NotEmpty().Min(8)).
			Field("ConfirmPassword", skrub.DefString().NotEmpty()).
			ValidateWith(func(sl core.StructLevel) error {
				pw, _ := sl.FieldValue("Password")
				confirm, _ := sl.FieldValue("ConfirmPassword")
				if pw != confirm {
					sl.ReportError("ConfirmPassword", "must match Password")
				}
				return nil
			}).
			Bind(p)
	}

	// Test 1: Short-circuit mode, valid input
	t.Run("ShortCircuit_Valid", func(t *testing.T) {
		rule := buildRule(&validPair)
		err := skrub.Validate(&validPair, rule)
		if err != nil {
			t.Errorf("expected pass, got: %v", err)
		}
	})

	// Test 2: Short-circuit mode, mismatch
	t.Run("ShortCircuit_Mismatch", func(t *testing.T) {
		rule := buildRule(&mismatchPair)
		err := skrub.Validate(&mismatchPair, rule)
		if err == nil {
			t.Fatal("expected error for mismatch")
		}
	})

	// Test 3: Accumulate mode, short password + mismatch
	t.Run("Accumulate_MultipleErrors", func(t *testing.T) {
		pair := PasswordPair{Password: "ab", ConfirmPassword: "different"}
		rule := buildRule(&pair)
		err := skrub.ValidateWithConfig(core.Config{AccumulateErrors: true}, rule)
		if err == nil {
			t.Fatal("expected errors")
		}
		ves, ok := err.(core.ValidationErrors)
		if !ok {
			t.Fatalf("expected ValidationErrors, got %T", err)
		}
		if len(ves) < 2 {
			t.Fatalf("expected at least 2 errors for short+pw+mismatch, got %d: %v", len(ves), ves)
		}
	})

	// Test 4: Concurrent access — 50 goroutines on the same rule
	t.Run("Concurrent_50Goroutines", func(t *testing.T) {
		rule := buildRule(&validPair)
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := skrub.Validate(&validPair, rule)
				if err != nil {
					t.Errorf("concurrent: expected pass, got: %v", err)
				}
			}()
		}
		wg.Wait()
	})

	// Test 5: Accumulate mode concurrent
	t.Run("Concurrent_Accumulate", func(_ *testing.T) {
		pair := PasswordPair{Password: "ab", ConfirmPassword: "different"}
		rule := buildRule(&pair)
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = skrub.ValidateWithConfig(core.Config{AccumulateErrors: true}, rule)
			}()
		}
		wg.Wait()
	})
}
