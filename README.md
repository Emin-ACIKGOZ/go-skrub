# go-skrub

**go-skrub** is a zero-panic, zero-dependency validation library for Go. It provides both a **declarative tag-based API** (compatible with go-validator struct tags) and a **programmatic builder API** for explicit validation rules. Rules are goroutine-safe, allocations are minimal, and misuse always returns errors — never panics.

## Features

- **Struct tag validation**: `validate:"required,min=3,max=50,email"` — drop-in compatible with go-validator
- **Goroutine-safe rules**: Stateless `Rule` types (`StringRule`, `IntRule`, `SliceRule`, `MapRule`) safe for concurrent use
- **Error accumulation**: Collect all validation errors instead of short-circuiting — `AccumulateErrors` mode
- **OR pipe**: `required|email` — first success wins
- **Control flow tags**: `omitempty`, `omitnil`, `omitzero`, `isdefault`, `dive`, `keys`/`endkeys`
- **Alias support**: `RegisterAlias("iscolor", "hexcolor|rgb|rgba|hsl|hsla|cmyk")`
- **Recursive validation**: N-dimensional slices, maps with key+value validation
- **Cross-field validation**: Struct-level callbacks via `ValidateWith`
- **Programmatic API**: Fluent builder for cases where tags aren't enough
- **Zero-panic**: All errors are returned — `ErrMisuse`, `ErrConcurrencyViolation`, `ErrPoolExhausted`
- **Zero runtime dependencies**: The library itself imports only the Go standard library
- **HTTP middleware**: Built-in `net/http` support
- **Adapters**: Validate `time.Time`, UUIDs, and custom types via the `Valuer` interface

## Installation

```bash
go get github.com/Emin-ACIKGOZ/go-skrub
```

## Quick Start (Tag-Based)

```go
type User struct {
    Name  string `validate:"required,min=3,max=50"`
    Email string `validate:"required,email"`
    Age   int    `validate:"min=18"`
    Tags  []string `validate:"required,min=1,dive,max=20"`
}

user := User{Name: "Alice", Email: "alice@example.com", Age: 25, Tags: []string{"go"}}

rule := skrub.DefStruct().UseTags().Bind(&user)
err := skrub.Validate(&user, rule)
```

## Quick Start (Programmatic)

```go
type User struct {
    Name  string
    Email string
    Age   int
}

user := User{Name: "Alice", Email: "alice@example.com", Age: 25}

err := skrub.Validate(&user,
    skrub.DefString().Min(3).Max(50).BindStateless(&user.Name, "name"),
    skrub.DefString().Email().BindStateless(&user.Email, "email"),
    skrub.DefInt().Min(18).BindStateless(&user.Age, "age"),
)
```

## Supported Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Value must be non-zero/non-nil | `validate:"required"` |
| `min` | Minimum length/value | `validate:"min=3"` |
| `max` | Maximum length/value | `validate:"max=100"` |
| `len` | Exact length | `validate:"len=10"` |
| `gte` | Greater than or equal | `validate:"gte=18"` |
| `lte` | Less than or equal | `validate:"lte=99"` |
| `gt` | Greater than | `validate:"gt=18"` |
| `lt` | Less than | `validate:"lt=99"` |
| `eq` | Equal to | `validate:"eq=42"` |
| `ne` | Not equal | `validate:"ne=0"` |
| `email` | Email format | `validate:"email"` |
| `url` | HTTP/HTTPS URL only | `validate:"url"` |
| `uuid` | UUID format (8-4-4-4-12) | `validate:"uuid"` |
| `ip` | IP address (v4 or v6) | `validate:"ip"` |
| `ipv4` | IPv4 address | `validate:"ipv4"` |
| `ipv6` | IPv6 address | `validate:"ipv6"` |
| `\|` (pipe) | OR — first success wins | `validate:"required\|email"` |
| `omitempty` | Skip validation if empty/nil/zero | `validate:"omitempty,min=3"` |
| `omitnil` | Skip validation if nil only | `validate:"omitnil,min=3"` |
| `omitzero` | Skip if zero (stricter for slices) | `validate:"omitzero"` |
| `isdefault` | Pass only when value IS zero | `validate:"isdefault"` |
| `dive` | Recurse into slice/map elements | `validate:"dive,min=2"` |
| `keys` | Validate map keys | `validate:"dive,keys,min=1,endkeys"` |
| `endkeys` | End map key validation block | `validate:"dive,keys,...,endkeys,value_tags"` |

## Error Accumulation

By default, go-skrub returns on the first error (short-circuit). Enable accumulation to collect all errors:

```go
err := skrub.ValidateWithConfig(core.Config{AccumulateErrors: true}, rule)
if err != nil {
    if ves, ok := err.(core.ValidationErrors); ok {
        for _, fe := range ves {
            fmt.Println(fe.Path, fe.Reason)
        }
    }
}
```

## Cross-Field Validation

Struct-level callbacks enable cross-field rules like password confirmation:

```go
rule := skrub.DefStruct().
    Field("Password", skrub.DefString().Min(8)).
    Field("ConfirmPassword", skrub.DefString()).
    ValidateWith(func(sl core.StructLevel) error {
        pw, _ := sl.FieldValue("Password")
        confirm, _ := sl.FieldValue("ConfirmPassword")
        if pw != confirm {
            sl.ReportError("ConfirmPassword", "must match Password")
        }
        return nil
    }).
    Bind(&user)
```

## Recursive / Matrix Validation

```go
// 3D matrix where every integer must be >= 0
matrixTemplate := skrub.DefMatrix(3, skrub.DefInt().Min(0))
matrixTemplate.Bind(&data, "matrix").Validate(ctx)
```

## Map Validation

```go
type Container struct {
    Data map[string]string `validate:"dive,keys,min=1,max=10,endkeys,required"`
}
rule := skrub.DefStruct().UseTags().Bind(&container)
```

## Custom Validators

Register custom type validators:

```go
skrub.Register(func(u *User, name string) core.Rule {
    return NewUserChain(u, name)
})
```

Register tag validators for struct tag discovery:

```go
skrub.RegisterTagValidator("hexcolor", func(param string) (core.Template, error) {
    return skrub.DefString().Pattern(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`), nil
})
```

## Aliases

```go
skrub.RegisterAlias("iscolor", "hexcolor|rgb|rgba|hsl|hsla|cmyk")
// Then use in tags: `validate:"iscolor"`
```

## HTTP Middleware

```go
hooks := middleware.NewHooks()
hooks.Compose(func(w http.ResponseWriter, r *http.Request, err error) {
    http.Error(w, err.Error(), http.StatusBadRequest)
})
handler := hooks.Validate(validateRequest, nextHandler)
```

## Adapters

Validate custom types without reflection:

```go
skrub.String(adapters.Time(t), "created_at").Pattern(`^\d{4}-`)
skrub.String(adapters.UUID(uuidObj), "id").UUID()
```

## Concurrency

- **Stateless Rules** (from `BindStateless` or `Def` → `Bind`): Goroutine-safe, no CAS guards
- **Stateful Chains** (from `skrub.String()`, `chains.NewStringChain()`): CAS-guarded, concurrent access returns `ErrConcurrencyViolation`
- All Rules are pooled for minimal allocation

## Benchmarks (i5-1135G7, go1.26)

Defaults are goroutine-safe. A `BindCAS()` fast path is available for single-goroutine use.

| Scenario | go-skrub (Bind) | go-skrub (BindCAS) | go-validator |
|----------|----------------|-------------------|-------------|
| Small struct (4 fields) | **950ns** / 7 allocs | 1,164ns / 13 allocs | 1,500ns / 10 allocs |
| Deep matrix (1000 el) | 147μs / 1,472 allocs | **43μs** / 2 allocs | 107μs / 2,322 allocs |
| URL validation | **305ns** / 2 allocs | — | — |
| Email validation | **492ns** / 1 alloc | — | 900ns / 5 allocs |

`Bind()` is goroutine-safe and at parity with go-validator for structs. `BindCAS()` matches the original zero-alloc performance for single-goroutine use. See `bench_results.txt` for full detail.

## Error Model

- `FieldError{Path, Value, Reason, Cause}` — path-aware validation failure
- `RecursionError{Path, Depth, MaxDepth}` — recursion limit exceeded
- `ValidationErrors` — accumulated errors (implements `Unwrap() []error` for `errors.Is`/`errors.As`)
- Sentinels: `ErrMisuse`, `ErrConcurrencyViolation`, `ErrPoolExhausted`

## Design Goals

- **Correctness first** — no panics, clear error paths, well-defined behavior for edge cases
- **Zero dependencies** — the library itself imports only stdlib
- **Performance** — pre-bound rules, immutable configs, minimal allocation on hot paths
- **Goroutine safety** — stateless Rules safe for concurrent use
- **go-validator compatible** — tag parsing with full structural parity (see `MIGRATION_GUIDE.md`)

## License

MIT
