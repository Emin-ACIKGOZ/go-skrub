// SPDX-License-Identifier: MIT

package skrub

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"

	"github.com/Emin-ACIKGOZ/go-skrub/defs"
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
		return nil, fmt.Errorf("skrub: GetChain called with nil target for field %q: %w", name, core.ErrMisuse)
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

// ---------------------------------------------------------------------------
// Tag-based validation
// ---------------------------------------------------------------------------

// TagTemplateFunc is a factory that creates a core.Template from a tag parameter.
// The param is the value after '=' in the tag (e.g., "5" from "min=5").
type TagTemplateFunc func(param string) (core.Template, error)

// tagValidatorRegistry maps tag names to Template factories for tag-based struct validation.
// Populated by RegisterTagValidator and the init() function below.
var tagValidatorRegistry sync.Map // map[string]TagTemplateFunc

// RegisterTagValidator registers a tag name that can be used in struct field
// "validate" tags. The factory receives the tag parameter (empty string if no
// parameter was provided) and must return a core.Template that performs the
// validation.
//
// Example:
//
//	skrub.RegisterTagValidator("hexcolor", func(param string) (core.Template, error) {
//	    return skrub.DefString().Pattern(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`), nil
//	})
func RegisterTagValidator(tagName string, factory TagTemplateFunc) {
	tagValidatorRegistry.Store(tagName, factory)
}

// init registers the default built-in tag validators.
//
// These match go-validator's baked-in validators where applicable.
//
//nolint:cyclop // init functions are naturally large; each registration is a simple factory
func init() {
	// String validators
	RegisterTagValidator("required", func(_ string) (core.Template, error) {
		return DefString().NotEmpty(), nil
	})
	RegisterTagValidator("min", func(param string) (core.Template, error) {
		n, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid min parameter %q: %w", param, err)
		}
		return DefString().Min(n), nil
	})
	RegisterTagValidator("max", func(param string) (core.Template, error) {
		n, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid max parameter %q: %w", param, err)
		}
		return DefString().Max(n), nil
	})
	RegisterTagValidator("len", func(param string) (core.Template, error) {
		n, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid len parameter %q: %w", param, err)
		}
		return DefString().Min(n).Max(n), nil
	})
	RegisterTagValidator("email", func(_ string) (core.Template, error) {
		return DefString().Email(), nil
	})
	RegisterTagValidator("url", func(_ string) (core.Template, error) {
		return DefString().URL(), nil
	})
	RegisterTagValidator("uuid", func(_ string) (core.Template, error) {
		return DefString().UUID(), nil
	})
	RegisterTagValidator("ip", func(_ string) (core.Template, error) {
		return DefString().IP(), nil
	})
	RegisterTagValidator("ipv4", func(_ string) (core.Template, error) {
		return DefString().IPv4(), nil
	})
	RegisterTagValidator("ipv6", func(_ string) (core.Template, error) {
		return DefString().IPv6(), nil
	})

	// Integer-compatible validators (also work on strings for length)
	// When the field is an int, these return IntDef templates.
	RegisterTagValidator("gte", func(param string) (core.Template, error) {
		n, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid gte parameter %q: %w", param, err)
		}
		return DefInt().Min(n), nil
	})
	RegisterTagValidator("gt", func(param string) (core.Template, error) {
		n, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid gt parameter %q: %w", param, err)
		}
		return DefInt().Min(n + 1), nil
	})
	RegisterTagValidator("lte", func(param string) (core.Template, error) {
		n, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid lte parameter %q: %w", param, err)
		}
		return DefInt().Max(n), nil
	})
	RegisterTagValidator("lt", func(param string) (core.Template, error) {
		n, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid lt parameter %q: %w", param, err)
		}
		return DefInt().Max(n - 1), nil
	})
	RegisterTagValidator("eq", func(param string) (core.Template, error) {
		n, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid eq parameter %q: %w", param, err)
		}
		return DefInt().Min(n).Max(n), nil
	})
	RegisterTagValidator("ne", func(param string) (core.Template, error) {
		_, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid ne parameter %q: %w", param, err)
		}
		return DefInt().NotZero(), nil
	})

	// Slice validators
	RegisterTagValidator("min_len", func(param string) (core.Template, error) {
		n, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid min_len parameter %q: %w", param, err)
		}
		return DefSlice().MinLen(n), nil
	})
	RegisterTagValidator("max_len", func(param string) (core.Template, error) {
		n, err := strconv.Atoi(param)
		if err != nil {
			return nil, fmt.Errorf("invalid max_len parameter %q: %w", param, err)
		}
		return DefSlice().MaxLen(n), nil
	})
}

// ResolveTag returns a core.Template for the given tag name and parameter.
// It checks the tagValidatorRegistry and returns an error if the tag is unknown.
func ResolveTag(tagName, param string) (core.Template, error) {
	factoryIface, ok := tagValidatorRegistry.Load(tagName)
	if !ok {
		return nil, fmt.Errorf("unknown validation tag %q", tagName)
	}
	factory := factoryIface.(TagTemplateFunc)
	return factory(param)
}

// ---------------------------------------------------------------------------
// Alias support
// ---------------------------------------------------------------------------

// RegisterAlias registers a tag alias that is expanded before tag parsing.
// When a tag is encountered, it is replaced with the expansion.
// This matches go-validator's RegisterAlias behavior.
//
// Example:
//
//	skrub.RegisterAlias("iscolor", "hexcolor|rgb|rgba|hsl|hsla|cmyk")
//	someField `validate:"iscolor"`  // expands to hexcolor|rgb|rgba|hsl|hsla|cmyk
func RegisterAlias(alias, expansion string) {
	defs.RegisterAlias(alias, expansion)
}
