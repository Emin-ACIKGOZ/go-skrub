// SPDX-License-Identifier: MIT

package chains_test

import (
	"errors"
	"regexp"
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

func TestIntChainNotZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     int
		expectErr bool
		reason    string
	}{
		// Valid non-zero integers
		{"Positive1", 1, false, ""},
		{"Positive100", 100, false, ""},
		{"PositiveMax", 9223372036854775807, false, ""},
		{"Negative1", -1, false, ""},
		{"NegativeMin", -9223372036854775808, false, ""},
		{"NegativeHundred", -100, false, ""},

		// Invalid zero
		{"Zero", 0, true, core.ReasonRequired},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val := tt.value
			chain := chains.NewIntChain(&val, "count")
			err := chain.NotZero().Validate(nil)

			if (err != nil) != tt.expectErr {
				t.Errorf("NotZero %d: expected error=%v, got error=%v", tt.value, tt.expectErr, err)
				return
			}

			if tt.expectErr && err != nil {
				if fe, ok := err.(*core.FieldError); !ok || fe.Reason != tt.reason {
					t.Errorf("NotZero %d: expected reason %q, got %q", tt.value, tt.reason, fe.Reason)
				}
			}
		})
	}
}

func TestIntChainNotZeroWithOtherValidators(t *testing.T) {
	t.Parallel()

	t.Run("NotZero_Then_Min", func(t *testing.T) {
		t.Parallel()
		val := -5
		chain := chains.NewIntChain(&val, "score").NotZero().Min(0)
		err := chain.Validate(nil)
		if err == nil {
			t.Error("Expected Min failure, got nil")
		}
	})

	t.Run("NotZero_Success_With_Max", func(t *testing.T) {
		t.Parallel()
		val := 50
		chain := chains.NewIntChain(&val, "percentage").NotZero().Max(100)
		err := chain.Validate(nil)
		if err != nil {
			t.Errorf("Expected success, got: %v", err)
		}
	})
}

func TestIntChainMatchString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     int
		pattern   string
		expectErr bool
		reason    string
	}{
		// Valid patterns
		{"DigitsOnly1", 123, `^\d+$`, false, ""},
		{"DigitsOnly2", 999, `^\d{3}$`, false, ""},
		{"StartsWith1", 100, `^1.*`, false, ""},
		{"StartsWith2", 500, `^[5-9].*`, false, ""},
		{"EvenNumbers", 42, `.*[02468]$`, false, ""},
		{"OddNumbers", 23, `.*[13579]$`, false, ""},

		// Invalid patterns
		{"NoMatch1", 123, `^[a-z]+$`, true, core.ReasonPattern},
		{"NoMatch2", 999, `^1`, true, core.ReasonPattern},
		{"NoMatch3", 500, `^[0-4]`, true, core.ReasonPattern},
		{"NegativeNoMatch", -42, `^\d+$`, true, core.ReasonPattern},
		{"NegativeNoMatch2", -10, `^\d+$`, true, core.ReasonPattern},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			re := regexp.MustCompile(tt.pattern)
			val := tt.value
			chain := chains.NewIntChain(&val, "id").MatchString(re)
			err := chain.Validate(nil)

			if (err != nil) != tt.expectErr {
				t.Errorf("MatchString %d (pattern %q): expected error=%v, got error=%v", tt.value, tt.pattern, tt.expectErr, err)
				return
			}

			if tt.expectErr && err != nil {
				if fe, ok := err.(*core.FieldError); !ok || fe.Reason != tt.reason {
					t.Errorf("MatchString %d: expected reason %q, got %q", tt.value, tt.reason, fe.Reason)
				}
			}
		})
	}
}

func TestIntChainMatchStringWithOtherValidators(t *testing.T) {
	t.Parallel()

	t.Run("MatchString_Then_Min", func(t *testing.T) {
		t.Parallel()
		re := regexp.MustCompile(`^\d+$`)
		val := 5
		chain := chains.NewIntChain(&val, "id").MatchString(re).Min(10)
		err := chain.Validate(nil)
		if err == nil {
			t.Error("Expected Min failure, got nil")
		}
	})

	t.Run("MatchString_Success", func(t *testing.T) {
		t.Parallel()
		re := regexp.MustCompile(`^[1-9]\d{2}$`) // 100-999
		val := 234
		chain := chains.NewIntChain(&val, "id").MatchString(re)
		err := chain.Validate(nil)
		if err != nil {
			t.Errorf("Expected success, got: %v", err)
		}
	})
}
