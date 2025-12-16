// SPDX-License-Identifier: MIT

package skrub

import "github.com/Emin-ACIKGOZ/go-skrub/pkg/core"

// Valuer aliases the core.Valuer interface, allowing custom types (adapters) to expose
// primitives for validation, bypassing complex reflection logic.
type Valuer = core.Valuer

// Rule aliases the core.Rule interface, defining the fundamental contract for all bound validation logic.
// The Validate method executes the rules against the bound target.
type Rule = core.Rule

// Template aliases the core.Template interface, representing unbound validation logic (definitions)
// that can be bound to a specific target later.
type Template = core.Template

// FieldError aliases the core.FieldError struct, representing a specific validation failure.
// This allows users to inspect the error fields, such as Path, Value, and Reason.
type FieldError = core.FieldError

// Config aliases the core.Config struct, defining the runtime configuration for validation,
// controlling safety mechanisms like recursion limits.
type Config = core.Config

// Context aliases the core.Context struct, which maintains the state of a validation request
// as it traverses the data structure, tracking recursion depth and the field path.
type Context = core.Context
