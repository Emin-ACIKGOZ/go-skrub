// SPDX-License-Identifier: MIT

package skrub

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

var (
	// registryMu protects the registry map for concurrent access.
	registryMu sync.RWMutex

	// registry stores the factory functions for custom types, keyed by reflect.Type of the pointer (*T).
	registry = make(map[reflect.Type]func(target any, name string) core.Rule)
)

// Register adds a custom chain factory for a specific type T.
// This allows users to extend go-skrub with their own domain-specific validation chains.
// The factory function receives a typed pointer (*T), ensuring strong type safety at registration.
//
// Usage:
//
//	skrub.Register(func(u *User, name string) core.Rule {
//	    return NewUserChain(u, name)
//	})
func Register[T any](factory func(ptr *T, name string) core.Rule) {
	registryMu.Lock()
	defer registryMu.Unlock()

	// Use a nil pointer to get the reflect.Type for registration key, as validation targets are passed by reference.
	var ptr *T
	typ := reflect.TypeOf(ptr)

	// Wrap the typed factory in a generic closure for storage and safe runtime assertion.
	registry[typ] = func(target any, name string) core.Rule {
		typedPtr := target.(*T)
		return factory(typedPtr, name)
	}
}

// GetChain retrieves a registered validation chain factory for the given target type.
// It uses reflection to look up the factory associated with the target's pointer type.
//
// Returns:
//   - The bound Rule if a factory is found.
//   - nil and a formatted error if no factory is registered for this type.
//   - nil and core.ErrMisuse if the target is nil.
func GetChain(target any, name string) (core.Rule, error) {
	if target == nil {
		return nil, core.ErrMisuse
	}

	typ := reflect.TypeOf(target)

	registryMu.RLock()
	factory, exists := registry[typ]
	registryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("skrub: no registered chain found for type %v", typ)
	}

	return factory(target, name), nil
}

// ClearRegistry removes all currently registered factories.
// This function is primarily used for testing cleanup.
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	for k := range registry {
		delete(registry, k)
	}
}
