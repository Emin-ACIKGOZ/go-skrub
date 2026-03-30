// SPDX-License-Identifier: MIT

package chains_test

import (
	"regexp"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// MockStringValuer implements core.Valuer and returns a string.
type MockStringValuer struct {
	Value string
}

func (m MockStringValuer) Unwrap() any {
	return m.Value
}

func TestStringChainValidation(t *testing.T) {
	t.Parallel()

	t.Run("MinLengthFailure", func(t *testing.T) {
		t.Parallel()
		val := "short"
		chain := chains.NewStringChain(&val, "input")
		err := chain.Min(10).Validate(nil)

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != core.ReasonMinLength {
			t.Errorf("Expected min length failure with reason %q, got: %v", core.ReasonMinLength, err)
		}
	})

	t.Run("EmailSuccess", func(t *testing.T) {
		t.Parallel()
		val := "test.user@example.com"
		chain := chains.NewStringChain(&val, "email")
		err := chain.Email().Validate(nil)

		if err != nil {
			t.Errorf("Expected email success, got: %v", err)
		}
	})

	t.Run("UUIDFailure", func(t *testing.T) {
		t.Parallel()
		val := "1234-abcd-5678" // Invalid UUID format
		chain := chains.NewStringChain(&val, "uuid")
		err := chain.UUID().Validate(nil)

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != core.ReasonInvalidUUID {
			t.Errorf("Expected UUID failure with reason %q, got: %v", core.ReasonInvalidUUID, err)
		}
	})

	t.Run("PatternCheck", func(t *testing.T) {
		t.Parallel()
		val := "AB-123"
		chain := chains.NewStringChain(&val, "code")

		re := regexp.MustCompile(`^[A-Z]{2}-\d{3}$`)
		err := chain.Pattern(re).Validate(nil)

		if err != nil {
			t.Errorf("Expected pattern success, got: %v", err)
		}
	})
}

func TestStringChainValuer(t *testing.T) {
	t.Parallel()

	t.Run("ValuerSuccess", func(t *testing.T) {
		t.Parallel()
		valuer := MockStringValuer{Value: "valid@mail.co"}
		chain := chains.NewStringChain(valuer, "email_valuer")
		err := chain.Email().Validate(nil)

		if err != nil {
			t.Errorf("Expected success with Valuer, got: %v", err)
		}
	})
}
