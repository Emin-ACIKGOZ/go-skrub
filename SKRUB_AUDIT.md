# SKRUB_AUDIT.md — Audit Summary of skrub.go

1. **Package & Imports**: skrub.go is the main entry point for the skrub validation library, importing pool for context reuse and core for validation logic.

2. **Global Pool**: A globalCtxPool (SafePool) with capacity 128 manages reusable core.Context instances, blocking when empty to provide backpressure.

3. **Validate()**: A convenience function that calls ValidateWithConfig with default core.Config{}, accepting a target value and variadic Rule arguments.

4. **ValidateWithConfig()**: The primary entry point - acquires a context from the pool, applies custom config (MaxDepth, WarningThreshold, OnWarning), executes all rules, and defers returning the context to the pool.

5. **Error Handling**: If pool acquisition fails, the error is returned immediately; if any rule's Validate() returns an error, execution short-circuits and returns that error.
