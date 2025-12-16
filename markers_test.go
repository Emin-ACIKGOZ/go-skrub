// SPDX-License-Identifier: MIT

package skrub_test

import (
	"reflect"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// verifyImplements ensures that the concrete type implements the specified interface type
// using Go reflection. It confirms expected structural contracts at runtime.
func verifyImplements(t *testing.T, concreteType reflect.Type, interfaceType reflect.Type) {
	t.Helper()

	if !concreteType.Implements(interfaceType) {
		t.Errorf("Type %s does not implement interface %s. Check requires the pointer type if methods use pointer receivers.", concreteType.String(), interfaceType.Name())
	}
}

// TestBoundChainImplementations verifies that all concrete Rule chain types
// correctly implement the core.Rule and core.Resetter interfaces.
func TestBoundChainImplementations(t *testing.T) {
	t.Parallel()

	ruleInterface := reflect.TypeOf((*core.Rule)(nil)).Elem()
	resetterInterface := reflect.TypeOf((*core.Resetter)(nil)).Elem()

	tests := []struct {
		name string
		typ  any // The concrete pointer type to check
	}{
		{
			name: "StringChain",
			typ:  &chains.StringChain{},
		},
		{
			name: "IntChain",
			typ:  &chains.IntChain{},
		},
		{
			name: "SliceChain",
			typ:  &chains.SliceChain{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Get the pointer type which implements the interfaces.
			concreteType := reflect.TypeOf(tt.typ)

			verifyImplements(t, concreteType, ruleInterface)
			verifyImplements(t, concreteType, resetterInterface)
		})
	}
}

// TestUnboundTemplateImplementations verifies that all concrete Definition types
// correctly implement the core.Template interface.
func TestUnboundTemplateImplementations(t *testing.T) {
	t.Parallel()

	templateInterface := reflect.TypeOf((*core.Template)(nil)).Elem()

	tests := []struct {
		name string
		typ  any // The concrete pointer type to check
	}{
		{
			name: "StringDef",
			typ:  &defs.StringDef{},
		},
		{
			name: "IntDef",
			typ:  &defs.IntDef{},
		},
		{
			name: "SliceDef",
			typ:  &defs.SliceDef{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Get the pointer type which implements the interface.
			concreteType := reflect.TypeOf(tt.typ)

			verifyImplements(t, concreteType, templateInterface)
		})
	}
}
