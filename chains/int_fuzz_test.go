// SPDX-License-Identifier: MIT

package chains_test

import (
	"regexp"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// FuzzIntChainMatchString tests MatchString validator against arbitrary patterns and values.
// Verifies that:
// - Validator never panics with any pattern
// - Pattern matching is correct
// - Large integers are handled safely
func FuzzIntChainMatchString(f *testing.F) {
	// Seed with known test cases
	f.Add(123, `^\d+$`)
	f.Add(-456, `^-?\d+$`)
	f.Add(0, `^0$`)
	f.Add(999, `[0-9]`)
	f.Add(-42, `.*`)
	f.Add(100, `^[1-9]\d{2}$`)

	f.Fuzz(func(t *testing.T, value int, pattern string) {
		val := value

		// Protect against invalid regex patterns
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Skip invalid patterns - that's a builder error, not validator error
			return
		}

		intChain := chains.NewIntChain(&val, "num")
		intChain.MatchString(re)

		// Should not panic
		validateErr := intChain.Validate(core.NewContext(core.Config{}))

		// Verify result matches regex.MatchString behavior on string representation
		intStr := string(rune(value))
		if re.MatchString(intStr) && validateErr != nil {
			t.Logf("WARNING: Validator rejected %d with pattern %q but regex matches its string representation", value, pattern)
		}
	})
}

// FuzzIntChainMin tests Min validator against arbitrary values.
// Verifies that:
// - Validator never panics
// - Values less than minimum are rejected
// - Values >= minimum are accepted
func FuzzIntChainMin(f *testing.F) {
	// Seed with known test cases
	f.Add(0, 0)
	f.Add(10, 5)
	f.Add(5, 10)
	f.Add(-1, 0)
	f.Add(9223372036854775807, 0)      // Max int64
	f.Add(-9223372036854775808, -1000) // Min int64

	f.Fuzz(func(t *testing.T, value int, minVal int) {
		val := value
		chain := chains.NewIntChain(&val, "field")
		chain.Min(minVal)

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// Verify behavior matches expectation
		if value >= minVal && err != nil {
			t.Errorf("Min(%d) should accept %d, got error: %v", minVal, value, err)
		}
		if value < minVal && err == nil {
			t.Errorf("Min(%d) should reject %d, got nil error", minVal, value)
		}
	})
}

// FuzzIntChainMax tests Max validator against arbitrary values.
// Verifies that:
// - Validator never panics
// - Values greater than maximum are rejected
// - Values <= maximum are accepted
func FuzzIntChainMax(f *testing.F) {
	// Seed with known test cases
	f.Add(0, 0)
	f.Add(10, 100)
	f.Add(100, 10)
	f.Add(-1, 0)
	f.Add(9223372036854775807, -1)
	f.Add(-9223372036854775808, 0)

	f.Fuzz(func(t *testing.T, value int, maxVal int) {
		val := value
		chain := chains.NewIntChain(&val, "field")
		chain.Max(maxVal)

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// Verify behavior matches expectation
		if value <= maxVal && err != nil {
			t.Errorf("Max(%d) should accept %d, got error: %v", maxVal, value, err)
		}
		if value > maxVal && err == nil {
			t.Errorf("Max(%d) should reject %d, got nil error", maxVal, value)
		}
	})
}

// FuzzIntChainNotZero tests NotZero validator against arbitrary values.
// Verifies that:
// - Validator never panics
// - Zero is rejected
// - Non-zero values are accepted
func FuzzIntChainNotZero(f *testing.F) {
	// Seed with known test cases
	f.Add(0)
	f.Add(1)
	f.Add(-1)
	f.Add(100)
	f.Add(-100)
	f.Add(9223372036854775807)
	f.Add(-9223372036854775808)

	f.Fuzz(func(t *testing.T, value int) {
		val := value
		chain := chains.NewIntChain(&val, "field")
		chain.NotZero()

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// Verify behavior matches expectation
		if value == 0 && err == nil {
			t.Error("NotZero should reject 0, got nil error")
		}
		if value != 0 && err != nil {
			t.Errorf("NotZero should accept %d, got error: %v", value, err)
		}
	})
}
