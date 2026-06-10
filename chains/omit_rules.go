// SPDX-License-Identifier: MIT

package chains

import (
	"reflect"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// OmitEmptyRule wraps a Rule and skips validation if the value is nil or
// the zero value for its type. Matches go-validator's omitempty behavior.
type OmitEmptyRule struct {
	inner  core.Rule
	target any
}

// NewOmitEmptyRule creates a new OmitEmptyRule wrapping the given rule.
func NewOmitEmptyRule(inner core.Rule, target any) *OmitEmptyRule {
	return &OmitEmptyRule{inner: inner, target: target}
}

// Validate implements core.Rule. Skips if the target value is empty.
func (r *OmitEmptyRule) Validate(ctx *core.Context) error {
	if ctx == nil {
		ctx = core.NewContext(core.Config{})
	}
	if r.isEmpty() {
		return nil
	}
	return r.inner.Validate(ctx)
}

func (r *OmitEmptyRule) isEmpty() bool {
	if r.target == nil {
		return true
	}
	rv := reflect.ValueOf(r.target)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	return rv.IsZero()
}

// OmitNilRule skips validation only if the value is nil.
// Unlike OmitEmptyRule, it does NOT skip zero values like 0 or "".
type OmitNilRule struct {
	inner  core.Rule
	target any
}

// NewOmitNilRule creates a new OmitNilRule wrapping the given rule.
func NewOmitNilRule(inner core.Rule, target any) *OmitNilRule {
	return &OmitNilRule{inner: inner, target: target}
}

// Validate implements core.Rule. Skips if the target value is nil.
func (r *OmitNilRule) Validate(ctx *core.Context) error {
	if ctx == nil {
		ctx = core.NewContext(core.Config{})
	}
	if r.isNil() {
		return nil
	}
	return r.inner.Validate(ctx)
}

func (r *OmitNilRule) isNil() bool {
	if r.target == nil {
		return true
	}
	rv := reflect.ValueOf(r.target)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}

// OmitZeroRule skips validation if the value is zero, with stronger semantics
// than OmitEmptyRule for slices/maps: requires non-nil AND non-empty.
// Matches go-validator's omitzero behavior.
type OmitZeroRule struct {
	inner  core.Rule
	target any
}

// NewOmitZeroRule creates a new OmitZeroRule wrapping the given rule.
func NewOmitZeroRule(inner core.Rule, target any) *OmitZeroRule {
	return &OmitZeroRule{inner: inner, target: target}
}

// Validate implements core.Rule. Skips if the target value is zero.
func (r *OmitZeroRule) Validate(ctx *core.Context) error {
	if ctx == nil {
		ctx = core.NewContext(core.Config{})
	}
	if r.isZero() {
		return nil
	}
	return r.inner.Validate(ctx)
}

func (r *OmitZeroRule) isZero() bool {
	if r.target == nil {
		return true
	}
	rv := reflect.ValueOf(r.target)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Map:
		// omitzero is stricter: non-nil AND non-empty
		return rv.Len() == 0
	default:
		return rv.IsZero()
	}
}

// IsDefaultRule passes validation only when the value IS the zero value.
// This is the opposite of required. Matches go-validator's isdefault tag.
type IsDefaultRule struct {
	inner  core.Rule
	target any
}

// NewIsDefaultRule creates a new IsDefaultRule wrapping the given rule.
func NewIsDefaultRule(inner core.Rule, target any) *IsDefaultRule {
	return &IsDefaultRule{inner: inner, target: target}
}

// Validate implements core.Rule. Passes only if the target value is zero.
func (r *IsDefaultRule) Validate(_ *core.Context) error {
	if r.isZero() {
		return nil
	}
	return &core.FieldError{
		Path:   "",
		Value:  r.target,
		Reason: "value is not default",
	}
}

func (r *IsDefaultRule) isZero() bool {
	if r.target == nil {
		return true
	}
	rv := reflect.ValueOf(r.target)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	return rv.IsZero()
}

// OrRule tries each alternative Rule in order and returns success
// on the first passing alternative. Matches go-validator's | operator.
type OrRule struct {
	alternatives []core.Rule
}

// NewOrRule creates a new OrRule.
func NewOrRule(alternatives []core.Rule) *OrRule {
	return &OrRule{alternatives: alternatives}
}

// Validate implements core.Rule. Tries each alternative; first success wins.
func (r *OrRule) Validate(ctx *core.Context) error {
	if ctx == nil {
		ctx = core.NewContext(core.Config{})
	}
	if len(r.alternatives) == 1 {
		return r.alternatives[0].Validate(ctx)
	}
	for _, alt := range r.alternatives {
		if err := alt.Validate(ctx); err == nil {
			return nil
		}
	}
	return r.alternatives[len(r.alternatives)-1].Validate(ctx)
}
