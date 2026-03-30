// SPDX-License-Identifier: MIT

package skrub_test

import (
	"reflect"
	"testing"

	skrub "github.com/Emin-ACIKGOZ/go-skrub"
	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// TestBoundChainConstructors verifies that the public builder functions
// correctly instantiate the underlying bound Chain types and set the Name property.
func TestBoundChainConstructors(t *testing.T) {
	t.Parallel()

	var name string
	var tags []string

	tests := []struct {
		name         string
		builderFunc  func() core.Rule
		expectedType reflect.Type
		expectedName string
	}{
		{
			name: "String Builder",
			builderFunc: func() core.Rule {
				return skrub.String(&name, "userName")
			},
			expectedType: reflect.TypeOf((*chains.StringChain)(nil)),
			expectedName: "userName",
		},
		{
			name: "Slice Builder",
			builderFunc: func() core.Rule {
				return skrub.Slice(&tags, "userTags")
			},
			expectedType: reflect.TypeOf((*chains.SliceChain)(nil)),
			expectedName: "userTags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chain := tt.builderFunc()

			// Check concrete type.
			if reflect.TypeOf(chain) != tt.expectedType {
				t.Fatalf("Builder returned unexpected type. Got %v, want %v", reflect.TypeOf(chain), tt.expectedType)
			}

			// Access embedded Name field via Elem() and FieldByName.
			val := reflect.ValueOf(chain).Elem()
			nameField := val.FieldByName("Name")
			if !nameField.IsValid() || nameField.String() != tt.expectedName {
				t.Errorf("Name mismatch. Got %q, want %q", nameField.String(), tt.expectedName)
			}
		})
	}
}

// TestUnboundTemplateConstructors ensures the Def functions return the correct struct types.
func TestUnboundTemplateConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		builderFunc  func() core.Template
		expectedType reflect.Type
	}{
		{
			name:         "DefString Builder",
			builderFunc:  func() core.Template { return skrub.DefString() },
			expectedType: reflect.TypeOf((*defs.StringDef)(nil)),
		},
		{
			name:         "DefInt Builder",
			builderFunc:  func() core.Template { return skrub.DefInt() },
			expectedType: reflect.TypeOf((*defs.IntDef)(nil)),
		},
		{
			name:         "DefSlice Builder",
			builderFunc:  func() core.Template { return skrub.DefSlice() },
			expectedType: reflect.TypeOf((*defs.SliceDef)(nil)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			template := tt.builderFunc()

			actualType := reflect.TypeOf(template)
			if actualType != tt.expectedType {
				t.Errorf("Template builder returned unexpected type. Got %v, want %v", actualType, tt.expectedType)
			}
		})
	}
}

// TestDefMatrix checks the recursive structural integrity of the matrix builder
// across various dimension depths, including edge cases.
func TestDefMatrix(t *testing.T) {
	t.Parallel()

	innerTemplate := skrub.DefInt().Min(0)

	tests := []struct {
		name           string
		dimensions     int
		expectedLayers int
	}{
		{
			name:           "0-Dimension Matrix (Edge Case)",
			dimensions:     0,
			expectedLayers: 0,
		},
		{
			name:           "1-Dimension Matrix (SliceDef)",
			dimensions:     1,
			expectedLayers: 1,
		},
		{
			name:           "3-Dimension Matrix (Nested SliceDef)",
			dimensions:     3,
			expectedLayers: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := skrub.DefMatrix(tt.dimensions, innerTemplate)

			// Logic Validation: For 0 dimensions, it should return the inner template directly.
			if tt.dimensions < 1 {
				if result != innerTemplate {
					t.Errorf("DefMatrix(%d) failed to return inner template. Got %v", tt.dimensions, result)
				}
				return
			}

			// Traverse the nested structure to verify depth and layer types.
			current := result
			for layers := 0; layers < tt.dimensions; layers++ {
				sliceDef, ok := current.(*defs.SliceDef)
				if !ok {
					t.Fatalf("Layer %d is not a *defs.SliceDef. Got %v", layers+1, reflect.TypeOf(current))
				}
				current = sliceDef.GetElementTemplate()
			}

			// Final check of the innermost core.
			if current != innerTemplate {
				t.Errorf("Innermost template mismatch. Got %v, want %v", current, innerTemplate)
			}
		})
	}
}
