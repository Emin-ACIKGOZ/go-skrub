# go-skrub Roadmap

## v1.0 (current)

- [x] Error accumulation (`RecordError`, `ValidationErrors`, `emitError`)
- [x] Goroutine-safe stateless Rules (`StringRule`, `IntRule`, `SliceRule`)
- [x] Struct traversal with explicit field binding (`StructDef.Field()`)
- [x] Cross-field validation via `StructDef.ValidateWith()`
- [x] Tag-based struct validation (`StructDef.UseTags()`)
- [x] Tag parser with AND composition and type dispatch
- [x] Built-in tag validators: required, min, max, len, email, url, uuid, ip, ipv4, ipv6, gte, gt, lte, lt, eq, ne
- [x] Parity tests against go-validator
- [x] Race-condition tests for concurrent validation
- [x] Object pooling for Context (safe, bounded)
- [x] Zero-panic guarantee
- [x] Zero runtime dependencies

## v1.1

- [ ] **`CrossFieldDef` standalone builder** — `DefCrossField().Eq().Bind(&a, "a", &b, "b")`
  - All comparison operators (Eq, Ne, Gt, Gte, Lt, Lte)
  - Full numeric type support including `time.Time`
- [ ] **OR-pipe (`|`) tag semantics** — `min=5|email` for alternative validators
- [ ] **`dive` tag support** — recursive validation of slice/map elements from struct tags
- [ ] **`omitempty` / `omitnil` / `omitzero` tag support**
- [ ] **Alias support** — `RegisterAlias("iscolor", "hexcolor|rgb|rgba|hsl|hsla")`
- [ ] **go-validator error parity** — `FieldError` gains `Tag()`, `ActualTag()`, `Param()`, `Kind()`, `Type()`
- [ ] **i18n / translations** — `Translate()` method on `FieldError`, locale packages
- [ ] **More built-in validators:**
  - String: alpha, alphanum, ascii, lowercase, uppercase, contains, starts_with, ends_with, json, base64, hexcolor, numeric, boolean
  - Network: hostname, fqdn, cidr, mac, port
  - Format: datetime, hostname, credit_card, ssn, ein, semver, cve, isbn
  - Conditional: required_if, required_with, excluded_if, skip_unless

## v1.2+

- [ ] **Map validation** — `keys` / `endkeys` tag support
- [ ] **`validateFn` tag** — call `Validate() error` on field types
- [ ] **Full go-validator tag parity** — all ~183 go-validator tags implemented
- [ ] **Code generation option** — compile-time validation via codegen
- [ ] **Memory optimization** — optional `sync.Pool` for SliceRule element flyweights if benchmarks show >2x allocation overhead vs SliceChain
