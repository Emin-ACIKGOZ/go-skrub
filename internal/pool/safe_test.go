// SPDX-License-Identifier: MIT

package pool_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Emin-ACIKGOZ/go-skrub/internal/pool"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// MockItem implements core.Resetter.
type MockItem struct {
	ID      int
	IsDirty bool
}

// Reset cleans up the item before reuse.
func (m *MockItem) Reset() {
	m.IsDirty = false
}

// Validate is implemented to satisfy interface requirements if needed elsewhere.
func (*MockItem) Validate(_ *core.Context) error { return nil }

// makeScopedFactory returns a factory function and a pointer to a counter.
// This ensures every test gets its own isolated state.
func makeScopedFactory() (func() any, *int32) {
	var counter int32
	factory := func() any {
		id := atomic.AddInt32(&counter, 1)
		return &MockItem{ID: int(id)}
	}
	return factory, &counter
}

// TestSafePoolInitialization verifies that pools are correctly sized and pre-populated.
func TestSafePoolInitialization(t *testing.T) {
	t.Parallel()

	t.Run("DefaultCapacity", func(t *testing.T) {
		t.Parallel()
		factory, counter := makeScopedFactory()

		cfg := pool.Config{Factory: factory, Capacity: 0, NonBlocking: true}
		p := pool.NewSafePool(cfg)

		if val := atomic.LoadInt32(counter); val != 10 {
			t.Errorf("Expected factory to run 10 times, ran %d", val)
		}

		for i := 0; i < 10; i++ {
			if _, err := p.Get(); err != nil {
				t.Fatalf("Failed to retrieve item %d from pool: %v", i, err)
			}
		}

		if _, err := p.Get(); err != core.ErrPoolExhausted {
			t.Errorf("Expected pool to be exhausted, got: %v", err)
		}
	})

	t.Run("CustomCapacity", func(t *testing.T) {
		t.Parallel()
		factory, counter := makeScopedFactory()

		cfg := pool.Config{Factory: factory, Capacity: 5, NonBlocking: true}
		p := pool.NewSafePool(cfg)

		if val := atomic.LoadInt32(counter); val != 5 {
			t.Errorf("Expected factory to run 5 times, ran %d", val)
		}

		for i := 0; i < 5; i++ {
			if _, err := p.Get(); err != nil {
				t.Fatalf("Failed to retrieve item %d from pool: %v", i, err)
			}
		}

		if _, err := p.Get(); err != core.ErrPoolExhausted {
			t.Errorf("Expected pool to be exhausted, got: %v", err)
		}
	})
}

// TestSafePoolBehavior verifies non-blocking exhaustion, the Resetter lifecycle,
// and the full pool item dropping mechanism.
func TestSafePoolBehavior(t *testing.T) {
	t.Parallel()
	t.Run("NonBlockingExhaustion", testNonBlockingExhaustion)
	t.Run("ResetterCallOnPut", testResetterCallOnPut)
	t.Run("FullPoolDrop", testFullPoolDrop)
}

func testNonBlockingExhaustion(t *testing.T) {
	t.Parallel()
	factory, _ := makeScopedFactory()
	p := pool.NewSafePool(pool.Config{
		Factory:     factory,
		Capacity:    2,
		NonBlocking: true,
	})

	if _, err := p.Get(); err != nil {
		t.Fatalf("Setup failed: could not drain item 1: %v", err)
	}
	if _, err := p.Get(); err != nil {
		t.Fatalf("Setup failed: could not drain item 2: %v", err)
	}

	if _, err := p.Get(); err != core.ErrPoolExhausted {
		t.Errorf("Expected ErrPoolExhausted, got: %v", err)
	}
}

func testResetterCallOnPut(t *testing.T) {
	t.Parallel()
	factory, _ := makeScopedFactory()
	p := pool.NewSafePool(pool.Config{
		Factory:     factory,
		Capacity:    1,
		NonBlocking: true,
	})

	itemRaw, err := p.Get()
	if err != nil {
		t.Fatalf("Setup failed: could not get item: %v", err)
	}
	mockItem := itemRaw.(*MockItem)
	mockItem.IsDirty = true

	p.Put(mockItem)

	recycledItem, err := p.Get()
	if err != nil {
		t.Fatalf("Get after Put failed: %v", err)
	}

	if recycledItem.(*MockItem).IsDirty {
		t.Errorf("Resetter failed: Item was still dirty after Put")
	}
}

func testFullPoolDrop(t *testing.T) {
	t.Parallel()
	factory, _ := makeScopedFactory()
	p := pool.NewSafePool(pool.Config{
		Factory:     factory,
		Capacity:    2,
		NonBlocking: true,
	})

	item1, err := p.Get()
	if err != nil {
		t.Fatalf("Setup failed: could not get item 1: %v", err)
	}
	item2, err := p.Get()
	if err != nil {
		t.Fatalf("Setup failed: could not get item 2: %v", err)
	}

	p.Put(item2)
	p.Put(item1)

	unmanagedItem := &MockItem{ID: 999}
	p.Put(unmanagedItem)

	if _, err := p.Get(); err != nil {
		t.Fatalf("Verification failed: expected item 1, got error: %v", err)
	}
	if _, err := p.Get(); err != nil {
		t.Fatalf("Verification failed: expected item 2, got error: %v", err)
	}

	if _, err := p.Get(); err != core.ErrPoolExhausted {
		t.Errorf("Expected pool to be exhausted after drop, got: %v", err)
	}
}

// TestSafePoolBlocking verifies the behavior of Get when NonBlocking is false.
func TestSafePoolBlocking(t *testing.T) {
	t.Parallel()
	const capacity = 1
	factory, _ := makeScopedFactory()
	cfg := pool.Config{Factory: factory, Capacity: capacity, NonBlocking: false}
	p := pool.NewSafePool(cfg)

	item, err := p.Get()
	if err != nil {
		t.Fatalf("Initial Get failed: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var gotItem int32

	go func() {
		defer wg.Done()
		if _, err := p.Get(); err == nil {
			atomic.StoreInt32(&gotItem, 1)
		} else {
			t.Errorf("Blocking Get failed unexpectedly: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&gotItem) == 1 {
		t.Fatal("Blocking Get failed: Goroutine should not have completed yet.")
	}

	p.Put(item)
	wg.Wait()

	if atomic.LoadInt32(&gotItem) == 0 {
		t.Fatal("Blocking Put failed: Goroutine did not retrieve the item after Put.")
	}
}
