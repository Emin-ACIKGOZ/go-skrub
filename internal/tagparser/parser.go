// SPDX-License-Identifier: MIT

// Package tagparser parses validation tags from struct fields.
// It matches go-validator's tag semantics:
//   - Tags are split by ',' (AND composition)
//   - Within each comma segment, tags are split by '|' (OR, deferred to v1.1+)
//   - Key-value pairs are split by '=': tag_name=param
//   - Hex-encoded commas (0x2C) and pipes (0x7C) are decoded in params
//   - Unknown tags and unsupported patterns return errors (no panics)
package tagparser

import (
	"errors"
	"fmt"
	"strings"
)

// Tag represents a single parsed validation directive.
type Tag struct {
	Name  string // validator name (e.g., "min", "email", "required")
	Param string // parameter (e.g., "5" from "min=5")
}

// ParseResult holds the parsed validation tags for a single field.
type ParseResult struct {
	Tags          []Tag   // AND-composed tags (comma-separated in source)
	Alternatives  [][]Tag // OR groups: each inner slice is one alternative
	HasOrGroups   bool
	HasDive       bool  // true if dive tag was found
	HasKeys       bool  // true if keys/endkeys section found
	KeyTags       []Tag // tags between keys and endkeys
	StructOnly    bool  // true if structonly tag found
	NoStructLevel bool  // true if nostructlevel tag found
}

// Parse parses a validate tag string into a ParseResult.
// It matches go-validator's parsing order: split by ',' first, then
// within each segment by '|'.
//
// Examples:
//
//	"required,min=3,max=100" → Tags: [{required,""}, {min,"3"}, {max,"100"}]
//	"required,omitempty,min=3" → Tags: [{required,""}, {omitempty,""}, {min,"3"}]
//	"required|email" → Alternatives: [[{required,""}], [{email,""}]]
//	"min=3|max=10,required" → Alternatives: [[{min,"3"}], [{max,"10"}]], Tags: [{required,""}]
//	"excludes=0x2C" → Tags: [{excludes,","}]
//
//nolint:cyclop // Complexity stems from OR, control tags, and error paths
func Parse(tag string) (ParseResult, error) {
	if tag == "" || tag == "-" {
		return ParseResult{}, nil
	}

	var result ParseResult

	// Step 1: Split by ',' (AND composition) — matches go-validator cache.go:178
	parts := strings.Split(tag, ",")

	// Track whether we're inside a keys/endkeys section
	inKeys := false
	var keyTags []Tag

	for i := 0; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}

		// Handle keys section start
		if part == "keys" {
			if inKeys {
				return ParseResult{}, errors.New("duplicate keys tag")
			}
			result.HasKeys = true
			inKeys = true
			result.Tags = append(result.Tags, Tag{Name: "dive"})
			continue
		}
		if part == "endkeys" {
			if !inKeys {
				return ParseResult{}, errors.New("endkeys without matching keys")
			}
			result.KeyTags = keyTags
			inKeys = false
			continue
		}
		if inKeys {
			t, err := parseSingleTag(part, tag)
			if err != nil {
				return ParseResult{}, err
			}
			keyTags = append(keyTags, t)
			continue
		}

		// structonly and nostructlevel are structural markers for nested structs
		if part == "structonly" {
			result.StructOnly = true
			result.Tags = append(result.Tags, Tag{Name: "structonly"})
			continue
		}
		if part == "nostructlevel" {
			result.NoStructLevel = true
			result.Tags = append(result.Tags, Tag{Name: "nostructlevel"})
			continue
		}

		// Control flow tags (dive, omitempty, omitnil, omitzero, isdefault)
		if isControlTag(part) {
			result.Tags = append(result.Tags, Tag{Name: part})
			continue
		}

		// Step 2: Split by '|' (OR semantics)
		orParts := strings.Split(part, "|")
		if len(orParts) > 1 {
			result.HasOrGroups = true
			for _, orPart := range orParts {
				orPart = strings.TrimSpace(orPart)
				if orPart == "" {
					continue
				}
				t, err := parseSingleTag(orPart, tag)
				if err != nil {
					return ParseResult{}, err
				}
				if isControlTag(t.Name) {
					return ParseResult{}, fmt.Errorf("control tag %q cannot be used within OR groups", t.Name)
				}
				result.Alternatives = append(result.Alternatives, []Tag{t})
			}
		} else {
			t, err := parseSingleTag(part, tag)
			if err != nil {
				return ParseResult{}, err
			}
			result.Tags = append(result.Tags, t)
		}
	}

	return result, nil
}

// parseSingleTag parses a single tag="param" string.
func parseSingleTag(part, origTag string) (Tag, error) {
	const splitLimit = 2
	vals := strings.SplitN(part, "=", splitLimit)
	name := vals[0]
	if name == "" {
		return Tag{}, fmt.Errorf("empty tag name in %q", origTag)
	}
	t := Tag{Name: name}
	if len(vals) > 1 {
		t.Param = strings.ReplaceAll(vals[1], "0x2C", ",")
		t.Param = strings.ReplaceAll(t.Param, "0x7C", "|")
	}
	return t, nil
}

// isControlTag returns true for structural tags that affect control flow.
func isControlTag(tag string) bool {
	switch tag {
	case "dive", "omitempty", "omitnil", "omitzero", "isdefault",
		"keys", "endkeys", "structonly", "nostructlevel":
		return true
	default:
		return false
	}
}
