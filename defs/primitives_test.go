// SPDX-License-Identifier: MIT

package defs_test

import (
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

func TestStringDefBinding(t *testing.T) { //nolint:cyclop
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

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != core.ReasonMinLength {
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
		if fe, ok := rule.Validate(nil).(*core.FieldError); !ok || fe.Reason != core.ReasonInvalidUUID {
			t.Errorf("Expected UUID format failure, got: %v", rule.Validate(nil))
		}
	})

	t.Run("BindingAppliesURL", func(t *testing.T) {
		t.Parallel()

		urlTemplate := defs.NewStringDef().URL()

		validURL := "https://api.example.com/v1/users"
		invalidURL := "not-a-url"

		// Test success.
		rule := urlTemplate.Bind(&validURL, "webhook_url")
		if err := rule.Validate(nil); err != nil {
			t.Errorf("Expected URL success, got: %v", err)
		}

		// Test failure.
		rule = urlTemplate.Bind(&invalidURL, "webhook_url")
		if fe, ok := rule.Validate(nil).(*core.FieldError); !ok || fe.Reason != core.ReasonInvalidURL {
			t.Errorf("Expected URL format failure, got: %v", rule.Validate(nil))
		}
	})

	t.Run("BindingComposesURLWithMin", func(t *testing.T) {
		t.Parallel()

		// Compose URL with Min length constraint
		urlTemplate := defs.NewStringDef().Min(15).URL()

		shortURL := "http://a.co" // Length 12, fails Min(15)
		longURL := "https://example.com/path"

		// Test failure on Min.
		rule := urlTemplate.Bind(&shortURL, "url")
		err := rule.Validate(nil)
		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != core.ReasonMinLength {
			t.Errorf("Expected Min length failure, got: %v", err)
		}

		// Test success.
		rule = urlTemplate.Bind(&longURL, "url")
		if err := rule.Validate(nil); err != nil {
			t.Errorf("Expected composed URL validation to succeed, got: %v", err)
		}
	})

	t.Run("BindingAppliesIPv4", func(t *testing.T) {
		t.Parallel()

		ipTemplate := defs.NewStringDef().IPv4()

		validIP := "192.168.1.1"
		invalidIP := "::1" // IPv6, should fail

		// Test success.
		rule := ipTemplate.Bind(&validIP, "ip_address")
		if err := rule.Validate(nil); err != nil {
			t.Errorf("Expected IPv4 success, got: %v", err)
		}

		// Test failure on IPv6.
		rule = ipTemplate.Bind(&invalidIP, "ip_address")
		if fe, ok := rule.Validate(nil).(*core.FieldError); !ok || fe.Reason != core.ReasonInvalidIPv4 {
			t.Errorf("Expected IPv4 format failure, got: %v", rule.Validate(nil))
		}
	})

	t.Run("BindingAppliesIPv6", func(t *testing.T) {
		t.Parallel()

		ipTemplate := defs.NewStringDef().IPv6()

		validIP := "2001:db8::1"
		invalidIP := "127.0.0.1" // IPv4, should fail

		// Test success.
		rule := ipTemplate.Bind(&validIP, "ip_address")
		if err := rule.Validate(nil); err != nil {
			t.Errorf("Expected IPv6 success, got: %v", err)
		}

		// Test failure on IPv4.
		rule = ipTemplate.Bind(&invalidIP, "ip_address")
		if fe, ok := rule.Validate(nil).(*core.FieldError); !ok || fe.Reason != core.ReasonInvalidIPv6 {
			t.Errorf("Expected IPv6 format failure, got: %v", rule.Validate(nil))
		}
	})

	t.Run("BindingAppliesNotEmpty", func(t *testing.T) {
		t.Parallel()

		emptyTemplate := defs.NewStringDef().NotEmpty()

		validStr := "non-empty"
		emptyStr := ""

		// Test success.
		rule := emptyTemplate.Bind(&validStr, "name")
		if err := rule.Validate(nil); err != nil {
			t.Errorf("Expected NotEmpty success, got: %v", err)
		}

		// Test failure on empty.
		rule = emptyTemplate.Bind(&emptyStr, "name")
		if fe, ok := rule.Validate(nil).(*core.FieldError); !ok || fe.Reason != core.ReasonRequired {
			t.Errorf("Expected NotEmpty failure, got: %v", rule.Validate(nil))
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

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != core.ReasonMinValue {
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

	t.Run("BindingAppliesNotZero", func(t *testing.T) {
		t.Parallel()

		zeroTemplate := defs.NewIntDef().NotZero()

		validInt := 42
		zeroInt := 0

		// Test success.
		rule := zeroTemplate.Bind(&validInt, "count")
		if err := rule.Validate(nil); err != nil {
			t.Errorf("Expected NotZero success, got: %v", err)
		}

		// Test failure on zero.
		rule = zeroTemplate.Bind(&zeroInt, "count")
		if fe, ok := rule.Validate(nil).(*core.FieldError); !ok || fe.Reason != core.ReasonRequired {
			t.Errorf("Expected NotZero failure, got: %v", rule.Validate(nil))
		}
	})
}
