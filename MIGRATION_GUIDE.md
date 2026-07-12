# Migrating from go-validator to go-skrub

## Quick Start

If you're using go-validator and want to switch, here's the minimal diff:

```go
// Before (go-validator):
type User struct {
    Name  string `validate:"required,min=3,max=50"`
    Email string `validate:"required,email"`
    Age   int    `validate:"min=18"`
}
v := validator.New()
err := v.Struct(&user)

// After (go-skrub):
type User struct {
    Name  string `validate:"required,min=3,max=50"`
    Email string `validate:"required,email"`
    Age   int    `validate:"min=18"`
}
rule := skrub.DefStruct().UseTags().Bind(&user)
err := skrub.Validate(&user, rule)
```

Most structs with basic tags work immediately. See below for behavioral differences.

## Intentional Behavioral Divergences

These are design decisions where go-skrub deliberately differs from go-validator.

### `url` — HTTP/HTTPS only

| Tag | go-validator | go-skrub | Rationale |
|-----|-------------|----------|-----------|
| `url` | Accepts any URL scheme (http, https, ftp, file, mailto, irc, etc.) | Accepts only http:// and https:// | `file://` and `javascript://` URLs are a common security issue. go-validator's own `http_url` and `https_url` tags exist because plain `url` is too permissive. Skrub merges them into one unambiguous validator. |

**Migration**: If you need generic URI validation, use a custom validator or explicitly validate your scheme separately.

### `required` on non-pointer structs — Always enforced

| Tag | go-validator | go-skrub | Rationale |
|-----|-------------|----------|-----------|
| `required` on non-pointer struct | Silently skipped (backward compat) | Always enforced | go-validator considers this a design mistake they plan to fix in their next major version. Skrub doesn't replicate it. |

**Migration**: None needed — this is strictly stricter, so previously broken code gets caught earlier.

### Panics vs Errors

| Scenario | go-validator | go-skrub | Rationale |
|----------|-------------|----------|-----------|
| Unknown tag | Panics at validation time | Returns error at Bind time | Errors are recoverable; panics are not. Bind-time errors surface during setup, not at runtime. |
| `dive` on non-slice field | Panics at validation time | Returns error at Bind time | Same rationale. |

### Tag whitespace

| Aspect | go-validator | go-skrub |
|--------|-------------|----------|
| `required ,email` | Fails (space in tag name) | Works (whitespace trimmed) |

## Remaining Gaps (v1.1+)

These go-validator features work differently or are not yet implemented:

| Feature | Status | Notes |
|---------|--------|-------|
| `hexcolor`, `rgb`, `rgba`, `hsl`, `hsla`, `cmyk` | ❌ | `iscolor` alias is registered but underlying validators aren't implemented |
| `keys`/`endkeys` | ✅ | Implemented — works on `map` fields with `dive,keys,...` |
| `structonly`/`nostructlevel` | ✅ | Parsed and flagged, nested struct control works |
| `isdefault` | ✅ | Implemented — passes only when value IS zero |
| `omitzero` | ✅ | Implemented (stronger than omitempty for slices/maps) |
| Alias expansion | ✅ | `RegisterAlias()` with recursive expansion |
| `RegisterTagNameFunc` | ✅ | `StructDef.SetTagNameFunc()` |
| Map rules override | ✅ | `StructDef.SetRulesOverride()` |
| Cross-field tags (`eqfield`, `required_if`) | ❌ | Cannot be expressed via tags yet; use `StructDef.ValidateWith()` |
| Conditional tags (`required_with`, `excluded_if`) | ❌ | Same — use struct-level callbacks |
| Nested struct tag traversal | ❌ | Tag-based recursion into nested structs (works via explicit `Field()`) |

## Feature Equivalence

| Feature | go-validator | go-skrub |
|---------|-------------|----------|
| Struct tags | ✅ | ✅ (UseTags) |
| Programmatic API | ✅ (RegisterValidation) | ✅ (DefString/Int/Slice builders) |
| Goroutine-safe singleton | ✅ | ✅ (stateless Rules) |
| Zero-panic | ❌ | ✅ |
| Zero dependencies | ❌ | ✅ |
| Error accumulation | ✅ | ✅ (AccumulateErrors) |
| Cross-field validation | ✅ tags | ✅ (ValidateWith) |
| i18n / translations | ✅ | ❌ (v1.1+) |
| ~183 built-in validators | ✅ | ~20 (v1.1+ adds more) |
