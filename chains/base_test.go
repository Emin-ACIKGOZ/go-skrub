// SPDX-License-Identifier: MIT

package chains_test

import (
	"sync"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// MockChain is a test utility that embeds chains.BaseChain
// to allow direct testing of concurrency control (Acquire/Release)
// and error creation (Fail) without implementing a complete Rule.
type MockChain struct {
	chains.BaseChain
}

// Validate is required to implement the core.Rule interface.
// Validate always returns nil, bypassing validation logic.
func (c *MockChain) Validate(_ *core.Context) error { return nil }

func TestBaseChainConcurrency(t *testing.T) {
	t.Parallel()

	chain := MockChain{chains.BaseChain{Name: "testField"}}
	var wg sync.WaitGroup

	// Acquire the lock successfully.
	if err := chain.Acquire(); err != nil {
		t.Fatalf("Expected initial Acquire to succeed, got %v", err)
	}

	// Test concurrent violation.
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := chain.Acquire()

		if err != core.ErrConcurrencyViolation {
			t.Errorf("Expected ErrConcurrencyViolation, got %v", err)
		}
	}()

	wg.Wait()

	// Release the lock.
	chain.Release()

	// Acquire again (after release) to ensure state is reset.
	if err := chain.Acquire(); err != nil {
		t.Errorf("Expected Acquire after Release to succeed, got %v", err)
	}
	chain.Release() // Clean up.
}

func TestBaseChainReset(t *testing.T) {
	t.Parallel()

	chain := MockChain{chains.BaseChain{Name: "oldName"}}

	// Set state to busy (1) and set a name.
	if err := chain.Acquire(); err != nil {
		t.Fatalf("Setup failed: Could not acquire lock for reset test: %v", err)
	}

	// Reset.
	chain.Reset()

	// Verify name is cleared.
	if chain.Name != "" {
		t.Errorf("Reset failed: Name was not cleared, got %s", chain.Name)
	}

	// Verify state is reset (try to acquire should succeed).
	if err := chain.Acquire(); err != nil {
		t.Errorf("Reset failed: Could not acquire lock after Reset, got %v", err)
	}
	chain.Release() // Clean up.
}

func TestBaseChainFail(t *testing.T) {
	t.Parallel()

	chain := MockChain{chains.BaseChain{Name: "age"}}

	// Create a mocked context to test pathing.
	mockCtx := core.NewContext(core.Config{})
	ctxChild, _ := mockCtx.Enter("user") // Path = "user".

	// Test failure with a context (should result in "user.age").
	err := chain.Fail(ctxChild, 25, "must be over 30")
	fe, ok := err.(*core.FieldError)

	if !ok || fe.Path != "user.age" {
		t.Errorf("Expected Fail path 'user.age', got %s", fe.Path)
	}
	if fe.Value != 25 {
		t.Errorf("Expected Fail value 25, got %v", fe.Value)
	}
}
