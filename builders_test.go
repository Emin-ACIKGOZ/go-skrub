// SPDX-License-Identifier: MIT

package skrub_test

import (
	"reflect"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"

	// Import the package under test (skrub) using its module path.
	skrub "github.com/Emin-ACIKGOZ/go-skrub"
)

// findNameField recursively searches for the 'Name' field in a reflect.Value,
// handling embedded structs.
func findNameField(v reflect.Value) (reflect.Value, bool) {
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	// 1. Check for 'Name' field directly.
	if nameField := v.FieldByName("Name"); nameField.IsValid() {
		return nameField, true
	}

	// 2. Check embedded/promoted fields.
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if field.Anonymous { // Check embedded/promoted fields.
			embeddedValue := v.Field(i)
			if nameField, found := findNameField(embeddedValue); found {
				return nameField, true
			}
		}
	}

	return reflect.Value{}, false
}

// TestBoundChainConstructors tests that the public builder functions (e.g., skrub.String)
// correctly instantiate the underlying bound Chain types and set the Name property.
func TestBoundChainConstructors(t *testing.T) {
	t.Parallel()

	// A simple variable to use as the target for the builders.
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
			expectedType: reflect.TypeOf(&chains.StringChain{}),
			expectedName: "userName",
		},
		{
			name: "Slice Builder",
			builderFunc: func() core.Rule {
				return skrub.Slice(&tags, "userTags")
			},
			expectedType: reflect.TypeOf(&chains.SliceChain{}),
			expectedName: "userTags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chain := tt.builderFunc()

			// 1. Check if the returned object is of the expected concrete chain type.
			actualType := reflect.TypeOf(chain)
			if actualType != tt.expectedType {
				t.Fatalf("Builder returned unexpected type. Got %v, want %v", actualType, tt.expectedType)
			}

			// 2. Use helper to find the 'Name' property in the embedded BaseChain.
			val := reflect.ValueOf(chain).Elem()
			nameField, found := findNameField(val)

			if !found {
				t.Fatalf("Failed to find 'Name' field in %T", chain)
			}
			if !nameField.IsValid() {
				t.Fatalf("Found 'Name' field, but it is invalid.")
			}

			// Access the value and assert it is a string before comparison.
			actualName, ok := nameField.Interface().(string)
			if !ok {
				t.Fatalf("Expected 'Name' field to be a string, got %T", nameField.Interface())
			}

			if actualName != tt.expectedName {
				t.Errorf("%T 'Name' field mismatch. Got %s, want %s", chain, actualName, tt.expectedName)
			}
		})
	}
}

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
			expectedType: reflect.TypeOf(&defs.StringDef{}),
		},
		{
			name:         "DefInt Builder",
			builderFunc:  func() core.Template { return skrub.DefInt() },
			expectedType: reflect.TypeOf(&defs.IntDef{}),
		},
		{
			name:         "DefSlice Builder",
			builderFunc:  func() core.Template { return skrub.DefSlice() },
			expectedType: reflect.TypeOf(&defs.SliceDef{}),
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

func TestDefMatrix(t *testing.T) {
	t.Parallel()

	innerTemplate := skrub.DefInt().Min(0)

	tests := []struct {
		name           string
		dimensions     int
		template       core.Template
		expectedLayers int
		wantError      bool
	}{
		{
			name:           "1-Dimension Matrix (SliceDef)",
			dimensions:     1,
			template:       innerTemplate,
			expectedLayers: 1,
			wantError:      false,
		},
		{
			name:           "3-Dimension Matrix (Nested SliceDef)",
			dimensions:     3,
			template:       innerTemplate,
			expectedLayers: 3,
			wantError:      false,
		},
		{
			name:           "0-Dimension Matrix (Edge Case)",
			dimensions:     0,
			template:       innerTemplate,
			expectedLayers: 0,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := skrub.DefMatrix(tt.dimensions, tt.template)

			if tt.dimensions < 1 {
				if result != innerTemplate {
					t.Errorf("DefMatrix(%d) for dimensions < 1 failed. Got %v, want %v (the inner template)", tt.dimensions, result, innerTemplate)
				}
				return
			}

			// Validate the depth of the recursive structure for dimensions >= 1.
			current := result
			layers := 0

			// Traverse the nested structure.
			for {
				// 1. Check if the current layer is a SliceDef.
				sliceDef, ok := current.(*defs.SliceDef)
				if !ok {
					t.Fatalf("Layer %d is not a *defs.SliceDef. Got %v", layers+1, reflect.TypeOf(current))
				}
				layers++

				// 2. Use the public accessor method.
				elementTemplate := sliceDef.GetElementTemplate()

				if elementTemplate == nil {
					t.Fatalf("Failed to retrieve element template using accessor at layer %d.", layers)
				}

				current = elementTemplate

				// 3. Check for termination condition: reached the innermost template.
				if layers == tt.dimensions {
					// The next layer must be the original innerTemplate.
					if current != innerTemplate {
						t.Errorf("Final layer template mismatch. Got %v, want %v", current, innerTemplate)
					}
					break
				}
			}

			// Final check of layer count.
			if layers != tt.expectedLayers {
				t.Errorf("DefMatrix(%d) created wrong number of layers. Got %d, want %d", tt.dimensions, layers, tt.expectedLayers)
			}
		})
	}
}
