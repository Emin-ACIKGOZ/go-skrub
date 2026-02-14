// SPDX-License-Identifier: MIT

package pool_test

import (
	"math/rand"
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

		p := pool.NewSafePool(pool.Config{Factory: factory, Capacity: 0, NonBlocking: true})

		if atomic.LoadInt32(counter) != 10 {
			t.Errorf("Expected default capacity 10, got %d", atomic.LoadInt32(counter))
		}
		
		if _, err := p.Get(); err != nil {
			t.Error(err)
		}
	})

	t.Run("NilFactorySafety", func(t *testing.T) {
		t.Parallel()
		// Use NonBlocking: true to ensure the test doesn't hang on an empty pool.
		p := pool.NewSafePool(pool.Config{Factory: nil, Capacity: 5, NonBlocking: true})
		if _, err := p.Get(); err != core.ErrPoolExhausted {
			t.Errorf("Expected exhaustion error from empty pool, got %v", err)
		}
	})
}

// TestSafePoolBehavior verifies non-blocking exhaustion, the Resetter lifecycle,
// and the full pool item dropping mechanism.
func TestSafePoolBehavior(t *testing.T) {
	t.Parallel()

	t.Run("ResetterLifecycle", func(t *testing.T) {
		factory, _ := makeScopedFactory()
		p := pool.NewSafePool(pool.Config{Factory: factory, Capacity: 1, NonBlocking: true})

		item, _ := p.Get()
		m := item.(*MockItem)
		m.IsDirty = true
		p.Put(m)

		recycled, _ := p.Get()
		if recycled.(*MockItem).IsDirty {
			t.Error("Item was not reset before reuse")
		}
	})

	t.Run("OverflowDrop", func(t *testing.T) {
		factory, _ := makeScopedFactory()
		p := pool.NewSafePool(pool.Config{Factory: factory, Capacity: 1, NonBlocking: true})

		item1, _ := p.Get()
		p.Put(item1)
		p.Put(&MockItem{ID: 999}) // Should be dropped silently

		_, _ = p.Get()
		if _, err := p.Get(); err != core.ErrPoolExhausted {
			t.Error("Pool should have dropped the overflow item")
		}
	})
}

// --- Stress & Chaos Tests (The "Production" Guard) ---

func TestSafePool_Chaos(t *testing.T) {
	const (
		capacity    = 100
		workers     = 2000
		iterations  = 100
		maxJitterMs = 5
	)

	factory, _ := makeScopedFactory()
	p := pool.NewSafePool(pool.Config{
		Factory:     factory,
		Capacity:    capacity,
		NonBlocking: false,
	})

	var (
		wg           sync.WaitGroup
		activeLeases int32
		totalGets    int64
		raceDetected int32
	)

	inUse := sync.Map{}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			//nolint:gosec // G404: math/rand is sufficient for jitter in a stress test
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

			for j := 0; j < iterations; j++ {
				item, err := p.Get()
				if err != nil {
					return
				}
				atomic.AddInt64(&totalGets, 1)

				current := atomic.AddInt32(&activeLeases, 1)
				if current > capacity {
					atomic.StoreInt32(&raceDetected, 1)
				}

				m := item.(*MockItem)
				if _, loaded := inUse.LoadOrStore(m.ID, true); loaded {
					atomic.StoreInt32(&raceDetected, 1)
				}

				if rng.Intn(100) < 30 {
					time.Sleep(time.Duration(rng.Intn(maxJitterMs)) * time.Millisecond)
				}

				inUse.Delete(m.ID)
				atomic.AddInt32(&activeLeases, -1)
				p.Put(m)
			}
		}(i)
	}

	wg.Wait()

	if raceDetected > 0 {
		t.Fatal("CRITICAL FAILURE: Pool invariant violated.")
	}
}

// TestSafePoolBlocking verifies the behavior of Get when NonBlocking is false.
func TestSafePoolBlocking_ThunderingHerd_Deep(t *testing.T) {
	t.Parallel()
	const capacity = 1
	const workers = 50
	factory, _ := makeScopedFactory()
	p := pool.NewSafePool(pool.Config{Factory: factory, Capacity: capacity, NonBlocking: false})

	mainItem, _ := p.Get()

	var wg sync.WaitGroup
	var successCount int32
	signal := make(chan struct{}, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			signal <- struct{}{}
			
			item, err := p.Get()
			if err == nil {
				atomic.AddInt32(&successCount, 1)
				p.Put(item)
			}
		}()
	}

	for i := 0; i < workers; i++ {
		<-signal
	}

	time.Sleep(10 * time.Millisecond)

	p.Put(mainItem)
	wg.Wait()

	if atomic.LoadInt32(&successCount) != int32(workers) {
		t.Errorf("Expected %d, got %d", workers, successCount)
	}
}
