// SPDX-License-Identifier: MIT

package defs

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/Emin-ACIKGOZ/go-skrub/chains"
	"github.com/Emin-ACIKGOZ/go-skrub/internal/tagparser"
	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// Global registry for tag aliases (e.g., iscolor → hexcolor|rgb|rgba|hsl|hsla|cmyk).
var tagAliases sync.Map // map[string]string

// RegisterAlias registers a tag alias that is expanded before tag parsing.
// This matches go-validator's RegisterAlias behavior.
func RegisterAlias(alias, expansion string) {
	tagAliases.Store(alias, expansion)
}

// expandAliases expands any alias references in a tag string by recursively
// replacing alias names with their expansions. Matches go-validator's alias
// behavior where aliases are expanded before tag parsing.
const expandSplitLimit = 2

func expandAliases(tag string) string {
	parts := strings.Split(tag, ",")
	expanded := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Check if the entire part is an alias (no param)
		if expansion, ok := tagAliases.Load(part); ok {
			expanded = append(expanded, expandAliases(expansion.(string)))
			continue
		}
		// Check for aliases in OR branches
		orParts := strings.Split(part, "|")
		for i, orPart := range orParts {
			orPart = strings.TrimSpace(orPart)
			name := strings.SplitN(orPart, "=", expandSplitLimit)[0]
			if expansion, ok := tagAliases.Load(name); ok {
				// Extract param if present
				vals := strings.SplitN(orPart, "=", expandSplitLimit)
				expandedOr := expandAliases(expansion.(string))
				if len(vals) > 1 {
					// Append param to the last tag in the expansion
					expandedOr = expandedOr + "=" + vals[1]
				}
				orParts[i] = expandedOr
			}
		}
		expanded = append(expanded, strings.Join(orParts, "|"))
	}
	return strings.Join(expanded, ",")
}

// StructDef defines validation rules for a struct type.
// Use StructDef.Field() to register per-field validators explicitly, or
// StructDef.UseTags() to discover validators from struct field tags.
// StructDef.ValidateWith() adds cross-field struct-level validation.
type StructDef struct {
	mu               sync.Mutex
	fields           []fieldRegistration
	structValidators []func(core.StructLevel) error
	useTags          bool
	tagName          string                           // struct tag to read; defaults to "validate"
	tagNameFunc      func(reflect.StructField) string // for RegisterTagNameFunc
	rulesOverride    map[string]string                // per-field rule overrides, for RegisterStructValidationMapRules
}

type fieldRegistration struct {
	Name     string
	Template core.Template
}

// NewStructDef creates a new StructDef with default settings.
func NewStructDef() *StructDef {
	return &StructDef{tagName: "validate"}
}

// UseTags enables tag-based field discovery from struct field tags.
// When enabled, Bind() reflects the struct and reads the configured tag name
// (default "validate") to build validation rules automatically.
// Explicitly registered fields (via Field()) override tag-based rules for
// the same field name.
//
// Unsupported tag patterns (dive, keys, omitempty, OR pipe, etc.) cause
// Bind() to return an error. This prevents silent partial validation.
func (d *StructDef) UseTags() *StructDef {
	d.useTags = true
	return d
}

// SetTagName overrides the default struct tag name ("validate").
func (d *StructDef) SetTagName(name string) *StructDef {
	d.tagName = name
	return d
}

// SetTagNameFunc registers a function that extracts the display name for a field.
// The returned name is used in error paths instead of the Go field name.
// Matches go-validator's RegisterTagNameFunc behavior.
func (d *StructDef) SetTagNameFunc(fn func(reflect.StructField) string) *StructDef {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tagNameFunc = fn
	return d
}

// SetRulesOverride sets per-field tag overrides, matching go-validator's
// RegisterStructValidationMapRules. Keys are field names, values are tag strings.
func (d *StructDef) SetRulesOverride(rules map[string]string) *StructDef {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rulesOverride = rules
	return d
}

// Field registers a field validator by name. The name must match a struct
// field (case-sensitive). Unexported fields cause an error at Bind time.
func (d *StructDef) Field(name string, tpl core.Template) *StructDef {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.fields = append(d.fields, fieldRegistration{Name: name, Template: tpl})
	return d
}

// ValidateWith registers a struct-level validator that runs after all
// field validators. It receives a StructLevel interface for accessing
// field values and reporting cross-field errors.
func (d *StructDef) ValidateWith(fn func(core.StructLevel) error) *StructDef {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.structValidators = append(d.structValidators, fn)
	return d
}

// Bind creates a StructChain bound to the given struct pointer.
// The StructChain walks all registered fields and runs their rules.
// It does not use CAS guards — create one per goroutine.
func (d *StructDef) Bind(target any) core.Rule {
	d.mu.Lock()
	fields := append([]fieldRegistration(nil), d.fields...)
	structValidators := append([]func(core.StructLevel) error(nil), d.structValidators...)
	useTags := d.useTags
	tagName := d.tagName
	tagNameFn := d.tagNameFunc
	rulesOverride := d.rulesOverride
	d.mu.Unlock()

	if tagName == "" {
		tagName = "validate"
	}

	val, rule := resolveBindTarget(target)
	if rule != nil {
		return rule
	}

	fc := getFieldCache(target, val.Type())

	compiled, used := bindExplicitFields(fields, fc, val, target)
	if err := compiledError(compiled); err != nil {
		return err
	}

	if useTags {
		compiled = bindTaggedFields(fc, val, tagName, tagNameFn, rulesOverride, used, compiled)
		if err := compiledError(compiled); err != nil {
			return err
		}
	}

	return chains.NewStructChain(val, compiled, structValidators)
}

// resolveBindTarget unwraps the target pointer and validates it's a struct.
// Returns the unwrapped reflect.Value and an error rule if the target is invalid.
func resolveBindTarget(target any) (reflect.Value, core.Rule) {
	val := reflect.ValueOf(target)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return reflect.Value{}, &structNilRule{}
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return reflect.Value{}, &structErrorRule{
			err: fmt.Errorf("skrub: StructDef.Bind target must be a struct pointer, got %T", target),
		}
	}
	return val, nil
}

// getFieldCache retrieves or builds the field metadata cache for a struct type.
func getFieldCache(target any, t reflect.Type) *structFieldCacheData {
	cacheKey := reflect.TypeOf(target)
	cacheIface, ok := structFieldCache.Load(cacheKey)
	if !ok {
		cache, loaded := structFieldCache.LoadOrStore(cacheKey, buildFieldCache(t))
		_ = loaded
		cacheIface = cache
	}
	return cacheIface.(*structFieldCacheData)
}

// compiledError checks if a compiled field list contains an error rule.
func compiledError(compiled []chains.StructFieldRule) core.Rule {
	if len(compiled) > 0 {
		if _, ok := compiled[0].Rule.(*structErrorRule); ok {
			return compiled[0].Rule
		}
	}
	return nil
}

// bindExplicitFields processes explicitly registered Field() calls.
// Returns compiled field rules and a set of used field names.
func bindExplicitFields(fields []fieldRegistration, fc *structFieldCacheData, val reflect.Value, target any) ([]chains.StructFieldRule, map[string]bool) {
	used := make(map[string]bool, len(fields))
	compiled := make([]chains.StructFieldRule, 0, len(fields))

	for _, f := range fields {
		idx, ok := fc.names[f.Name]
		if !ok {
			return []chains.StructFieldRule{{
				Rule: &structErrorRule{err: fmt.Errorf("skrub: field %q not found in %T", f.Name, target)},
			}}, used
		}
		cf := fc.fields[idx]
		if !cf.Addressable {
			return []chains.StructFieldRule{{
				Rule: &structErrorRule{err: fmt.Errorf("skrub: field %q is not addressable", f.Name)},
			}}, used
		}
		fieldVal := val.FieldByIndex(cf.Index)
		rule := f.Template.Bind(fieldVal.Addr().Interface(), "")
		compiled = append(compiled, chains.StructFieldRule{
			Name: f.Name,
			Rule: rule,
		})
		used[f.Name] = true
	}
	return compiled, used
}

// bindTaggedFields discovers fields from struct tags and appends their rules.
//
//nolint:cyclop // Complexity from tag name func, override rules, error handling
func bindTaggedFields(fc *structFieldCacheData, val reflect.Value, tagName string, tagNameFn func(reflect.StructField) string, rulesOverride map[string]string, used map[string]bool, compiled []chains.StructFieldRule) []chains.StructFieldRule {
	for _, cf := range fc.fields {
		if used[cf.Name] {
			continue
		}
		if !cf.Addressable {
			continue
		}
		sf := val.Type().FieldByIndex(cf.Index)

		// Check for override rules first (RegisterStructValidationMapRules)
		tag := ""
		if rulesOverride != nil {
			if rt, ok := rulesOverride[cf.Name]; ok {
				tag = rt
			}
		}
		if tag == "" {
			tag = sf.Tag.Get(tagName)
		}
		if tag == "" || tag == "-" {
			continue
		}

		// Compute field name for error paths (RegisterTagNameFunc)
		fieldName := cf.Name
		if tagNameFn != nil {
			if name := tagNameFn(sf); name != "" {
				fieldName = name
			}
		}

		fieldVal := val.FieldByIndex(cf.Index)
		tpl, err := buildTemplateFromTag(tag, fieldVal.Type())
		if err != nil {
			return []chains.StructFieldRule{{
				Rule: &structErrorRule{err: fmt.Errorf("skrub: field %q: %w", cf.Name, err)},
			}}
		}
		if tpl == nil {
			continue
		}

		rule := tpl.Bind(fieldVal.Addr().Interface(), "")
		compiled = append(compiled, chains.StructFieldRule{
			Name: fieldName,
			Rule: rule,
		})
	}
	return compiled
}

// buildTemplateFromTag parses a validate tag string and builds a core.Template
// that matches the field's Go type.
//
//nolint:cyclop // Type dispatch with string/int/slice/map is inherently branching
func buildTemplateFromTag(tag string, fieldType reflect.Type) (core.Template, error) {
	tag = expandAliases(tag)
	result, err := tagparser.Parse(tag)
	if err != nil {
		return nil, err
	}
	if len(result.Tags) == 0 && len(result.Alternatives) == 0 {
		return nil, nil
	}

	kind := fieldType.Kind()
	for kind == reflect.Ptr {
		kind = fieldType.Elem().Kind()
	}

	switch {
	case kind == reflect.String:
		return buildStringTemplate(result)
	case isIntKind(kind):
		return buildIntTemplate(result)
	case kind == reflect.Slice || kind == reflect.Array:
		elemType := fieldType.Elem()
		return buildSliceTemplate(result, elemType)
	case kind == reflect.Map:
		return buildMapTemplate(result, fieldType)
	default:
		return nil, fmt.Errorf("unsupported field type %v for tags", fieldType)
	}
}

// buildMapTemplate builds a MapRule template from parsed tags with keys/endkeys.
func buildMapTemplate(result tagparser.ParseResult, mapType reflect.Type) (core.Template, error) {
	keyType := mapType.Key()
	valType := mapType.Elem()

	// Handle structonly/nostructlevel flags
	// These are noted in the result but we return a simple template that
	// delegates to the appropriate validation

	// If HasKeys, build separate key and value templates
	if result.HasKeys {
		// Build key template from KeyTags
		var keyTpl core.Template
		if len(result.KeyTags) > 0 {
			keyTagStr := reconstructTagString(result.KeyTags, result)
			var err error
			keyTpl, err = buildTemplateFromTag(keyTagStr, keyType)
			if err != nil {
				return nil, err
			}
		}

		// Build value template from tags after endkeys (everything in Tags that
		// isn't a structural tag)
		valTags := make([]tagparser.Tag, 0, len(result.Tags))
		for _, t := range result.Tags {
			switch t.Name {
			case "dive", "structonly", "nostructlevel":
				continue
			default:
				valTags = append(valTags, t)
			}
		}

		// Also check if this is a nested dive for map values that are slices/structs
		// by checking Alternatives and remaining tags

		valTpl, err := buildTemplateFromTag(reconstructTagString(valTags, result), valType)
		if err != nil {
			return nil, err
		}

		return &mapRuleTemplate{keyTpl: keyTpl, valTpl: valTpl}, nil
	}

	// Without keys, just validate values using the tag chain
	valTpl, err := buildTemplateFromTag(reconstructTagString(result.Tags, result), valType)
	if err != nil {
		return nil, err
	}
	return &mapRuleTemplate{valTpl: valTpl}, nil
}

// mapRuleTemplate implements core.Template for map validation.
type mapRuleTemplate struct {
	keyTpl core.Template
	valTpl core.Template
}

func (m *mapRuleTemplate) Bind(target any, name string) core.Rule {
	return &mapRuleBinding{
		keyTpl: m.keyTpl,
		valTpl: m.valTpl,
		target: target,
		name:   name,
	}
}

// mapRuleBinding bridges between Template and Rule for map validation.
type mapRuleBinding struct {
	keyTpl core.Template
	valTpl core.Template
	target any
	name   string
}

func (m *mapRuleBinding) Validate(ctx *core.Context) error {
	// Delegate to MapRule, creating it here since Bind is the right time to
	// resolve the map iteration logic
	return chains.NewMapRule(m.keyTpl, m.valTpl, m.target, m.name).Validate(ctx)
}

// Common tag name constants to satisfy goconst linter.
const (
	tagRequired = "required"
	tagMin      = "min"
	tagMax      = "max"
	tagGte      = "gte"
	tagLte      = "lte"
)

func isIntKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

// buildStringTemplate builds a Template from parsed tags, handling OR groups
// and control tags like omitempty.
const tagDive = "dive"

func buildStringTemplate(result tagparser.ParseResult) (core.Template, error) {
	// dive on a non-slice field is an error
	for _, t := range result.Tags {
		if t.Name == tagDive {
			return nil, errors.New("dive tag is only valid on slice/array fields")
		}
	}
	if result.HasOrGroups {
		return buildOrTemplate(result, func(tags []tagparser.Tag) (core.Template, error) {
			return buildStringDefFromTags(tags)
		})
	}
	def, err := buildStringDefFromTags(result.Tags)
	if err != nil {
		return nil, err
	}
	return wrapControlTags(def, result.Tags)
}

// buildIntTemplate builds a Template from parsed tags for integer fields.
func buildIntTemplate(result tagparser.ParseResult) (core.Template, error) {
	for _, t := range result.Tags {
		if t.Name == tagDive {
			return nil, errors.New("dive tag is only valid on slice/array fields")
		}
	}
	if result.HasOrGroups {
		return buildOrTemplate(result, func(tags []tagparser.Tag) (core.Template, error) {
			return buildIntDefFromTags(tags)
		})
	}
	def, err := buildIntDefFromTags(result.Tags)
	if err != nil {
		return nil, err
	}
	return wrapControlTags(def, result.Tags)
}

// buildSliceTemplate builds a Template from parsed tags for slice fields.
// It handles 'dive' as a split point: pre-dive tags apply to the slice,
// post-dive tags apply to each element.
//
//nolint:cyclop // Complexity from dive split, element template, control flow
func buildSliceTemplate(result tagparser.ParseResult, elemType reflect.Type) (core.Template, error) {
	// Find the dive split point
	diveIdx := -1
	beforeDive := result.Tags[:0]
	for i, t := range result.Tags {
		if t.Name == "dive" {
			diveIdx = i
			break
		}
		beforeDive = append(beforeDive, t)
	}

	// Build slice-level def from pre-dive tags
	def := NewSliceDef()
	if len(beforeDive) > 0 {
		for _, t := range beforeDive {
			if isControlTagName(t.Name) {
				continue
			}
			if err := applySliceTag(def, t.Name, t.Param); err != nil {
				return nil, err
			}
		}
	}

	// If there are post-dive tags, build element templates
	if diveIdx >= 0 {
		afterDiveTags := result.Tags[diveIdx+1:]
		if len(afterDiveTags) > 0 {
			// Re-parse the element tags to handle OR and composition
			elemTagStr := reconstructTagString(afterDiveTags, result)
			tpl, err := buildTemplateFromTag(elemTagStr, elemType)
			if err != nil {
				return nil, err
			}
			if tpl != nil {
				def.Elements(tpl)
			}
		}
	}

	return def, nil
}

// reconstructTagString rebuilds a tag string from parsed tags.
func reconstructTagString(tags []tagparser.Tag, _ tagparser.ParseResult) string {
	parts := make([]string, 0, len(tags))
	for _, t := range tags {
		if t.Param != "" {
			parts = append(parts, t.Name+"="+t.Param)
		} else {
			parts = append(parts, t.Name)
		}
	}
	return strings.Join(parts, ",")
}

// buildOrTemplate creates an OrRule by building each alternative as a separate Template.
func buildOrTemplate(result tagparser.ParseResult, buildFn func([]tagparser.Tag) (core.Template, error)) (core.Template, error) {
	var alternatives []core.Template
	for _, altTags := range result.Alternatives {
		nonEmpty := make([]tagparser.Tag, 0, len(altTags))
		for _, t := range altTags {
			if t.Name != "" {
				nonEmpty = append(nonEmpty, t)
			}
		}
		if len(nonEmpty) == 0 {
			continue
		}
		tpl, err := buildFn(nonEmpty)
		if err != nil {
			return nil, err
		}
		alternatives = append(alternatives, tpl)
	}
	if len(alternatives) == 0 {
		return nil, errors.New("empty OR group")
	}
	if len(alternatives) == 1 {
		return alternatives[0], nil
	}
	return &orTemplate{alternatives: alternatives}, nil
}

// orTemplate implements core.Template by wrapping alternatives in an OrRule on Bind.
type orTemplate struct {
	alternatives []core.Template
}

func (o *orTemplate) Bind(target any, name string) core.Rule {
	var rules []core.Rule
	for _, alt := range o.alternatives {
		rules = append(rules, alt.Bind(target, name))
	}
	return chains.NewOrRule(rules)
}

// buildStringDefFromTags builds a StringDef from a flat list of tags (no OR).
func buildStringDefFromTags(tags []tagparser.Tag) (*StringDef, error) {
	def := NewStringDef()
	for _, t := range tags {
		if isControlTagName(t.Name) {
			continue
		}
		if err := applyStringTag(def, t.Name, t.Param); err != nil {
			return nil, err
		}
	}
	return def, nil
}

// buildIntDefFromTags builds an IntDef from a flat list of tags (no OR).
func buildIntDefFromTags(tags []tagparser.Tag) (*IntDef, error) {
	def := NewIntDef()
	for _, t := range tags {
		if isControlTagName(t.Name) {
			continue
		}
		if err := applyIntTag(def, t.Name, t.Param); err != nil {
			return nil, err
		}
	}
	return def, nil
}

// isControlTagName returns true for structural tag names that are not validators.
func isControlTagName(name string) bool {
	switch name {
	case tagDive, "omitempty", "omitnil", "omitzero", "isdefault":
		return true
	default:
		return false
	}
}

// wrapControlTags wraps a Def Template with control-flow wrappers.
func wrapControlTags(tpl core.Template, tags []tagparser.Tag) (core.Template, error) {
	var hasOmitEmpty, hasOmitNil, hasOmitZero, hasIsDefault bool
	for _, t := range tags {
		switch t.Name {
		case "omitempty":
			hasOmitEmpty = true
		case "omitnil":
			hasOmitNil = true
		case "omitzero":
			hasOmitZero = true
		case "isdefault":
			hasIsDefault = true
		}
	}
	if !hasOmitEmpty && !hasOmitNil && !hasOmitZero && !hasIsDefault {
		return tpl, nil
	}

	return &controlFlowTemplate{
		inner:     tpl,
		omitEmpty: hasOmitEmpty,
		omitNil:   hasOmitNil,
		omitZero:  hasOmitZero,
		isDefault: hasIsDefault,
	}, nil
}

// controlFlowTemplate wraps a Template's Bind to add control-flow wrapping.
type controlFlowTemplate struct {
	inner     core.Template
	omitEmpty bool
	omitNil   bool
	omitZero  bool
	isDefault bool
}

func (c *controlFlowTemplate) Bind(target any, name string) core.Rule {
	innerRule := c.inner.Bind(target, name)
	switch {
	case c.isDefault:
		return chains.NewIsDefaultRule(innerRule, target)
	case c.omitNil:
		return chains.NewOmitNilRule(innerRule, target)
	case c.omitZero:
		return chains.NewOmitZeroRule(innerRule, target)
	case c.omitEmpty:
		return chains.NewOmitEmptyRule(innerRule, target)
	default:
		return innerRule
	}
}

//nolint:cyclop // Tag dispatch uses a switch that is essentially a lookup table
func applyStringTag(def *StringDef, name, param string) error {
	switch name {
	case tagRequired:
		def.NotEmpty()
	case tagMin:
		n, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("invalid min parameter %q: %w", param, err)
		}
		def.Min(n)
	case tagMax:
		n, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("invalid max parameter %q: %w", param, err)
		}
		def.Max(n)
	case "len":
		n, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("invalid len parameter %q: %w", param, err)
		}
		def.Min(n).Max(n)
	case "email":
		def.Email()
	case "url":
		def.URL()
	case "uuid":
		def.UUID()
	case "ip":
		def.IP()
	case "ipv4":
		def.IPv4()
	case "ipv6":
		def.IPv6()
	case tagGte:
		n, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("invalid gte parameter %q: %w", param, err)
		}
		def.Min(n)
	case tagLte:
		n, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("invalid lte parameter %q: %w", param, err)
		}
		def.Max(n)
	default:
		return fmt.Errorf("unknown validation tag %q for string field", name)
	}
	return nil
}

func applyIntTag(def *IntDef, name, param string) error {
	switch name {
	case tagRequired:
		def.NotZero()
	case tagMin, tagGte:
		n, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("invalid %s parameter %q: %w", name, param, err)
		}
		def.Min(n)
	case tagMax, tagLte:
		n, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("invalid %s parameter %q: %w", name, param, err)
		}
		def.Max(n)
	case "eq":
		n, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("invalid eq parameter %q: %w", param, err)
		}
		def.Min(n).Max(n)
	default:
		return fmt.Errorf("unknown validation tag %q for integer field", name)
	}
	return nil
}

func applySliceTag(def *SliceDef, name, param string) error {
	switch name {
	case "required":
		def.NotEmpty()
	case "min", "gte":
		n, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("invalid %s parameter %q: %w", name, param, err)
		}
		def.MinLen(n)
	case "max", "lte":
		n, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("invalid %s parameter %q: %w", name, param, err)
		}
		def.MaxLen(n)
	default:
		return fmt.Errorf("unknown validation tag %q for slice field", name)
	}
	return nil
}

// structNilRule is a no-op rule returned when binding to a nil pointer.
type structNilRule struct{}

func (r *structNilRule) Validate(_ *core.Context) error { return nil }

// structErrorRule is a rule that always returns the given error.
type structErrorRule struct {
	err error
}

func (r *structErrorRule) Validate(_ *core.Context) error { return r.err }

// structFieldCacheData caches parsed struct field metadata to avoid repeated reflection.
var structFieldCache sync.Map // map[reflect.Type]*structFieldCacheData

type structFieldCacheData struct {
	fields []cachedField
	names  map[string]int
}

type cachedField struct {
	Name        string
	Index       []int
	Addressable bool
}

func buildFieldCache(t reflect.Type) *structFieldCacheData {
	n := t.NumField()
	fields := make([]cachedField, 0, n)
	names := make(map[string]int, n)
	idx := 0
	for i := 0; i < n; i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		if sf.Anonymous {
			// Flatten embedded struct fields
			if sf.Type.Kind() == reflect.Struct {
				embedded := buildFieldCache(sf.Type)
				for _, ef := range embedded.fields {
					fullIdx := append([]int{i}, ef.Index...)
					cached := cachedField{
						Name:        ef.Name,
						Index:       fullIdx,
						Addressable: ef.Addressable,
					}
					fields = append(fields, cached)
					names[ef.Name] = idx
					idx++
				}
			}
			continue
		}
		cached := cachedField{
			Name:        sf.Name,
			Index:       []int{i},
			Addressable: true,
		}
		fields = append(fields, cached)
		names[sf.Name] = idx
		idx++
	}
	return &structFieldCacheData{fields: fields, names: names}
}

func init() {
	RegisterAlias("iscolor", "hexcolor|rgb|rgba|hsl|hsla|cmyk")
	RegisterAlias("country_code", "iso3166_1_alpha2|iso3166_1_alpha3|iso3166_1_alpha_numeric")
}
