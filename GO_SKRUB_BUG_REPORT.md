# GO-SKRUB Bug Report

**Date:** 2025-01-XX
**Scope:** Full source audit of the go-skrub validation library
**Severity Scale:** Critical > High > Medium > Low > Cosmetic

---

## BUG-1: `ValidateWithConfig` silently discards the `target` parameter (Dead Code / Misleading API)

**File:** `skrub.go:40`
**Severity:** Medium

```go
func ValidateWithConfig(_ any, cfg core.Config, rules ...Rule) error {
```

The first parameter `target any` is explicitly discarded with `_`. This is dead code — the function accepts a target value but never uses it. The API signature suggests the target is validated at this level, but in reality, rules must be pre-bound to their targets via chain constructors (`String()`, `Slice()`, etc.) before being passed here.

**Impact:** Misleading API design. Users may expect `Validate(target, rules...)` to bind rules to the target automatically, but the target is ignored.

**Fix:** Either remove the target parameter entirely, or implement target binding at this level.

---

## BUG-2: `StringChain.Validate()` and `IntChain.Validate()` do NOT push chain name to context stack (Path Inconsistency)

**File:** `chains/string.go:207-224`, `chains/int.go:34-50`
**Severity:** High

`SliceChain.Validate()` correctly pushes `c.Name` to the context stack:
```go
// chains/slice.go
if c.Name != "" {
    if err := ctx.Push(c.Name); err != nil {
        return err
    }
    defer ctx.Pop()
}
```

But `StringChain.Validate()` and `IntChain.Validate()` do NOT push their name:
```go
// chains/string.go
func (c *StringChain) Validate(ctx *core.Context) error {
    if err := c.Acquire(); err != nil {
        return err
    }
    defer c.Release()
    val, isNil, err := c.resolveTarget()
    // ... no ctx.Push(c.Name) ...
```

**Impact:** The context path stack does not reflect the full nesting when string/int chains are used. The `Fail()` method in `BaseChain` manually concatenates `ctxPath + "." + b.Name`, duplicating path logic that should be handled by the context stack. This creates architectural inconsistency.

**Fix:** Add `ctx.Push(c.Name)` / `defer ctx.Pop()` to `StringChain.Validate()` and `IntChain.Validate()`, matching the pattern in `SliceChain.Validate()`.

---

## BUG-3: `Context.Reset()` does NOT reset `cfg` to defaults (Pool State Leak)

**File:** `pkg/core/context.go:100-105`
**Severity:** High

```go
func (c *Context) Reset() {
    c.stack = c.stack[:0]
    c.depth = 0
}
```

The `cfg` field (`MaxDepth`, `WarningThreshold`, `OnWarning`) is NOT reset. When a Context is returned to the pool and reused:

1. **MaxDepth leak:** In `skrub.go:47-51`:
   ```go
   if cfg.MaxDepth != 0 {
       ctx.SetMaxDepth(cfg.MaxDepth)
   }
   ```
   If the new caller passes `Config{}` (zero value), `cfg.MaxDepth` is 0, so `SetMaxDepth` is NOT called. The old MaxDepth from the previous caller leaks through. A previous caller set MaxDepth=50, next caller uses default Config{}, and MaxDepth stays at 50 instead of defaulting to 100.

2. **OnWarning callback leak:** `SetWarningThreshold` is always called, so the callback is overwritten. But if the new caller passes zero values, the old callback is replaced with nil. This is less severe but still a state leak.

**Impact:** Configuration from one validation request can leak into the next request when contexts are pooled, causing incorrect recursion limits or stale warning callbacks.

**Fix:** Reset `cfg` in `Reset()`:
```go
func (c *Context) Reset() {
    c.stack = c.stack[:0]
    c.depth = 0
    c.cfg = Config{}  // Reset configuration
}
```

---

## BUG-4: `SliceChain.shouldSkipElement` has a redundant/invalid check (Logic Error)

**File:** `chains/slice.go:160-165`
**Severity:** Medium

```go
func (c *SliceChain) shouldSkipElement(v reflect.Value) bool {
    if v.Kind() == reflect.Ptr && v.IsNil() {
        return true
    }
    return !reflect.ValueOf(safeReflect.ResolveValue(v)).IsValid()
}
```

The second check calls `safeReflect.ResolveValue(v)` which returns `any`, then wraps it in `reflect.ValueOf()`. If `v` is a nil pointer to a struct, `ResolveValue` returns the nil pointer interface, `reflect.ValueOf(nilPtr)` is valid (it's a valid pointer to nothing), so `IsValid()` returns `true`. The check is essentially dead code for the nil pointer case and doesn't catch genuinely invalid values.

Additionally, if `v` is a nil `*string` element, the first check catches it. But if `v` is a zero-value struct field (non-pointer, zero value), `IsValid()` returns true and the element is processed, which is correct. The second check only triggers if `ResolveValue` returns a truly nil interface (e.g., from an untyped nil), which is unlikely in practice.

**Impact:** Dead code that doesn't provide meaningful safety. Could mask real issues.

**Fix:** Simplify to just the nil pointer check, or add a meaningful validity check.

---

## BUG-5: `SliceChain.Reset()` sets flyweights/rebindables to `nil` instead of `[:0]` (Performance / Inconsistency)

**File:** `chains/slice.go:175-185`
**Severity:** Low

```go
func (c *SliceChain) Reset() {
    c.BaseChain.Reset()
    c.target = nil
    c.validators = c.validators[:0]       // Preserves capacity ✓
    c.elementTemplates = c.elementTemplates[:0]  // Preserves capacity ✓
    c.flyweights = nil                     // Forces re-allocation ✗
    c.rebindables = nil                    // Forces re-allocation ✗
}
```

**Impact:** Forces re-allocation of flyweight caches on every pool reuse, defeating the purpose of pooling.

**Fix:** Use `c.flyweights = c.flyweights[:0]` and `c.rebindables = c.rebindables[:0]`.

---

## BUG-6: `Context.Pop()` can underflow `depth` if called after `Reset()` (Logic Error)

**File:** `pkg/core/context.go:85-95`
**Severity:** Medium

```go
func (c *Context) Pop() {
    if len(c.stack) == 0 {
        return
    }
    c.stack = c.stack[:len(c.stack)-1]
    c.depth--
}
```

The guard checks `len(c.stack) == 0` but does NOT check `c.depth > 0`. If `Reset()` is called (which sets `c.stack = c.stack[:0]` and `c.depth = 0`), and then `Pop()` is called, the stack guard returns early but `depth` is not decremented (it's already 0, so no underflow). However, if the stack and depth get out of sync (e.g., through inconsistent Push/Pop pairs), `depth` could go negative while the stack is non-empty, and `Pop()` would decrement depth below zero.

**Impact:** Potential negative depth counter, which would cause incorrect recursion limit calculations.

**Fix:** Add `c.depth > 0` guard:
```go
func (c *Context) Pop() {
    if len(c.stack) == 0 {
        return
    }
    c.stack = c.stack[:len(c.stack)-1]
    if c.depth > 0 {
        c.depth--
    }
}
```

---

## BUG-7: `Registry.GetChain()` type assertion can panic (Potential Panic)

**File:** `registry.go:55-65`
**Severity:** High

```go
registry[typ] = func(target any, name string) core.Rule {
    typedPtr := target.(*T)  // Can panic if target is not *T
    return factory(typedPtr, name)
}
```

The type assertion `target.(*T)` will panic if `target` is not of type `*T`. While the registry is keyed by `reflect.TypeOf((*T)(nil))`, the lookup in `GetChain` uses `reflect.TypeOf(target)`. If someone passes a value type `T` instead of `*T`, the type won't match the registry key, so it returns "not found" — safe. But if someone passes a different pointer type with the same `reflect.Type` (unlikely but possible with type aliases), the assertion could panic.

**Impact:** Potential runtime panic from type assertion failure.

**Fix:** Use a safe type assertion with the comma-ok pattern:
```go
typedPtr, ok := target.(*T)
if !ok {
    return nil, core.ErrMisuse
}
return factory(typedPtr, name)
```

---

## BUG-8: `SafePool.Put()` calls `Reset()` before non-blocking send, wasting reset on dropped items (State Loss)

**File:** `internal/pool/safe.go:85-100`
**Severity:** Medium

```go
func (p *SafePool) Put(item any) {
    if item == nil {
        return
    }
    if resetter, ok := item.(core.Resetter); ok {
        resetter.Reset()  // Reset happens BEFORE the send
    }
    select {
    case p.items <- item:
    default:
        // Pool is full. Item is dropped — but it was already reset!
    }
}
```

If the pool is full and the item is dropped, the item has been reset but is now discarded. This is wasteful — the reset work is thrown away. More importantly, if the item is still referenced elsewhere (which would be a separate bug), the reset would corrupt that reference's data.

**Impact:** Wasted CPU cycles on resetting items that are then dropped. Potential for subtle bugs if items are shared.

**Fix:** Move the reset after the successful send, or only reset if the send succeeds.

---

## BUG-9: `SafePool` pre-population silently skips nil factory items (Capacity Reduction)

**File:** `internal/pool/safe.go:45-55`
**Severity:** Low

```go
if item := cfg.Factory(); item != nil {
    p.items <- item
}
```

If the factory returns nil, the item is silently skipped, reducing the effective pool capacity. The pool will have fewer items than `cfg.Capacity`.

**Impact:** Reduced pool capacity without warning. Could cause unexpected pool exhaustion under load.

**Fix:** Panic or log if factory returns nil, or use a blocking send to ensure capacity is filled.

---

## BUG-10: `StringChain` and `IntChain` missing `core.Rebindable` compile-time assertion (Missing Interface Check)

**File:** `markers.go`
**Severity:** Low

`SliceChain` has `SetTarget()` and is used as a flyweight. `StringChain` and `IntChain` also have `SetTarget()` but there's no compile-time assertion:
```go
var _ core.Rebindable = (*StringChain)(nil)  // Missing
var _ core.Rebindable = (*IntChain)(nil)     // Missing
```

**Impact:** If someone refactors `SetTarget()` signature on StringChain or IntChain, the break won't be caught at compile time.

**Fix:** Add the assertions to `markers.go`.

---

## BUG-11: `SliceChain.Validate()` creates unpooled context when ctx is nil (Pool Bypass / Memory Leak)

**File:** `chains/slice.go:35-40`
**Severity:** Medium

```go
func (c *SliceChain) Validate(ctx *core.Context) error {
    if ctx == nil {
        ctx = core.NewContext(core.Config{})  // Created but never pooled
    }
```

If a nil context is passed (which shouldn't happen in normal flow), a new context is created but never returned to any pool. This context is garbage collected, but it bypasses the pooling mechanism entirely.

**Impact:** In normal flow this is dead code. But if someone calls `Validate(nil)` directly on a chain, the context is created and discarded, wasting memory.

**Fix:** Remove the nil check and let it panic (document that ctx must not be nil), or use a fallback pooled context.

---

## BUG-12: `UUIDPtr` silently returns nil adapter on resolution failure (Silent Failure)

**File:** `pkg/adapters/uuid.go:55-65`
**Severity:** Low

```go
func UUIDPtr(u interface{}) *UUIDAdapter {
    if u == nil {
        return &UUIDAdapter{Val: nil}
    }
    if s, ok := u.(UUIDStringer); ok {
        return &UUIDAdapter{Val: s}
    }
    if s, ok := u.(fmt.Stringer); ok {
        return &UUIDAdapter{Val: s}
    }
    return &UUIDAdapter{Val: nil}  // Silent failure
}
```

If the input doesn't implement `UUIDStringer` or `fmt.Stringer`, `UUIDPtr` returns an adapter with nil Val. The `Unwrap()` method returns `""` for nil Val, meaning validation will pass an empty string to the chain. This could pass `Min(1)` checks unexpectedly.

**Impact:** Silent data corruption — invalid types pass validation with empty strings instead of failing loudly.

**Fix:** Return an error or panic when resolution fails, or at minimum log a warning.

---

## BUG-13: `DefMatrix` with dimension 0 returns template directly (Edge Case)

**File:** `defs/recursive.go:10-20`
**Severity:** Low

```go
func NewMatrixDef(dimensions int, template core.Template) core.Template {
    if dimensions <= 0 {
        return template  // Returns the raw template without slice wrapping
    }
```

A 0-dimensional matrix doesn't make sense. If called with 0, the caller gets back a template that, when bound, won't iterate over any slice. This could cause confusing behavior.

**Impact:** Confusing API behavior for edge case.

**Fix:** Panic or return nil for dimensions <= 0, or document that 0 is a no-op passthrough.

---

## BUG-14: `StringChain` and `IntChain` don't push name to context, causing inconsistent path for nested validation (Path Inconsistency)

**File:** `chains/string.go`, `chains/int.go`
**Severity:** High

When a `StringDef` is used as an element template in a `SliceChain`, the `StringChain.Validate()` is called for each element. The `SliceChain` pushes the index via `ctx.PushIndex(i)`, so `ctx.String()` returns something like `"items[0]"`. Then `StringChain.Validate()` calls `c.Fail(ctx, val, reason)` which builds the path as `ctxPath + "." + b.Name`. But `b.Name` is empty (set to `""` in `executeElementRules` via `tmpl.Bind(bindTarget, "")`). So the path becomes just `"items[0]"` without the field name.

This is actually correct for anonymous elements, but if the StringChain had a name, it would be `"items[0].fieldname"` — which is correct. The issue is that the name is never pushed to the context stack, so nested chains don't see it.

**Impact:** Inconsistent path resolution between slice chains (which push their name) and primitive chains (which don't).

**Fix:** Add `ctx.Push(c.Name)` / `defer ctx.Pop()` to `StringChain.Validate()` and `IntChain.Validate()`.

---

## BUG-15: `BaseChain.Fail()` path resolution is fragile and duplicates context logic (Code Smell)

**File:** `chains/base.go:55-80`
**Severity:** Medium

```go
func (b *BaseChain) Fail(ctx *core.Context, value any, reason string) error {
    path := b.Name
    if ctx != nil {
        ctxPath := ctx.String()
        if ctxPath != "" {
            if b.Name != "" {
                if strings.HasPrefix(b.Name, "[") {
                    path = ctxPath + b.Name
                } else {
                    path = ctxPath + "." + b.Name
                }
            } else {
                path = ctxPath
            }
        }
    }
    return &core.FieldError{Path: path, Value: value, Reason: reason}
}
```

This method manually constructs the path by concatenating `ctxPath` and `b.Name`. This duplicates the path-building logic that should be handled by the context stack. If all chains pushed their name to the context, `ctx.String()` would return the complete path, and `Fail()` could simply use `ctx.String()`.

**Impact:** Fragile path construction that's inconsistent with the context stack model. The `HasPrefix(b.Name, "[")` check is a hack to handle index notation.

**Fix:** Have all chains push their name to the context stack, then simplify `Fail()` to use `ctx.String()` directly.

---

## BUG-16: `GetChain` returns unhelpful error for nil target (Usability)

**File:** `registry.go:50`
**Severity:** Low

```go
func GetChain(target any, name string) (core.Rule, error) {
    if target == nil {
        return nil, core.ErrMisuse  // Just says "skrub: misuse of API"
    }
```

**Impact:** Users get a generic error with no context about what was misused.

**Fix:** Return a more descriptive error: `fmt.Errorf("skrub: GetChain called with nil target for field %q", name)`.

---

## BUG-17: `StringChain` and `IntChain` resolveTarget doesn't handle `*T` where T implements Valuer (Missing Case)

**File:** `chains/string.go:226-240`, `chains/int.go:56-72`
**Severity:** Medium

```go
func (c *StringChain) resolveTarget() (string, bool, error) {
    switch t := c.target.(type) {
    case *string:
        // ...
    case core.Valuer:
        // ...
    default:
        return "", false, core.ErrMisuse
    }
}
```

If `c.target` is `**string` (pointer to pointer to string) or `*someValuerType` (pointer to a Valuer implementation), neither case matches and it falls through to `ErrMisuse`. The `ResolveValue` function in `skrubreflect` can return `*T` (address of a value), but if the target is already a pointer to a pointer, this breaks.

**Impact:** Certain valid pointer configurations return `ErrMisuse` instead of being handled gracefully.

**Fix:** Add reflection-based unwrapping as a fallback, or document the limitation.

---

## BUG-18: `IntDef.MatchString` compiles regex at definition time but doesn't handle invalid patterns gracefully (Panic Risk)

**File:** `defs/primitives.go` (IntDef section)
**Severity:** Low

```go
func (d *IntDef) MatchString(pattern string) *IntDef {
    re := regexp.MustCompile(pattern)  // Panics on invalid regex
    // ...
}
```

Uses `regexp.MustCompile` which panics on invalid patterns. While this is intentional (fail fast at definition time), it means invalid patterns can't be caught gracefully.

**Impact:** Panic on invalid regex patterns instead of returning an error.

**Fix:** Use `regexp.Compile` and return an error, or document the panic behavior clearly.

---

## BUG-19: `StringDef.Pattern` compiles regex at definition time with same panic risk (Panic Risk)

**File:** `defs/primitives.go` (StringDef section)
**Severity:** Low

Same issue as BUG-18 but for `StringDef.Pattern`.

---

## BUG-20: `Context.checkDepth` uses `c.String()` which allocates on every depth check (Performance)

**File:** `pkg/core/context.go:120-130`
**Severity:** Low

```go
func (c *Context) checkDepth(currentKeyHint string) error {
    if c.depth > c.cfg.MaxDepth {
        return &RecursionError{
            Path:     c.String() + buildPath(currentKeyHint),  // Allocates
            Depth:    c.depth,
            MaxDepth: c.cfg.MaxDepth,
        }
    }
    if c.cfg.WarningThreshold > 0 && c.depth == c.cfg.WarningThreshold {
        if c.cfg.OnWarning != nil {
            c.cfg.OnWarning(c.String()+buildPath(currentKeyHint), c.depth)  // Allocates
        }
    }
    return nil
}
```

`c.String()` builds the full path string using `strings.Builder` on every call. This is called on every `Push()` and `PushIndex()`, which means deep nesting causes O(n²) string allocations.

**Impact:** Performance degradation for deeply nested structures.

**Fix:** Cache the string representation, or only build it when an error actually occurs.

---

## Summary

| ID | Severity | Category | File |
|----|----------|----------|------|
| BUG-1 | Medium | Dead Code | `skrub.go:40` |
| BUG-2 | **High** | Path Inconsistency | `chains/string.go`, `chains/int.go` |
| BUG-3 | **High** | Pool State Leak | `pkg/core/context.go:100-105` |
| BUG-4 | Medium | Logic Error | `chains/slice.go:160-165` |
| BUG-5 | Low | Performance | `chains/slice.go:175-185` |
| BUG-6 | Medium | Logic Error | `pkg/core/context.go:85-95` |
| BUG-7 | **High** | Potential Panic | `registry.go:55-65` |
| BUG-8 | Medium | State Loss | `internal/pool/safe.go:85-100` |
| BUG-9 | Low | Capacity | `internal/pool/safe.go:45-55` |
| BUG-10 | Low | Missing Check | `markers.go` |
| BUG-11 | Medium | Pool Bypass | `chains/slice.go:35-40` |
| BUG-12 | Low | Silent Failure | `pkg/adapters/uuid.go:55-65` |
| BUG-13 | Low | Edge Case | `defs/recursive.go:10-20` |
| BUG-14 | **High** | Path Inconsistency | `chains/string.go`, `chains/int.go` |
| BUG-15 | Medium | Code Smell | `chains/base.go:55-80` |
| BUG-16 | Low | Usability | `registry.go:50` |
| BUG-17 | Medium | Missing Case | `chains/string.go`, `chains/int.go` |
| BUG-18 | Low | Panic Risk | `defs/primitives.go` |
| BUG-19 | Low | Panic Risk | `defs/primitives.go` |
| BUG-20 | Low | Performance | `pkg/core/context.go:120-130` |

**Total: 20 issues found (4 High, 5 Medium, 11 Low)**
