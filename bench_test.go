// SPDX-License-Identifier: MIT

package skrub_test

import (
	"regexp"
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

// --- TIER 3: INDIVIDUAL VALIDATORS ---

// Benchmark URL validator in isolation
func BenchmarkSkrub_Validator_URL(b *testing.B) {
	url := "https://api.example.com/v1/users"
	chain := skrub.String(&url, "webhook_url").URL()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark URL validator from Def (template reuse)
func BenchmarkSkrub_Validator_URLDef(b *testing.B) {
	urlTemplate := defs.NewStringDef().URL()
	url := "https://api.example.com/v1/users"
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain := urlTemplate.Bind(&url, "webhook_url")
		ctx := core.NewContext(cfg)
		if err := chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark URL validator with invalid input (worst case)
func BenchmarkSkrub_Validator_URL_InvalidInput(b *testing.B) {
	url := "not-a-url-at-all"
	chain := skrub.String(&url, "webhook_url").URL()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		_ = chain.Validate(ctx) // Expect error, ignore it
	}
}

// Benchmark Email for comparison with URL
func BenchmarkSkrub_Validator_Email(b *testing.B) {
	email := "user@example.com"
	chain := skrub.String(&email, "email").Email()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark IPv4 validator
func BenchmarkSkrub_Validator_IPv4(b *testing.B) {
	ip := "192.168.1.1"
	chain := skrub.String(&ip, "ipv4_address").IPv4()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark IPv6 validator
func BenchmarkSkrub_Validator_IPv6(b *testing.B) {
	ip := "2001:db8::1"
	chain := skrub.String(&ip, "ipv6_address").IPv6()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark IP validator (both IPv4 and IPv6)
func BenchmarkSkrub_Validator_IP(b *testing.B) {
	ip := "192.168.1.1"
	chain := skrub.String(&ip, "ip_address").IP()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark multiple validators combined (URL + Min length + Pattern)
func BenchmarkSkrub_Validators_Combined(b *testing.B) {
	url := "https://example.com/webhook"
	pattern := regexp.MustCompile(`^https://`)
	chain := skrub.String(&url, "webhook_url").URL().Min(15).Pattern(pattern)
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark all network validators combined (URL + IPv4 + IPv6)
func BenchmarkSkrub_Validators_Network_Combined(b *testing.B) {
	url := "https://192.168.1.1"
	urlChain := skrub.String(&url, "webhook_url").URL()
	ipv4 := "10.0.0.1"
	ipv4Chain := skrub.String(&ipv4, "ipv4").IPv4()
	ipv6 := "2001:db8::1"
	ipv6Chain := skrub.String(&ipv6, "ipv6").IPv6()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := urlChain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
		if err := ipv4Chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
		if err := ipv6Chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark NotEmpty validator for strings
func BenchmarkSkrub_Validator_NotEmpty_String(b *testing.B) {
	str := "non-empty"
	chain := skrub.String(&str, "name").NotEmpty()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark NotZero validator for integers
func BenchmarkSkrub_Validator_NotZero(b *testing.B) {
	num := 42
	template := defs.NewIntDef().NotZero()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain := template.Bind(&num, "count")
		ctx := core.NewContext(cfg)
		if err := chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark NotEmpty validator for slices
func BenchmarkSkrub_Validator_NotEmpty_Slice(b *testing.B) {
	items := []string{"item"}
	chain := skrub.Slice(&items, "items").NotEmpty()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := chain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark all required validators combined (NotEmpty String + NotZero Int + NotEmpty Slice)
func BenchmarkSkrub_Validators_Required_Combined(b *testing.B) {
	str := "value"
	strChain := skrub.String(&str, "name").NotEmpty()
	num := 1
	numTemplate := defs.NewIntDef().NotZero()
	items := []string{"item"}
	sliceChain := skrub.Slice(&items, "items").NotEmpty()
	cfg := core.Config{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := core.NewContext(cfg)
		if err := strChain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
		numChain := numTemplate.Bind(&num, "count")
		if err := numChain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
		if err := sliceChain.Validate(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
