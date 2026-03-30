// SPDX-License-Identifier: MIT

package chains_test

import (
	"errors"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// MockIntValuer implements core.Valuer and returns a concrete integer value.
type MockIntValuer struct {
	Value int
}

// Unwrap returns the stored Value as an any type.
func (m MockIntValuer) Unwrap() any {
	return m.Value
}

// WrongValuer implements core.Valuer but returns a string.
type WrongValuer struct{}

// Unwrap returns the string "not an int" to trigger type checking misuse errors.
func (w WrongValuer) Unwrap() any { return "not an int" }

func TestIntChainValidation(t *testing.T) {
	t.Parallel()

	t.Run("MinCheckFailure", func(t *testing.T) {
		t.Parallel()
		val := 10
		// Fix: Removed third 'nil' argument to match NewIntChain(target, name)
		chain := chains.NewIntChain(&val, "score")
		err := chain.Min(20).Validate(core.NewContext(core.Config{}))

		if fe, ok := err.(*core.FieldError); !ok || fe.Path != "score" {
			t.Errorf("Expected min failure at 'score', got: %v", err)
		}
	})

	t.Run("MaxCheckSuccess", func(t *testing.T) {
		t.Parallel()
		val := 95
		chain := chains.NewIntChain(&val, "percentage")
		err := chain.Max(100).Validate(core.NewContext(core.Config{}))

		if err != nil {
			t.Errorf("Expected success, got: %v", err)
		}
	})

	t.Run("NilPointerSkip", func(t *testing.T) {
		t.Parallel()
		var ptr *int
		chain := chains.NewIntChain(ptr, "optional_id")
		err := chain.Min(1).Validate(core.NewContext(core.Config{}))

		if err != nil {
			t.Errorf("Expected success for nil pointer, got: %v", err)
		}
	})
}

func TestIntChainValuer(t *testing.T) {
	t.Parallel()

	t.Run("ValuerSuccess", func(t *testing.T) {
		t.Parallel()
		valuer := MockIntValuer{Value: 150}
		chain := chains.NewIntChain(valuer, "limit")
		err := chain.Max(200).Validate(core.NewContext(core.Config{}))

		if err != nil {
			t.Errorf("Expected success with Valuer, got: %v", err)
		}
	})

	t.Run("ValuerTypeMismatch", func(t *testing.T) {
		t.Parallel()
		chain := chains.NewIntChain(WrongValuer{}, "bad_type")
		err := chain.Validate(core.NewContext(core.Config{}))

		if !errors.Is(err, core.ErrMisuse) {
			t.Errorf("Expected ErrMisuse for wrong Valuer return, got: %v", err)
		}
	})
}

func TestIntChain_MisuseGuard(t *testing.T) {
	t.Parallel()
	strVal := "not-an-int"
	chain := chains.NewIntChain(&strVal, "field")

	err := chain.Validate(core.NewContext(core.Config{}))
	if !errors.Is(err, core.ErrMisuse) {
		t.Errorf("Regression: IntChain failed to catch type misuse. Got %v", err)
	}
}
