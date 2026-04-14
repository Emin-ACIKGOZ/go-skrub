// SPDX-License-Identifier: MIT

package chains_test

import (
	"regexp"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// MockStringValuer implements core.Valuer and returns a string.
type MockStringValuer struct {
	Value string
}

func (m MockStringValuer) Unwrap() any {
	return m.Value
}

func TestStringChainValidation(t *testing.T) {
	t.Parallel()

	t.Run("MinLengthFailure", func(t *testing.T) {
		t.Parallel()
		val := "short"
		chain := chains.NewStringChain(&val, "input")
		err := chain.Min(10).Validate(nil)

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != core.ReasonMinLength {
			t.Errorf("Expected min length failure with reason %q, got: %v", core.ReasonMinLength, err)
		}
	})

	t.Run("EmailSuccess", func(t *testing.T) {
		t.Parallel()
		val := "test.user@example.com"
		chain := chains.NewStringChain(&val, "email")
		err := chain.Email().Validate(nil)

		if err != nil {
			t.Errorf("Expected email success, got: %v", err)
		}
	})

	t.Run("UUIDFailure", func(t *testing.T) {
		t.Parallel()
		val := "1234-abcd-5678" // Invalid UUID format
		chain := chains.NewStringChain(&val, "uuid")
		err := chain.UUID().Validate(nil)

		if fe, ok := err.(*core.FieldError); !ok || fe.Reason != core.ReasonInvalidUUID {
			t.Errorf("Expected UUID failure with reason %q, got: %v", core.ReasonInvalidUUID, err)
		}
	})

	t.Run("PatternCheck", func(t *testing.T) {
		t.Parallel()
		val := "AB-123"
		chain := chains.NewStringChain(&val, "code")

		re := regexp.MustCompile(`^[A-Z]{2}-\d{3}$`)
		err := chain.Pattern(re).Validate(nil)

		if err != nil {
			t.Errorf("Expected pattern success, got: %v", err)
		}
	})
}

func TestStringChainValuer(t *testing.T) {
	t.Parallel()

	t.Run("ValuerSuccess", func(t *testing.T) {
		t.Parallel()
		valuer := MockStringValuer{Value: "valid@mail.co"}
		chain := chains.NewStringChain(valuer, "email_valuer")
		err := chain.Email().Validate(nil)

		if err != nil {
			t.Errorf("Expected success with Valuer, got: %v", err)
		}
	})
}

func TestStringChainURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		url       string
		expectErr bool
		reason    string
	}{
		// Valid HTTP(S) URLs
		{"ValidHTTP", "http://example.com", false, ""},
		{"ValidHTTPS", "https://example.com", false, ""},
		{"HTTPSWithPath", "https://example.com/path", false, ""},
		{"HTTPSWithQuery", "https://example.com?key=value", false, ""},
		{"HTTPSWithPort", "https://example.com:8080", false, ""},
		{"HTTPSWithAuth", "https://user:pass@example.com", false, ""},
		{"HTTPSWithFragment", "https://example.com#section", false, ""},
		{"SimpleHTTP", "http://a.co", false, ""},
		{"LocalhostHTTP", "http://localhost", false, ""},
		{"LocalhostWithPort", "http://localhost:3000", false, ""},
		{"IPv4URL", "http://192.168.1.1", false, ""},
		{"IPv4WithPort", "http://192.168.1.1:8080", false, ""},
		{"ComplexPath", "https://api.example.com/v1/users/123", false, ""},
		{"MultipleQueryParams", "https://example.com?a=1&b=2&c=3", false, ""},
		{"HTTPWithPath", "http://example.com/some/path/here", false, ""},
		{"HTTPWithComplexQuery", "https://example.com/search?q=test&filter=active", false, ""},
		{"SubdomainURL", "https://api.v2.example.co.uk", false, ""},
		{"DeepPath", "https://example.com/a/b/c/d/e/f", false, ""},
		{"QueryWithSpecialChars", "https://example.com?encoded=%20space", false, ""},

		// Invalid URLs
		{"EmptyString", "", true, core.ReasonInvalidURL},
		{"NoScheme", "example.com", true, core.ReasonInvalidURL},
		{"WrongScheme", "ftp://example.com", true, core.ReasonInvalidURL},
		{"WrongSchemeHTTPS", "ftps://example.com", true, core.ReasonInvalidURL},
		{"NoHost", "http://", true, core.ReasonInvalidURL},
		{"OnlyScheme", "https://", true, core.ReasonInvalidURL},
		{"MalformedScheme", "ht!tp://example.com", true, core.ReasonInvalidURL},
		{"RelativePath", "/path/to/resource", true, core.ReasonInvalidURL},
		{"ProtocolRelative", "//example.com", true, core.ReasonInvalidURL},
	}

	for _, tt := range tests {
		tt := tt // Capture loop variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val := tt.url
			chain := chains.NewStringChain(&val, "url")
			err := chain.URL().Validate(nil)

			if (err != nil) != tt.expectErr {
				t.Errorf("URL %q: expected error=%v, got error=%v", tt.url, tt.expectErr, err)
				return
			}

			if tt.expectErr && err != nil {
				if fe, ok := err.(*core.FieldError); !ok || fe.Reason != tt.reason {
					t.Errorf("URL %q: expected reason %q, got %q", tt.url, tt.reason, fe.Reason)
				}
			}
		})
	}
}

func TestStringChainURLWithValuer(t *testing.T) {
	t.Parallel()

	t.Run("URLViaValuer_Success", func(t *testing.T) {
		t.Parallel()
		valuer := MockStringValuer{Value: "https://api.example.com/v1"}
		chain := chains.NewStringChain(valuer, "webhook_url")
		err := chain.URL().Validate(nil)

		if err != nil {
			t.Errorf("Expected URL validation via Valuer to succeed, got: %v", err)
		}
	})

	t.Run("URLViaValuer_Failure", func(t *testing.T) {
		t.Parallel()
		valuer := MockStringValuer{Value: "not-a-url"}
		chain := chains.NewStringChain(valuer, "webhook_url")
		err := chain.URL().Validate(nil)

		if err == nil {
			t.Error("Expected URL validation via Valuer to fail, got nil")
		}
	})
}

func TestStringChainIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ip        string
		expectErr bool
		reason    string
	}{
		// Valid IPv4 addresses
		{"IPv4Simple", "192.168.1.1", false, ""},
		{"IPv4Localhost", "127.0.0.1", false, ""},
		{"IPv4Broadcast", "255.255.255.255", false, ""},
		{"IPv4Zero", "0.0.0.0", false, ""},
		{"IPv4Public", "8.8.8.8", false, ""},
		{"IPv4EdgeCase1", "1.1.1.1", false, ""},
		{"IPv4EdgeCase2", "10.0.0.1", false, ""},

		// Valid IPv6 addresses
		{"IPv6Simple", "2001:db8::1", false, ""},
		{"IPv6Localhost", "::1", false, ""},
		{"IPv6Full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false, ""},
		{"IPv6Compressed", "2001:db8:85a3::8a2e:370:7334", false, ""},
		{"IPv6Zero", "::", false, ""},
		{"IPv6Multicast", "ff00::1", false, ""},
		{"IPv6LinkLocal", "fe80::1", false, ""},

		// Invalid addresses
		{"Empty", "", true, core.ReasonInvalidIP},
		{"NotAnIP", "not-an-ip", true, core.ReasonInvalidIP},
		{"OnlyNumbers", "1", true, core.ReasonInvalidIP},
		{"IPv4TooMany", "192.168.1.1.1", true, core.ReasonInvalidIP},
		{"IPv4TooFew", "192.168.1", true, core.ReasonInvalidIP},
		{"IPv4InvalidOctet", "256.1.1.1", true, core.ReasonInvalidIP},
		{"IPv4Negative", "-1.0.0.0", true, core.ReasonInvalidIP},
		{"IPv6Invalid", "gggg::1", true, core.ReasonInvalidIP},
		{"Mixed", "192.168.1.1:8080", true, core.ReasonInvalidIP}, // Has port
		{"URL", "http://192.168.1.1", true, core.ReasonInvalidIP},
		{"Domain", "example.com", true, core.ReasonInvalidIP},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val := tt.ip
			chain := chains.NewStringChain(&val, "ip")
			err := chain.IP().Validate(nil)

			if (err != nil) != tt.expectErr {
				t.Errorf("IP %q: expected error=%v, got error=%v", tt.ip, tt.expectErr, err)
				return
			}

			if tt.expectErr && err != nil {
				if fe, ok := err.(*core.FieldError); !ok || fe.Reason != tt.reason {
					t.Errorf("IP %q: expected reason %q, got %q", tt.ip, tt.reason, fe.Reason)
				}
			}
		})
	}
}

func TestStringChainIPv4(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ip        string
		expectErr bool
	}{
		{"Valid1", "192.168.1.1", false},
		{"Valid2", "127.0.0.1", false},
		{"Valid3", "0.0.0.0", false},
		{"IPv6Rejected", "2001:db8::1", true},
		{"IPv6Localhost", "::1", true},
		{"Empty", "", true},
		{"Invalid", "not-an-ip", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val := tt.ip
			chain := chains.NewStringChain(&val, "ipv4")
			err := chain.IPv4().Validate(nil)

			if (err != nil) != tt.expectErr {
				t.Errorf("IPv4 %q: expected error=%v, got error=%v", tt.ip, tt.expectErr, err)
			}
		})
	}
}

func TestStringChainIPv6(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ip        string
		expectErr bool
	}{
		{"Valid1", "2001:db8::1", false},
		{"Valid2", "::1", false},
		{"Valid3", "::", false},
		{"IPv4Rejected", "192.168.1.1", true},
		{"IPv4Localhost", "127.0.0.1", true},
		{"Empty", "", true},
		{"Invalid", "not-an-ip", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val := tt.ip
			chain := chains.NewStringChain(&val, "ipv6")
			err := chain.IPv6().Validate(nil)

			if (err != nil) != tt.expectErr {
				t.Errorf("IPv6 %q: expected error=%v, got error=%v", tt.ip, tt.expectErr, err)
			}
		})
	}
}

func TestStringChainNotEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		expectErr bool
		reason    string
	}{
		// Valid non-empty strings
		{"SingleChar", "a", false, ""},
		{"ShortString", "test", false, ""},
		{"LongString", "this is a longer string", false, ""},
		{"Spaces", "   ", false, ""}, // Spaces are not empty
		{"SingleSpace", " ", false, ""},
		{"Tab", "\t", false, ""},
		{"Newline", "\n", false, ""},
		{"Special", "!@#$%^&*()", false, ""},
		{"Emoji", "😀", false, ""},
		{"MultiUnicode", "你好世界", false, ""},
		{"Mixed", "abc 123 !@#", false, ""},

		// Invalid empty strings
		{"Empty", "", true, core.ReasonRequired},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val := tt.value
			chain := chains.NewStringChain(&val, "name")
			err := chain.NotEmpty().Validate(nil)

			if (err != nil) != tt.expectErr {
				t.Errorf("NotEmpty %q: expected error=%v, got error=%v", tt.value, tt.expectErr, err)
				return
			}

			if tt.expectErr && err != nil {
				if fe, ok := err.(*core.FieldError); !ok || fe.Reason != tt.reason {
					t.Errorf("NotEmpty %q: expected reason %q, got %q", tt.value, tt.reason, fe.Reason)
				}
			}
		})
	}
}

func TestStringChainNotEmptyWithOtherValidators(t *testing.T) {
	t.Parallel()

	t.Run("NotEmpty_Then_Min", func(t *testing.T) {
		t.Parallel()
		val := "a"
		chain := chains.NewStringChain(&val, "username").NotEmpty().Min(3)
		err := chain.Validate(nil)
		if err == nil {
			t.Error("Expected Min length failure, got nil")
		}
	})

	t.Run("NotEmpty_Then_Email", func(t *testing.T) {
		t.Parallel()
		val := "a"
		chain := chains.NewStringChain(&val, "email").NotEmpty().Email()
		err := chain.Validate(nil)
		if err == nil {
			t.Error("Expected Email failure, got nil")
		}
	})

	t.Run("NotEmpty_Success_With_Email", func(t *testing.T) {
		t.Parallel()
		val := "test@example.com"
		chain := chains.NewStringChain(&val, "email").NotEmpty().Email()
		err := chain.Validate(nil)
		if err != nil {
			t.Errorf("Expected success, got: %v", err)
		}
	})
}
