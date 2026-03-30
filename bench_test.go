// SPDX-License-Identifier: MIT

package skrub_test

import (
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub"
	"github.com/Emin-ACIKGOZ/go-skrub/defs"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
	"github.com/go-playground/validator/v10"
)

type User struct {
	Name  string   `validate:"required,min=3"`
	Age   int      `validate:"required,min=18"`
	Email string   `validate:"required,email"`
	Tags  []string `validate:"required,min=1,dive,max=10"`
}

var (
	testUser = User{
		Name:  "Emin",
		Age:   25,
		Email: "emin@example.com",
		Tags:  []string{"go", "dev", "skrub"},
	}

	matrixData = make([][][]int, 10)
	v          = validator.New()

	// -------------------------------------------------------------------------
	// Skrub Pre-compiled Rules (Optimized Engine)
	// -------------------------------------------------------------------------
	nameRule  = defs.NewStringDef().Min(3).Bind(&testUser.Name, "Name")
	ageRule   = defs.NewIntDef().Min(18).Bind(&testUser.Age, "Age")
	emailRule = skrub.String(&testUser.Email, "Email").Email()
	tagsRule  = defs.NewSliceDef().MinLen(1).Elements(
		defs.NewStringDef().Max(10),
	).Bind(&testUser.Tags, "Tags")

	matrixTemplate = skrub.DefMatrix(3, skrub.DefInt().Min(0))
	boundMatrix    = matrixTemplate.Bind(&matrixData, "matrix")
)

func init() {
	// Initialize 3D Matrix: 10x10x10 = 1,000 integers
	for i := 0; i < 10; i++ {
		matrixData[i] = make([][]int, 10)
		for j := 0; j < 10; j++ {
			matrixData[i][j] = make([]int, 10)
			for k := 0; k < 10; k++ {
				matrixData[i][j][k] = k
			}
		}
	}
}

// --- TIER 1: SMALL STRUCT ---

func BenchmarkGoValidator_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := v.Struct(testUser); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSkrub_Small_Optimized(b *testing.B) {
	b.ReportAllocs()
	cfg := core.Config{}
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := nameRule.Validate(ctx); err != nil {
			b.Fatal(err)
		}
		if err := ageRule.Validate(ctx); err != nil {
			b.Fatal(err)
		}
		if err := emailRule.Validate(ctx); err != nil {
			b.Fatal(err)
		}
		if err := tagsRule.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// --- TIER 2: DEEP MATRIX (1,000 Elements) ---

func BenchmarkGoValidator_DeepMatrix(b *testing.B) {
	// To validate [][][]int, we need a wrapper struct with nested dive tags.
	type MatrixWrapper struct {
		Data [][][]int `validate:"required,dive,dive,dive,min=0"`
	}
	wrapper := MatrixWrapper{Data: matrixData}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := v.Struct(wrapper); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSkrub_DeepMatrix_Optimized(b *testing.B) {
	b.ReportAllocs()
	cfg := core.Config{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := boundMatrix.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
