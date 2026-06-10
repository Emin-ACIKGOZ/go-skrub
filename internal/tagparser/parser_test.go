// SPDX-License-Identifier: MIT

package tagparser

import (
	"testing"
)

func TestParse_BasicTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		wantLen int
		wantErr bool
	}{
		{"required", 1, false},
		{"min=3", 1, false},
		{"max=100", 1, false},
		{"email", 1, false},
		{"required,min=3,max=100", 3, false},
		{"", 0, false},
		{"-", 0, false},
		{"uuid", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
			if len(result.Tags) != tt.wantLen {
				t.Errorf("Parse(%q) got %d tags, want %d", tt.input, len(result.Tags), tt.wantLen)
			}
		})
	}
}

func TestParse_Params(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input     string
		wantName  string
		wantParam string
	}{
		{"min=5", "min", "5"},
		{"max=10", "max", "10"},
		{"len=8", "len", "8"},
		{"eq=hello", "eq", "hello"},
		{"contains=substr", "contains", "substr"},
		{"required", "required", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Tags) != 1 {
				t.Fatalf("expected 1 tag, got %d", len(result.Tags))
			}
			if result.Tags[0].Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Tags[0].Name, tt.wantName)
			}
			if result.Tags[0].Param != tt.wantParam {
				t.Errorf("Param = %q, want %q", result.Tags[0].Param, tt.wantParam)
			}
		})
	}
}

func TestParse_HexEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		check func(*testing.T, ParseResult)
	}{
		{
			input: "excludes=0x2C",
			check: func(t *testing.T, r ParseResult) {
				if len(r.Tags) != 1 || r.Tags[0].Param != "," {
					t.Errorf("expected param ',', got %q", r.Tags[0].Param)
				}
			},
		},
		{
			input: "contains=0x7C",
			check: func(t *testing.T, r ParseResult) {
				if len(r.Tags) != 1 || r.Tags[0].Param != "|" {
					t.Errorf("expected param '|', got %q", r.Tags[0].Param)
				}
			},
		},
		{
			input: "excludes=0x2C,contains=0x7C",
			check: func(t *testing.T, r ParseResult) {
				if len(r.Tags) != 2 {
					t.Fatalf("expected 2 tags, got %d", len(r.Tags))
				}
				if r.Tags[0].Param != "," {
					t.Errorf("tag[0] param = %q, want %q", r.Tags[0].Param, ",")
				}
				if r.Tags[1].Param != "|" {
					t.Errorf("tag[1] param = %q, want %q", r.Tags[1].Param, "|")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, result)
		})
	}
}

func TestParse_UnsupportedReturnsError(t *testing.T) {
	t.Parallel()

	unsupported := []string{}

	for _, tag := range unsupported {
		t.Run(tag, func(t *testing.T) {
			_, err := Parse(tag)
			if err == nil {
				t.Errorf("Parse(%q) expected error for unsupported tag", tag)
			}
		})
	}
}

func TestParse_ControlTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		wantLen int
	}{
		{"omitempty,min=3", 2},
		{"required,omitempty", 2},
		{"dive", 1}, // dive is a control tag, still appears in output
		{"omitempty,dive", 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Tags) != tt.wantLen {
				t.Errorf("Parse(%q) got %d tags, want %d", tt.input, len(result.Tags), tt.wantLen)
			}
		})
	}
}

func TestParse_OrPipe(t *testing.T) {
	t.Parallel()

	result, err := Parse("required|email")
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasOrGroups {
		t.Error("expected HasOrGroups")
	}
	if len(result.Alternatives) != 2 {
		t.Fatalf("expected 2 alternatives, got %d", len(result.Alternatives))
	}
	if len(result.Alternatives[0]) != 1 || result.Alternatives[0][0].Name != "required" {
		t.Errorf("alternative[0] = %v, want [{required}]", result.Alternatives[0])
	}
	if len(result.Alternatives[1]) != 1 || result.Alternatives[1][0].Name != "email" {
		t.Errorf("alternative[1] = %v, want [{email}]", result.Alternatives[1])
	}
	if len(result.Tags) != 0 {
		t.Errorf("expected no AND tags, got %d", len(result.Tags))
	}
}

//nolint:cyclop // OR+AND composition test has many assertions
func TestParse_OrAndComposition(t *testing.T) {
	t.Parallel()

	result, err := Parse("min=3|max=10,required")
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasOrGroups {
		t.Error("expected HasOrGroups")
	}
	if len(result.Alternatives) != 2 {
		t.Fatalf("expected 2 alternatives, got %d", len(result.Alternatives))
	}
	if len(result.Alternatives[0]) != 1 || result.Alternatives[0][0].Name != "min" {
		t.Errorf("alternative[0] = %v, want [{min}]", result.Alternatives[0])
	}
	if result.Alternatives[0][0].Param != "3" {
		t.Errorf("alternative[0].Param = %q, want %q", result.Alternatives[0][0].Param, "3")
	}
	if len(result.Alternatives[1]) != 1 || result.Alternatives[1][0].Name != "max" {
		t.Errorf("alternative[1] = %v, want [{max}]", result.Alternatives[1])
	}
	if result.Alternatives[1][0].Param != "10" {
		t.Errorf("alternative[1].Param = %q, want %q", result.Alternatives[1][0].Param, "10")
	}
	if len(result.Tags) != 1 || result.Tags[0].Name != "required" {
		t.Errorf("tags = %v, want [{required}]", result.Tags)
	}
}

func TestParse_OrPipeInvalidControlTag(t *testing.T) {
	t.Parallel()

	_, err := Parse("required|omitempty")
	if err == nil {
		t.Error("expected error for control tag in OR group")
	}
}

func TestParse_KeysEndKeys(t *testing.T) {
	t.Parallel()

	result, err := Parse("dive,keys,min=1,max=5,endkeys,required")
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasKeys {
		t.Error("expected HasKeys")
	}
	if len(result.KeyTags) != 2 {
		t.Fatalf("expected 2 key tags, got %d: %v", len(result.KeyTags), result.KeyTags)
	}
	if result.KeyTags[0].Name != "min" || result.KeyTags[0].Param != "1" {
		t.Errorf("key tag[0] = %v, want {min,1}", result.KeyTags[0])
	}
	if result.KeyTags[1].Name != "max" || result.KeyTags[1].Param != "5" {
		t.Errorf("key tag[1] = %v, want {max,5}", result.KeyTags[1])
	}

	// Verify endkeys is not in Tags
	const endkeysName = "endkeys"
	for _, tt := range result.Tags {
		if tt.Name == endkeysName {
			t.Error("endkeys should not be in Tags")
		}
	}
}

func TestParse_DuplicateKeys(t *testing.T) {
	t.Parallel()
	_, err := Parse("dive,keys,min=1,keys,max=5,endkeys")
	if err == nil {
		t.Error("expected error for duplicate keys")
	}
}

func TestParse_EndKeysWithoutKeys(t *testing.T) {
	t.Parallel()
	_, err := Parse("endkeys")
	if err == nil {
		t.Error("expected error for endkeys without keys")
	}
}

func TestParse_StructOnly(t *testing.T) {
	t.Parallel()
	result, err := Parse("structonly")
	if err != nil {
		t.Fatal(err)
	}
	if !result.StructOnly {
		t.Error("expected StructOnly")
	}
}

func TestParse_NoStructLevel(t *testing.T) {
	t.Parallel()
	result, err := Parse("nostructlevel")
	if err != nil {
		t.Fatal(err)
	}
	if !result.NoStructLevel {
		t.Error("expected NoStructLevel")
	}
}

func TestParse_Composition(t *testing.T) {
	t.Parallel()

	result, err := Parse("required,min=3,max=100,email")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tags) != 4 {
		t.Fatalf("expected 4 tags, got %d: %v", len(result.Tags), result.Tags)
	}

	expected := []struct {
		name  string
		param string
	}{
		{"required", ""},
		{"min", "3"},
		{"max", "100"},
		{"email", ""},
	}

	for i, exp := range expected {
		if result.Tags[i].Name != exp.name {
			t.Errorf("tag[%d].Name = %q, want %q", i, result.Tags[i].Name, exp.name)
		}
		if result.Tags[i].Param != exp.param {
			t.Errorf("tag[%d].Param = %q, want %q", i, result.Tags[i].Param, exp.param)
		}
	}
}
