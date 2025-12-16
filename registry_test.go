// SPDX-License-Identifier: MIT

package skrub_test

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	skrub "github.com/Emin-ACIKGOZ/go-skrub"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// --- Test Structures and Mocks ---

// MockTarget is a custom type used for registration testing.
type MockTarget struct {
	ID int
}

// MockChain implements core.Rule.
type MockChain struct {
	Target *MockTarget
	Name   string
}

// Validate implements core.Rule, satisfying the interface but performing no logic.
func (c *MockChain) Validate(_ *core.Context) error {
	return nil
}

// MockFactory returns a new MockChain for *MockTarget.
func MockFactory(ptr *MockTarget, name string) core.Rule {
	if ptr == nil {
		panic("Factory received nil pointer!")
	}
	return &MockChain{
		Target: ptr,
		Name:   name,
	}
}

// --- Additional Mocks for Concurrency Tests ---

type AnotherTarget struct{}

type AnotherChain struct{}

// Validate implements core.Rule, satisfying the interface but performing no logic.
func (c *AnotherChain) Validate(_ *core.Context) error {
	return nil
}

// AnotherFactory returns a new AnotherChain for *AnotherTarget.
func AnotherFactory(_ *AnotherTarget, _ string) core.Rule {
	return &AnotherChain{}
}

// --- Setup/Teardown ---

func setup(_ *testing.T) {
	// Ensure a clean state before each test block.
	skrub.ClearRegistry()
}

// --- Test Cases ---

func TestRegisterAndGetChain(t *testing.T) {
	setup(t)

	// 1. Setup: Register the MockFactory for *MockTarget
	skrub.Register(MockFactory)

	var target MockTarget
	targetPtr := &target
	const name = "testField"

	t.Run("Success_Retrieval", func(t *testing.T) {
		t.Parallel()

		chain, err := skrub.GetChain(targetPtr, name)
		if err != nil {
			t.Fatalf("GetChain failed unexpectedly: %v", err)
		}

		mockChain, ok := chain.(*MockChain)
		if !ok {
			t.Fatalf("Returned chain was not *MockChain. Got %T", chain)
		}

		// Check if the factory correctly bound the target and name
		if mockChain.Target != targetPtr {
			t.Errorf("Chain target mismatch. Got %v, want %v", mockChain.Target, targetPtr)
		}
		if mockChain.Name != name {
			t.Errorf("Chain name mismatch. Got %s, want %s", mockChain.Name, name)
		}
	})

	t.Run("Failure_UnregisteredType", func(t *testing.T) {
		t.Parallel()

		// Use a different, unregistered type
		var unregisteredTarget int
		unregisteredPtr := &unregisteredTarget

		_, err := skrub.GetChain(unregisteredPtr, "otherField")

		expectedType := reflect.TypeOf(unregisteredPtr)
		expectedErr := fmt.Sprintf("skrub: no registered chain found for type %v", expectedType)

		if err == nil {
			t.Fatal("Expected an error for unregistered type, got nil")
		}
		if err.Error() != expectedErr {
			t.Errorf("Error message mismatch. Got '%s', want '%s'", err.Error(), expectedErr)
		}
	})

	t.Run("Failure_NilTarget", func(t *testing.T) {
		t.Parallel()

		_, err := skrub.GetChain(nil, "nilField")

		if !errors.Is(err, core.ErrMisuse) {
			t.Errorf("Expected core.ErrMisuse for nil target, got: %v", err)
		}
	})
}

func TestClearRegistry(t *testing.T) {

	// 1. Register a type
	skrub.Register(MockFactory)
	var target MockTarget
	targetPtr := &target

	// 2. Verify registration worked
	_, err := skrub.GetChain(targetPtr, "temp")
	if err != nil {
		t.Fatalf("Setup failed, expected chain, got error: %v", err)
	}

	// 3. Clear the registry
	skrub.ClearRegistry()

	// 4. Verify the type is no longer registered
	_, err = skrub.GetChain(targetPtr, "temp")
	expectedErr := fmt.Sprintf("skrub: no registered chain found for type %v", reflect.TypeOf(targetPtr))

	if err == nil {
		t.Fatal("Expected error after ClearRegistry, got nil")
	}
	if err.Error() != expectedErr {
		t.Errorf("Error message mismatch after clearing. Got '%s', want '%s'", err.Error(), expectedErr)
	}
}

func TestConcurrentSafety(t *testing.T) {
	setup(t)

	const numRoutines = 100
	var wg sync.WaitGroup

	// Concurrently Register and Clear
	wg.Add(numRoutines * 2)
	for i := 0; i < numRoutines; i++ {
		go func() {
			defer wg.Done()
			skrub.Register(MockFactory)
		}()

		// Use the correct closure pattern for i to capture the index safely
		go func(i int) {
			defer wg.Done()
			// Clear the registry occasionally to test concurrent access patterns
			if i%10 == 0 {
				skrub.ClearRegistry()
			}
		}(i)
	}
	wg.Wait()

	// Concurrently Read (GetChain)
	targetPtr := &MockTarget{}
	wg.Add(numRoutines)

	for i := 0; i < numRoutines; i++ {
		// Use a local variable to capture the index for logging

		go func() {
			defer wg.Done()
			// Must check the error return value to satisfy errcheck linter.
			_, err := skrub.GetChain(targetPtr, "concurrentField")

			// Check the error: if it's NOT a "not found" error, it indicates a serious failure
			// in the concurrent mechanism that should fail the test immediately.
			if err != nil && !isNotFoundError(err) {
				t.Errorf("Goroutine %d: Unexpected non-registry error during concurrent read: %v", i, err)
			}
		}()
	}
	wg.Wait()

	// Final cleanup
	skrub.ClearRegistry()
}

// isNotFoundError checks if the error indicates a missing chain in the registry.
func isNotFoundError(err error) bool {
	// Check if the error string starts with the package prefix, indicating a registry lookup failure.
	return err != nil && (len(err.Error()) > 6 && err.Error()[:6] == "skrub:")
}
