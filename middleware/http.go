// SPDX-License-Identifier: MIT

// Package middleware provides utilities for integrating go-skrub into standard HTTP handlers.
// It includes a mechanism for defining validation logic and attaching error handling hooks
// to control the response flow when validation fails.
package middleware

import (
	"net/http"
)

// Hooks defines the lifecycle callbacks used by the validation middleware,
// primarily for defining global or route-specific error handling logic.
// Hooks are executed sequentially when the Validate function encounters a validation error.
type Hooks struct {
	// OnError is a slice of functions executed in order when validation fails.
	// This allows stacking logic such as logging, error transformation, and response formatting.
	OnError []func(w http.ResponseWriter, r *http.Request, err error)
}

// NewHooks creates and returns a default hook registry with an empty OnError chain.
func NewHooks() *Hooks {
	return &Hooks{
		OnError: make([]func(w http.ResponseWriter, r *http.Request, err error), 0),
	}
}

// Compose adds new error handlers to the end of the execution chain.
// Handlers are executed in the order they are composed when a validation error occurs.
func (h *Hooks) Compose(next ...func(w http.ResponseWriter, r *http.Request, err error)) {
	h.OnError = append(h.OnError, next...)
}

// Validate provides a standard HTTP middleware wrapper for executing go-skrub validation.
//
// If the validator function returns nil, the request proceeds to the next handler (next).
// If the validator returns an error (e.g., *core.FieldError), the configured OnError hooks
// are executed sequentially, and the handler chain is terminated.
// If no hooks are configured on failure, Validate falls back to a 400 Bad Request response.
func (h *Hooks) Validate(validator func(r *http.Request) error, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Execute validation logic.
		if err := validator(r); err != nil {
			// 2. Validation Failed: Execute Hooks.
			if len(h.OnError) > 0 {
				for _, hook := range h.OnError {
					// Execute hook. The hook is responsible for writing the response and handling errors.
					hook(w, r, err)
				}
			} else {
				// Fallback error response if no hooks are configured.
				http.Error(w, "Validation failed", http.StatusBadRequest)
			}
			return // Stop the chain after handling the error.
		}

		// 3. Validation Succeeded: Proceed to business logic.
		next.ServeHTTP(w, r)
	})
}
