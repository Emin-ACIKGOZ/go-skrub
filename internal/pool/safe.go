// SPDX-License-Identifier: MIT

// Package pool provides concurrency-safe resource pooling mechanisms.
// It uses buffered channels to implement bounded object pooling with capacity
// enforcement, offering both blocking and non-blocking retrieval methods.
package pool

import (
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// Config defines the configuration and behavior parameters for the SafePool.
type Config struct {
	// Factory creates new pool items upon initialization. This function is called
	// once for each item up to the defined Capacity when NewSafePool is executed.
	Factory func() any

	// Capacity sets the hard limit on the number of items in the pool,
	// controlling the size of the underlying buffered channel. It acts as a
	// semaphore to prevent unbounded memory growth. If Capacity is zero, it defaults to 10.
	Capacity int

	// NonBlocking controls the behavior when the pool is empty:
	// If true, Get returns core.ErrPoolExhausted immediately if no item is available.
	// If false, Get blocks indefinitely until an item is returned via Put.
	NonBlocking bool
}

// SafePool is a concurrency-safe, bounded resource pool.
// It manages the lifecycle of reusable objects, ensuring they are reset before reuse.
type SafePool struct {
	// items is a buffered channel that holds the available pool items.
	// Its buffer size equals the configured Capacity.
	items  chan any
	config Config
}

// NewSafePool initializes and returns a new bounded pool based on the provided configuration.
// If cfg.Capacity is zero or negative, it defaults to 10.
// If cfg.Factory is provided, NewSafePool pre-populates the pool up to the defined capacity.
func NewSafePool(cfg Config) *SafePool {
	if cfg.Capacity <= 0 {
		cfg.Capacity = 10
	}

	p := &SafePool{
		items:  make(chan any, cfg.Capacity),
		config: cfg,
	}

	if cfg.Factory != nil {
		// Pre-populate the channel with factory-created items.
		for i := 0; i < cfg.Capacity; i++ {
			item := cfg.Factory()
			if item == nil {
				panic("skrub: SafePool factory returned nil during pre-population")
			}
			p.items <- item
		}
	}

	return p
}

// Get retrieves an item from the pool.
//
// If p.config.NonBlocking is true and the pool is empty, Get returns nil and core.ErrPoolExhausted.
// If p.config.NonBlocking is false and the pool is empty, Get blocks until an item is available.
func (p *SafePool) Get() (any, error) {
	if p.config.NonBlocking {
		// Non-blocking mode: use select/default to return immediately if the channel is empty.
		select {
		case item := <-p.items:
			return item, nil
		default:
			return nil, core.ErrPoolExhausted
		}
	}

	// Blocking mode: Execute a direct, blocking read from the channel.
	item := <-p.items
	return item, nil
}

// Put returns an item to the pool.
//
// If the item implements core.Resetter, its Reset method is called before the
// item is returned to the pool. This ensures the item is in a clean state
// before any goroutine receives it from the channel.
// If the pool is full, the item is silently dropped to prevent blocking the caller.
func (p *SafePool) Put(item any) {
	if item == nil {
		return
	}

	// Reset the item BEFORE returning it to the channel. This avoids a race
	// where another goroutine's Get() receives the item before Reset() completes.
	if resetter, ok := item.(core.Resetter); ok {
		resetter.Reset()
	}

	// Use a non-blocking send to prevent blocking if the pool is already full
	// (channel capacity is reached). The item is dropped on default.
	select {
	case p.items <- item:
	default:
	}
}
