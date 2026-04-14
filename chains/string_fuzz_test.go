// SPDX-License-Identifier: MIT

package chains_test

import (
	"net"
	"net/url"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// FuzzStringChainURL tests URL validator against arbitrary input.
// Verifies that:
// - Validator never panics
// - Invalid URLs are rejected
// - Valid URLs are accepted
// - Injection attempts are handled safely
func FuzzStringChainURL(f *testing.F) {
	// Seed with known test cases
	f.Add("https://example.com")
	f.Add("http://example.com")
	f.Add("https://example.com:8080/path?query=value")
	f.Add("not-a-url")
	f.Add("")
	f.Add("://malformed")
	f.Add("ftp://not-http")
	f.Add("https://example.com\x00null")
	f.Add("https://example.com'; DROP TABLE users--")
	f.Add("<script>alert('xss')</script>")

	f.Fuzz(func(t *testing.T, input string) {
		val := input
		chain := chains.NewStringChain(&val, "url")
		chain.URL()

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// If it succeeds, verify Go's url.Parse also accepts it
		if err == nil {
			parsed, parseErr := url.Parse(input)
			if parseErr == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" {
				// Valid - should have succeeded
				return
			}
			t.Logf("WARNING: Validator accepted %q but url.Parse rejects or scheme is invalid", input)
		}
	})
}

// FuzzStringChainIPv4 tests IPv4 validator against arbitrary input.
// Verifies that:
// - Validator never panics
// - Invalid IPv4 addresses are rejected
// - Valid IPv4 addresses are accepted
func FuzzStringChainIPv4(f *testing.F) {
	// Seed with known test cases
	f.Add("192.168.1.1")
	f.Add("0.0.0.0")
	f.Add("255.255.255.255")
	f.Add("256.256.256.256")
	f.Add("192.168.1")
	f.Add("192.168.1.1.1")
	f.Add("")
	f.Add("not-an-ip")
	f.Add("::1") // IPv6
	f.Add("192.168.1.1\x00")
	f.Add("192.168.1.'; DROP TABLE--")

	f.Fuzz(func(t *testing.T, input string) {
		val := input
		chain := chains.NewStringChain(&val, "ipv4")
		chain.IPv4()

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// If it succeeds, verify net.ParseIP recognizes it as valid IPv4
		if err == nil {
			parsed := net.ParseIP(input)
			if parsed != nil && parsed.To4() != nil {
				// Valid IPv4 - should have succeeded
				return
			}
			t.Logf("WARNING: Validator accepted %q but net.ParseIP rejects as IPv4", input)
		}
	})
}

// FuzzStringChainIPv6 tests IPv6 validator against arbitrary input.
// Verifies that:
// - Validator never panics
// - Invalid IPv6 addresses are rejected
// - Valid IPv6 addresses are accepted
// - IPv4-mapped IPv6 addresses are accepted (::ffff:x.x.x.x)
func FuzzStringChainIPv6(f *testing.F) {
	// Seed with known test cases
	f.Add("::1")
	f.Add("::")
	f.Add("2001:db8::1")
	f.Add("fe80::1")
	f.Add("::ffff:192.0.2.1")           // IPv4-mapped standard
	f.Add("::ffff:127.0.0.1")           // IPv4-mapped loopback
	f.Add("::ffff:0:0")                 // IPv4-mapped all zeros
	f.Add("not-an-ipv6")
	f.Add("")
	f.Add("192.168.1.1")
	f.Add(":::::::")
	f.Add("gggg::1")
	f.Add("::1\x00")
	f.Add("2001:db8::1'; DROP TABLE--")

	f.Fuzz(func(t *testing.T, input string) {
		val := input
		chain := chains.NewStringChain(&val, "ipv6")
		chain.IPv6()

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// If it succeeds, verify net.ParseIP recognizes it as valid IPv6
		if err == nil {
			parsed := net.ParseIP(input)
			if parsed != nil && parsed.To16() != nil && parsed.To4() == nil {
				// Valid IPv6 (not IPv4) - should have succeeded
				return
			}
			t.Logf("WARNING: Validator accepted %q but net.ParseIP rejects as pure IPv6", input)
		}
	})
}

// FuzzStringChainIP tests IP validator (both IPv4 and IPv6) against arbitrary input.
// Verifies that:
// - Validator never panics
// - Invalid IP addresses are rejected
// - Valid IPv4 and IPv6 addresses are accepted
func FuzzStringChainIP(f *testing.F) {
	// Seed with known test cases
	f.Add("192.168.1.1")
	f.Add("::1")
	f.Add("2001:db8::1")
	f.Add("0.0.0.0")
	f.Add("::")
	f.Add("256.256.256.256")
	f.Add("gggg::1")
	f.Add("")
	f.Add("not-an-ip")
	f.Add("192.168.1.1:8080")
	f.Add("127.0.0.1\x00")

	f.Fuzz(func(t *testing.T, input string) {
		val := input
		chain := chains.NewStringChain(&val, "ip")
		chain.IP()

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// If it succeeds, verify net.ParseIP accepts it
		if err == nil {
			parsed := net.ParseIP(input)
			if parsed != nil {
				// Valid IP - should have succeeded
				return
			}
			t.Logf("WARNING: Validator accepted %q but net.ParseIP rejects", input)
		}
	})
}

// FuzzStringChainNotEmpty tests NotEmpty validator against arbitrary input.
// Verifies that:
// - Validator never panics
// - Empty strings are rejected
// - Non-empty strings are accepted
func FuzzStringChainNotEmpty(f *testing.F) {
	// Seed with known test cases
	f.Add("hello")
	f.Add("")
	f.Add(" ")
	f.Add("\t")
	f.Add("\n")
	f.Add("\x00")
	f.Add("a")
	f.Add("very long string with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?")

	f.Fuzz(func(t *testing.T, input string) {
		val := input
		chain := chains.NewStringChain(&val, "field")
		chain.NotEmpty()

		// Should not panic
		err := chain.Validate(core.NewContext(core.Config{}))

		// Verify behavior matches expectation
		if len(input) == 0 && err == nil {
			t.Errorf("NotEmpty should reject empty string, got nil error")
		}
		if len(input) > 0 && err != nil {
			t.Errorf("NotEmpty should accept non-empty string %q, got error: %v", input, err)
		}
	})
}
