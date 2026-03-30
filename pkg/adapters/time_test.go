// SPDX-License-Identifier: MIT

package adapters_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/Emin-ACIKGOZ/go-skrub"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/adapters"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// Define a test time known to be valid.
var testTime = time.Date(2025, time.December, 15, 10, 30, 0, 0, time.UTC)

func TestTimeAdapterUnwrap(t *testing.T) {
	t.Parallel()

	t.Run("DefaultRFC3339Layout", func(t *testing.T) {
		t.Parallel()

		adapter := adapters.Time(testTime)

		// Expected RFC3339 format.
		expected := "2025-12-15T10:30:00Z"

		unwrapped := adapter.Unwrap()

		if s, ok := unwrapped.(string); !ok || s != expected {
			t.Fatalf("Expected Unwrap to return RFC3339 '%s', got '%v'", expected, unwrapped)
		}
	})

	t.Run("CustomLayout", func(t *testing.T) {
		t.Parallel()
		// Layout for "15 Dec 2025".
		layout := "02 Jan 2006"
		adapter := adapters.TimeWithLayout(testTime, layout)

		expected := "15 Dec 2025"
		unwrapped := adapter.Unwrap()

		if s, ok := unwrapped.(string); !ok || s != expected {
			t.Fatalf("Expected Unwrap to return custom format '%s', got '%v'", expected, unwrapped)
		}
	})
}

func TestTimeAdapterIntegration(t *testing.T) {
	t.Parallel()

	t.Run("ValidationSuccess", func(t *testing.T) {
		t.Parallel()
		// Use a simple, non-standard layout.
		layout := time.RubyDate
		adapter := adapters.TimeWithLayout(testTime, layout)

		// Rule: String must start with the weekday abbreviation (Mon).
		// Pass pre-compiled *regexp.Regexp
		re := regexp.MustCompile(`^Mon`)
		rule := skrub.String(adapter, "date").Pattern(re)

		if err := rule.Validate(nil); err != nil {
			t.Fatalf("Expected validation success, got error: %v", err)
		}
	})

	t.Run("ValidationFailure", func(t *testing.T) {
		t.Parallel()
		// Use an incorrect pattern that should fail.
		layout := time.ANSIC
		adapter := adapters.TimeWithLayout(testTime, layout)

		// Rule: String must start with "Sun" (it starts with Mon).
		// Pass pre-compiled *regexp.Regexp
		re := regexp.MustCompile(`^Sun`)
		rule := skrub.String(adapter, "date").Pattern(re)

		err := rule.Validate(nil)

		// skrub.String defaults to setting the ReasonPattern in chains/string.go
		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != core.ReasonPattern {
			t.Fatalf("Expected pattern failure, got: %v", err)
		}
	})
}
