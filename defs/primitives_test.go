// SPDX-License-Identifier: MIT

package defs_test

import (
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

func TestStringDefBinding(t *testing.T) {
	t.Parallel()

	// Define the unbound template with multiple modifiers.
	template := defs.NewStringDef().
		Min(10).
		Email().
		Max(50)

	t.Run("BindingAppliesMinMax", func(t *testing.T) {
		t.Parallel()
		s := "a@b.c" // Length 5 (fails Min(10) check)

		// Bind the template to a target.
		rule := template.Bind(&s, "email_field")

		// Validate (should fail on Min(10)).
		err := rule.Validate(nil)

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != "length is less than required minimum" {
			t.Fatalf("Expected Min length failure, got: %v", err)
		}

		// Re-test success case.
		s = "long.valid.email@example.com"
		rule = template.Bind(&s, "email_field")
		err = rule.Validate(nil)
		if err != nil {
			t.Fatalf("Expected success, got: %v", err)
		}
	})

	t.Run("BindingAppliesUUID", func(t *testing.T) {
		t.Parallel()

		uuidTemplate := defs.NewStringDef().UUID()

		validID := "a1b2c3d4-e5f6-4000-8000-1234567890ab"
		invalidID := "not-a-uuid"

		// Test success.
		rule := uuidTemplate.Bind(&validID, "id")
		if err := rule.Validate(nil); err != nil {
			t.Errorf("Expected UUID success, got: %v", err)
		}

		// Test failure.
		rule = uuidTemplate.Bind(&invalidID, "id")
		// The rule's Validate() must be called once and its result checked.
		if fe, ok := rule.Validate(nil).(*core.FieldError); !ok || fe.Reason != "invalid UUID format" {
			t.Errorf("Expected UUID format failure, got: %v", rule.Validate(nil))
		}
	})
}

func TestIntDefBinding(t *testing.T) {
	t.Parallel()

	// Define the unbound template.
	template := defs.NewIntDef().
		Min(50).
		Max(100)

	t.Run("BindingAppliesMinMax", func(t *testing.T) {
		t.Parallel()
		val := 49 // Fails Min(50)

		// Bind the template.
		rule := template.Bind(&val, "int_field")

		// Validate.
		err := rule.Validate(nil)

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != "value is less than minimum" {
			t.Fatalf("Expected Min value failure, got: %v", err)
		}

		// Test success.
		val = 75
		rule = template.Bind(&val, "int_field")
		err = rule.Validate(nil)
		if err != nil {
			t.Fatalf("Expected success, got: %v", err)
		}
	})
}
