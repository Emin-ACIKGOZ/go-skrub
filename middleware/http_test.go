// SPDX-License-Identifier: MIT

package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/middleware"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// errTestValidation is a mock error for testing the hooks.
var errTestValidation = errors.New("test validation failed")

// MockHandler is the final handler in the chain. It records if it was executed.
func MockHandler(_ *testing.T, executed *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*executed = true
		w.WriteHeader(http.StatusOK)
		// Error return value from Write is ignored here for simplicity in test handler.
		_, _ = w.Write([]byte("OK"))
	})
}

// MockValidator returns nil on success, or a core.FieldError on failure.
func MockValidator(shouldFail bool) func(_ *http.Request) error {
	return func(_ *http.Request) error {
		if shouldFail {
			// Use a core.FieldError to simulate real skrub output.
			return core.NewFieldError("field", "value", errTestValidation.Error())
		}
		return nil
	}
}

func TestMiddlewareSuccess(t *testing.T) {
	t.Parallel()

	hooks := middleware.NewHooks()
	nextExecuted := false

	// Middleware setup: Validator succeeds (returns nil).
	handler := hooks.Validate(MockValidator(false), MockHandler(t, &nextExecuted))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Assertions.
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !nextExecuted {
		t.Error("Next handler was not executed on validation success.")
	}
}

func TestMiddlewareFailureWithHooks(t *testing.T) {
	t.Parallel()

	var hookExecuted bool
	var capturedError error

	errorHook := func(w http.ResponseWriter, _ *http.Request, err error) {
		hookExecuted = true
		capturedError = err
		// Custom error formatting: set status and body.
		w.WriteHeader(http.StatusUnauthorized)
		// Error return value from Write is ignored here to satisfy errcheck linter.
		_, _ = w.Write([]byte("CUSTOM ERROR"))
	}

	hooks := middleware.NewHooks()
	hooks.Compose(errorHook) // Add the custom hook.

	nextExecuted := false

	// Middleware setup: Validator fails (returns error).
	handler := hooks.Validate(MockValidator(true), MockHandler(t, &nextExecuted))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Assertions.
	if !hookExecuted {
		t.Error("OnError hook was not executed on validation failure.")
	}
	if nextExecuted {
		t.Error("Next handler should NOT have executed on validation failure.")
	}
	if capturedError.(*core.FieldError).Reason != errTestValidation.Error() {
		t.Errorf("Hook failed to capture the correct error.")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected custom hook status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if rr.Body.String() != "CUSTOM ERROR" {
		t.Errorf("Hook failed to write custom body.")
	}
}

func TestMiddlewareFailureFallback(t *testing.T) {
	t.Parallel()

	// Test the default fallback logic when no hooks are registered.
	hooks := middleware.NewHooks() // No hooks composed.

	nextExecuted := false

	// Middleware setup: Validator fails.
	handler := hooks.Validate(MockValidator(true), MockHandler(t, &nextExecuted))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Assertions.
	if nextExecuted {
		t.Error("Next handler should NOT have executed on validation failure.")
	}
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected fallback status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if rr.Body.String() != "Validation failed\n" {
		t.Errorf("Expected fallback body, got: %s", rr.Body.String())
	}
}

func TestHooksComposeOrder(t *testing.T) {
	t.Parallel()

	var executionOrder []int

	hook1 := func(_ http.ResponseWriter, _ *http.Request, _ error) {
		executionOrder = append(executionOrder, 1)
	}
	hook2 := func(w http.ResponseWriter, _ *http.Request, _ error) {
		executionOrder = append(executionOrder, 2)
		// Stop response here by writing headers.
		w.WriteHeader(http.StatusForbidden)
	}

	hooks := middleware.NewHooks()
	hooks.Compose(hook1, hook2) // Compose them in order 1, 2.

	// Validator fails.
	handler := hooks.Validate(MockValidator(true), MockHandler(t, new(bool)))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)

	handler.ServeHTTP(rr, req)

	// Assertions.
	if len(executionOrder) != 2 || executionOrder[0] != 1 || executionOrder[1] != 2 {
		t.Errorf("Hooks were executed in wrong order. Expected [1, 2], got %v", executionOrder)
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status from hook 2 (%d), got %d", http.StatusForbidden, rr.Code)
	}
}
