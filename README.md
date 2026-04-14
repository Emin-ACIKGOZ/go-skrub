# go-skrub

**go-skrub** is a zero-panic, request-scoped validation library for Go that provides **stateful, bound validators** with strong concurrency guarantees, reusable templates, and controlled recursion. It is designed for correctness, composability, and safe reuse in high-concurrency environments.

## Features

- **Bound validation chains**: Validators bind directly to target memory addresses.
- **Reusable templates**: Define validation once, bind many times.
- **Zero-panic guarantees**: Misuse and concurrency errors are returned explicitly.
- **Concurrency-safe by design**: Atomic guards prevent concurrent chain execution.
- **Recursive validation**: Safe traversal of nested structures with depth limits.
- **Extensible**: Register custom validators for domain-specific types.
- **HTTP middleware support**: First-class integration with `net/http`.
- **Adapters**: Validate custom types (e.g. `time.Time`, UUIDs) without reflection.

## Installation

```bash
go get github.com/Emin-ACIKGOZ/go-skrub
```

## Core Concepts

### Templates vs Chains

* **Templates (`defs`)** are *unbound* validation definitions.
* **Chains (`chains`)** are *stateful*, bound validators targeting a specific value.
* Binding a template produces a chain (`core.Rule`) that can be executed.

### Validation Flow

1. Define templates (`DefString`, `DefInt`, `DefSlice`, …)
2. Bind templates to targets (via `Bind` or facade helpers)
3. Execute validation with `Validate` or `ValidateWithConfig`

## Quick Example

```go
type User struct {
    Name  string
    Tags  []string
}

err := skrub.Validate(
    &user,
    skrub.DefString().Min(3).Bind(&user.Name, "name"),
    skrub.DefSlice().
        MinLen(1).
        Elements(skrub.DefString().Max(20)).
        Bind(&user.Tags, "tags"),
)
```

## Validators

### String Validators

* **`URL()`**: Validates HTTP/HTTPS URLs with valid scheme and host.
  ```go
  skrub.String(&webhook, "url").URL()
  ```

* **`IP()`**, **`IPv4()`**, **`IPv6()`**: Validate IP addresses.
  ```go
  skrub.String(&addr, "ipv4").IPv4()
  skrub.String(&addr, "ipv6").IPv6()
  skrub.String(&addr, "ip").IP()  // Accepts both
  ```

* **`NotEmpty()`**: Rejects empty strings.
  ```go
  skrub.String(&name, "name").NotEmpty()
  ```

* **`Email()`**, **`UUID()`**, **`Pattern()`**, **`Min()`**, **`Max()`**: Standard string validators.

### Integer Validators

* **`NotZero()`**: Rejects zero values.
  ```go
  skrub.DefInt().NotZero().Bind(&count, "count")
  ```

* **`MatchString(regexp)`**: Validates string representation against a regex.
  ```go
  pattern := regexp.MustCompile(`^[1-9]\d{2}$`)
  skrub.DefInt().MatchString(pattern).Bind(&id, "id")
  ```

* **`Min()`**, **`Max()`**: Boundary validation.

### Slice Validators

* **`NotEmpty()`**: Rejects empty slices.
  ```go
  skrub.Slice(&items, "items").NotEmpty()
  ```

* **`MinLen()`**, **`MaxLen()`**: Length constraints.
* **`Elements(def)`**: Validate each element recursively.

## Facade API

The `skrub` package re-exports core types and provides helpers:

* `DefString`, `DefInt`, `DefSlice`, `DefMatrix`
* `String`, `Slice` (direct chain creation)
* `Validate`, `ValidateWithConfig`
* `Register` for custom validators
* Error types: `ErrMisuse`, `ErrConcurrencyViolation`, `ErrPoolExhausted`

## Recursive Validation

```go
matrix := skrub.DefMatrix(3, skrub.DefInt().Min(0))
matrix.Bind(&data, "matrix").Validate(ctx)
```

Recursion is guarded by:

* **MaxDepth** (hard stop)
* **WarningThreshold** (soft warning hook)

## Concurrency & Safety

* Chains use atomic state guards (`Acquire` / `Release`)
* Concurrent use of a chain returns `ErrConcurrencyViolation`
* All misuse is reported via errors—never panics

## Extensibility

Register custom validators with full type safety:

```go
skrub.Register(func(u *User, name string) core.Rule {
    return NewUserChain(u, name)
})
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

* `adapters.Time`, `TimeWithLayout`
* `adapters.UUID`, `UUIDPtr`

```go
skrub.String(adapters.Time(t), "created_at").Pattern(`^\d{4}-`)
```

## Error Model

* `FieldError`: Path-aware validation failure
* `RecursionError`: Max depth exceeded
* Errors support standard `errors.Is` / `errors.As`

## Design Goals

* Explicit state
* Predictable execution
* No hidden globals
* No reflection-heavy hot paths
* Production-safe defaults

## License

MIT

