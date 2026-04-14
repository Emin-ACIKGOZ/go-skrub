// SPDX-License-Identifier: MIT

package chains_test

import (
	"regexp"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
)

// TestStringValidator_URLNotEmpty tests URL and NotEmpty combination.
func TestStringValidator_URLNotEmpty(t *testing.T) {
	t.Parallel()

	// Should pass: valid URL
	url := "https://example.com"
	chain := chains.NewStringChain(&url, "url").URL().NotEmpty()
	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected valid URL with NotEmpty to pass, got: %v", err)
	}

	// Should fail on NotEmpty: empty string
	emptyUrl := ""
	chain = chains.NewStringChain(&emptyUrl, "url").URL().NotEmpty()
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected empty string to fail NotEmpty check")
	}

	// Should fail on URL: invalid scheme
	badUrl := "not-a-url"
	chain = chains.NewStringChain(&badUrl, "url").URL().NotEmpty()
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected invalid URL to fail URL validation")
	}
}

// TestStringValidator_URLMin tests URL and Min combination.
func TestStringValidator_URLMin(t *testing.T) {
	t.Parallel()

	// Should pass: URL with enough characters
	url := "https://example.com"
	chain := chains.NewStringChain(&url, "url").URL().Min(5)
	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected URL meeting Min constraint to pass, got: %v", err)
	}

	// Should fail on Min: URL too short
	shortUrl := "http://a.co" // 11 chars, min 20
	chain = chains.NewStringChain(&shortUrl, "url").URL().Min(20)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected short URL to fail Min constraint")
	}
}

// TestStringValidator_IPNotEmpty tests IP and NotEmpty combination.
func TestStringValidator_IPNotEmpty(t *testing.T) {
	t.Parallel()

	// Should pass: valid IPv4
	ip := "192.168.1.1"
	chain := chains.NewStringChain(&ip, "ip").IP().NotEmpty()
	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected valid IP with NotEmpty to pass, got: %v", err)
	}

	// Should pass: valid IPv6
	ipv6 := "::1"
	chain = chains.NewStringChain(&ipv6, "ip").IPv6().NotEmpty()
	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected valid IPv6 with NotEmpty to pass, got: %v", err)
	}

	// Should fail on NotEmpty: empty string
	emptyIp := ""
	chain = chains.NewStringChain(&emptyIp, "ip").IP().NotEmpty()
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected empty string to fail NotEmpty check")
	}
}

// TestStringValidator_IPv4IPv6_Order tests IPv4 and IPv6 in different orders.
func TestStringValidator_IPv4IPv6_Order(t *testing.T) {
	t.Parallel()

	// Valid IPv4 should pass both IPv4 and general IP validators
	ip := "192.168.1.1"
	chain1 := chains.NewStringChain(&ip, "ip").IPv4().NotEmpty()
	chain2 := chains.NewStringChain(&ip, "ip").IP().NotEmpty()

	if err := chain1.Validate(nil); err != nil {
		t.Errorf("IPv4 should pass IPv4 validator, got: %v", err)
	}
	if err := chain2.Validate(nil); err != nil {
		t.Errorf("IPv4 should pass general IP validator, got: %v", err)
	}

	// IPv6 should fail IPv4 validator
	ipv6 := "::1"
	chain3 := chains.NewStringChain(&ipv6, "ip").IPv4()
	if err := chain3.Validate(nil); err == nil {
		t.Error("IPv6 should fail IPv4 validator")
	}
}

// TestIntValidator_MatchStringMin tests MatchString and Min combination.
func TestIntValidator_MatchStringMin(t *testing.T) {
	t.Parallel()

	// Should pass: matches pattern and passes Min
	val := 100
	pattern := regexp.MustCompile(`^[1-9]\d{2}$`)
	chain := chains.NewIntChain(&val, "id").MatchString(pattern).Min(50)
	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected value matching pattern and Min to pass, got: %v", err)
	}

	// Should fail on Min: matches pattern but fails Min
	val = 5
	chain = chains.NewIntChain(&val, "id").MatchString(pattern).Min(100)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected Min failure even though pattern might match")
	}

	// Should fail on MatchString: passes Min but fails pattern
	val = 5
	pattern = regexp.MustCompile(`^[1-9]\d{2}$`) // 100-999 only
	chain = chains.NewIntChain(&val, "id").MatchString(pattern).Min(1)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected pattern failure even though Min passes")
	}
}

// TestIntValidator_NotZeroMin tests NotZero and Min combination.
func TestIntValidator_NotZeroMin(t *testing.T) {
	t.Parallel()

	// Should pass: non-zero and meets Min
	val := 10
	chain := chains.NewIntChain(&val, "count").NotZero().Min(5)
	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected non-zero value meeting Min to pass, got: %v", err)
	}

	// Should fail on NotZero: zero fails
	val = 0
	chain = chains.NewIntChain(&val, "count").NotZero().Min(0)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected NotZero to reject zero")
	}

	// Should fail on Min: non-zero but below minimum
	val = 2
	chain = chains.NewIntChain(&val, "count").NotZero().Min(10)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected Min to reject value below minimum")
	}
}

// TestIntValidator_NotZeroMax tests NotZero and Max combination.
func TestIntValidator_NotZeroMax(t *testing.T) {
	t.Parallel()

	// Should pass: non-zero and within Max
	val := 50
	chain := chains.NewIntChain(&val, "percentage").NotZero().Max(100)
	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected non-zero value within Max to pass, got: %v", err)
	}

	// Should fail on Max: non-zero but exceeds maximum
	val = 150
	chain = chains.NewIntChain(&val, "percentage").NotZero().Max(100)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected Max to reject value exceeding maximum")
	}
}

// TestSliceValidator_NotEmptyMinLen tests NotEmpty and MinLen combination.
func TestSliceValidator_NotEmptyMinLen(t *testing.T) {
	t.Parallel()

	// Should pass: non-empty and meets MinLen
	items := []string{"a", "b", "c"}
	chain := chains.NewSliceChain(&items, "items").NotEmpty().MinLen(2)
	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected non-empty slice meeting MinLen to pass, got: %v", err)
	}

	// Should fail on NotEmpty: empty slice
	emptyItems := []string{}
	chain = chains.NewSliceChain(&emptyItems, "items").NotEmpty().MinLen(1)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected NotEmpty to reject empty slice")
	}

	// Should fail on MinLen: single item doesn't meet minimum
	singleItem := []string{"a"}
	chain = chains.NewSliceChain(&singleItem, "items").NotEmpty().MinLen(3)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected MinLen to reject slice below minimum")
	}
}

// TestSliceValidator_NotEmptyMaxLen tests NotEmpty and MaxLen combination.
func TestSliceValidator_NotEmptyMaxLen(t *testing.T) {
	t.Parallel()

	// Should pass: non-empty and within MaxLen
	items := []string{"a", "b"}
	chain := chains.NewSliceChain(&items, "items").NotEmpty().MaxLen(5)
	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected non-empty slice within MaxLen to pass, got: %v", err)
	}

	// Should fail on MaxLen: too many items
	manyItems := []string{"a", "b", "c", "d", "e"}
	chain = chains.NewSliceChain(&manyItems, "items").NotEmpty().MaxLen(3)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected MaxLen to reject slice exceeding maximum")
	}
}

// TestSliceValidator_MinLenMaxLen tests MinLen and MaxLen combination.
func TestSliceValidator_MinLenMaxLen(t *testing.T) {
	t.Parallel()

	// Should pass: within bounds
	items := []string{"a", "b", "c"}
	chain := chains.NewSliceChain(&items, "items").MinLen(2).MaxLen(5)
	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected slice within bounds to pass, got: %v", err)
	}

	// Should fail on MinLen: too few items
	smallItems := []string{"a"}
	chain = chains.NewSliceChain(&smallItems, "items").MinLen(2).MaxLen(5)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected MinLen to reject slice with too few items")
	}

	// Should fail on MaxLen: too many items
	largeItems := []string{"a", "b", "c", "d", "e", "f"}
	chain = chains.NewSliceChain(&largeItems, "items").MinLen(2).MaxLen(5)
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected MaxLen to reject slice with too many items")
	}
}

// TestStringValidator_AllCombinations tests all combinations of string validators.
func TestStringValidator_AllCombinations(t *testing.T) {
	t.Parallel()

	// Valid email: should pass all string validators
	email := "user@example.com"
	chain := chains.NewStringChain(&email, "email").
		NotEmpty().
		Email().
		Min(5).
		Max(255)

	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected email passing all validators to pass, got: %v", err)
	}

	// Invalid email (too short): should fail
	shortEmail := "a@b"
	chain = chains.NewStringChain(&shortEmail, "email").
		NotEmpty().
		Email().
		Min(10)

	if err := chain.Validate(nil); err == nil {
		t.Error("Expected short email to fail Min constraint")
	}
}

// TestIntValidator_AllCombinations tests all combinations of int validators.
func TestIntValidator_AllCombinations(t *testing.T) {
	t.Parallel()

	// Valid value: should pass all int validators
	val := 42
	pattern := regexp.MustCompile(`^[0-9]+$`)
	chain := chains.NewIntChain(&val, "count").
		NotZero().
		Min(10).
		Max(100).
		MatchString(pattern)

	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected valid value passing all validators to pass, got: %v", err)
	}

	// Invalid value (zero): should fail NotZero
	val = 0
	chain = chains.NewIntChain(&val, "count").NotZero()
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected zero to fail NotZero validator")
	}

	// Invalid value (too large): should fail Max
	val = 150
	chain = chains.NewIntChain(&val, "count").
		NotZero().
		Min(10).
		Max(100)

	if err := chain.Validate(nil); err == nil {
		t.Error("Expected value exceeding Max to fail")
	}
}

// TestSliceValidator_AllCombinations tests all combinations of slice validators.
func TestSliceValidator_AllCombinations(t *testing.T) {
	t.Parallel()

	// Valid slice: should pass all validators
	items := []string{"item1", "item2"}
	chain := chains.NewSliceChain(&items, "items").
		NotEmpty().
		MinLen(1).
		MaxLen(10)

	if err := chain.Validate(nil); err != nil {
		t.Errorf("Expected valid slice passing all validators to pass, got: %v", err)
	}

	// Empty slice: should fail NotEmpty
	emptyItems := []string{}
	chain = chains.NewSliceChain(&emptyItems, "items").NotEmpty()
	if err := chain.Validate(nil); err == nil {
		t.Error("Expected empty slice to fail NotEmpty validator")
	}

	// Slice too large: should fail MaxLen
	largeItems := make([]string, 15)
	for i := 0; i < 15; i++ {
		largeItems[i] = "item"
	}
	chain = chains.NewSliceChain(&largeItems, "items").
		NotEmpty().
		MaxLen(10)

	if err := chain.Validate(nil); err == nil {
		t.Error("Expected large slice to fail MaxLen validator")
	}
}
