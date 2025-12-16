// SPDX-License-Identifier: MIT

// Package adapters provides core.Valuer implementations for common custom types,
// allowing these types to be validated without relying on runtime reflection.
package adapters

import (
	"time"
)

// TimeAdapter wraps a time.Time value to make it compatible with validation chains,
// primarily chains that operate on strings (e.g., chains.StringChain).
//
// By implementing the core.Valuer interface, TimeAdapter allows the validation
// engine to "unwrap" the time value into a formatted string, enabling the use of
// string-based validation rules like MinLen, MaxLen, or Pattern.
type TimeAdapter struct {
	// Val is the time.Time value being wrapped.
	Val time.Time
	// Layout specifies the format used to convert Val to a string.
	// If Layout is empty, it defaults to time.RFC3339.
	Layout string
}

// Unwrap satisfies the core.Valuer interface.
//
// Unwrap returns the formatted string representation of the time using the
// specified TimeAdapter.Layout or the time.RFC3339 default if Layout is empty.
func (t *TimeAdapter) Unwrap() any {
	layout := t.Layout
	if layout == "" {
		layout = time.RFC3339
	}
	return t.Val.Format(layout)
}

// Time creates and returns a new TimeAdapter for a time.Time value.
// It uses time.RFC3339 format by default.
//
// Example:
//
//	skrub.String(adapters.Time(createdAt), "created_at").Pattern(^\d{4}-\d{2})
func Time(t time.Time) *TimeAdapter {
	return &TimeAdapter{Val: t}
}

// TimeWithLayout creates and returns a new TimeAdapter with a specific string format layout.
//
// Example:
//
//	skrub.String(adapters.TimeWithLayout(t, time.RubyDate), "ruby_date")
func TimeWithLayout(t time.Time, layout string) *TimeAdapter {
	return &TimeAdapter{Val: t, Layout: layout}
}
