// SPDX-License-Identifier: MIT

package chains_test

import (
	"fmt"
	"sync"
	"sync/atomic"
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
}

// --- Atomic State Machine Audit ---

func TestBaseChain_StateRace(t *testing.T) {
	// Do not use t.Parallel() here to maximize CPU contention on the atomic swap.
	const (
		workers    = 1000
		iterations = 100
	)

	chain := &MockChain{chains.BaseChain{Name: "immutable_field"}}
	var wg sync.WaitGroup
	var successCount int64
	var violationCount int64

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// 1. Attempt Atomic Acquisition
				err := chain.Acquire()

				if err == core.ErrConcurrencyViolation {
					atomic.AddInt64(&violationCount, 1)
					continue
				}

				if err != nil {
					continue
				}

				// 2. Perform "Work" - Reading the Name field.
				// If Reset() is called concurrently with this read,
				// the race detector will catch it.
				val := fmt.Sprintf("val-%d", workerID)
				failErr := chain.Fail(nil, val, "error")

				if fe, ok := failErr.(*core.FieldError); ok {
					// Logic Check: If we held the lock, the Name should
					// NOT have been cleared yet by a Reset (unless logic is leaky).
					if fe.Path == "" && chain.Name != "" {
						t.Errorf("Logic Breach: Path was empty while Name was %s", chain.Name)
					}
				}

				atomic.AddInt64(&successCount, 1)

				// 3. Atomic Release
				chain.Release()
			}
		}(i)
	}

	wg.Wait()

	t.Logf("State Race Results: Successes: %d, Violations: %d", successCount, violationCount)

	// Property Check: At least one must succeed, but exactly one must hold the lock at any time.
	// The Race Detector (-race) will handle the memory visibility verification.
	if successCount == 0 {
		t.Error("Zero successful acquisitions under contention - possible deadlock in Acquire")
	}
}

func TestBaseChain_ResetSafety(t *testing.T) {
	t.Parallel()

	chain := &MockChain{chains.BaseChain{Name: "pool_item"}}

	// Ensure Reset is idempotent and resets state
	for i := 0; i < 10; i++ {
		_ = chain.Acquire()
		chain.Reset()

		if chain.Name != "" {
			t.Errorf("Iteration %d: Name not cleared by Reset", i)
		}

		if err := chain.Acquire(); err != nil {
			t.Errorf("Iteration %d: State not reset to 0 by Reset, got: %v", i, err)
		}
		chain.Release()
	}
}

func TestBaseChain_AnonymousPathFormatting(t *testing.T) {
	t.Parallel()
	ctx := core.NewContext(core.Config{})
	_ = ctx.Push("Items")
	_ = ctx.PushIndex(5)

	// Chain with empty Name (typical for slice elements).
	chain := &MockChain{chains.BaseChain{Name: ""}}

	err := chain.Fail(ctx, "val", "reason")
	fe := err.(*core.FieldError)

	// Logic Check: Ensure no trailing or double dots.
	expected := "Items[5]"
	if fe.Path != expected {
		t.Errorf("Path integrity failure. Got %q, want %q", fe.Path, expected)
	}
}

func TestBaseChain_LockResiliencyAfterPanic(t *testing.T) {
	t.Parallel()
	chain := &MockChain{chains.BaseChain{Name: "resilient_field"}}

	// Simulate a panic during validation.
	func() {
		defer func() { _ = recover() }()
		_ = chain.Acquire()
		panic("simulated failure")
	}()

	// Reset should be able to recover the "bricked" chain for pooling.
	chain.Reset()

	if err := chain.Acquire(); err != nil {
		t.Errorf("Regression: Chain remained locked after panic and Reset: %v", err)
	}
}
