// SPDX-License-Identifier: MIT

package defs_test

import (
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

func TestSliceDefBinding(t *testing.T) {
	t.Parallel()

	// Inner definition: string must be an email.
	innerTemplate := defs.NewStringDef().Email()

	// Outer definition: slice must have length between 1 and 3, and all elements must be emails.
	sliceTemplate := defs.NewSliceDef().
		MinLen(1).
		MaxLen(3).
		Elements(innerTemplate)

	t.Run("SuccessCase", func(t *testing.T) {
		t.Parallel()
		data := []string{"a@b.c", "d@e.f"}

		rule := sliceTemplate.Bind(&data, "emails")
		err := rule.Validate(nil)

		if err != nil {
			t.Errorf("Expected success, got: %v", err)
		}
	})

	t.Run("LengthFailure", func(t *testing.T) {
		t.Parallel()
		data := []string{"a", "b", "c", "d"} // Length 4 (Fails MaxLen 3)

		rule := sliceTemplate.Bind(&data, "emails")
		err := rule.Validate(nil)

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != "slice length exceeds maximum" {
			t.Fatalf("Expected MaxLen failure, got: %v", err)
		}
	})

	t.Run("ElementFailure", func(t *testing.T) {
		t.Parallel()
		data := []string{"valid@mail.com", "not-an-email", "ok@mail.com"} // Length 3, element 1 fails

		rule := sliceTemplate.Bind(&data, "emails")
		err := rule.Validate(nil)

		// Expected error path: emails[1]
		expectedPath := "emails[1]"
		if fe, ok := err.(*core.FieldError); !ok || fe.Path != expectedPath || fe.Reason != "invalid email format" {
			t.Fatalf("Expected failure at '%s' with 'invalid email format', got: %v", expectedPath, err)
		}
	})
}
