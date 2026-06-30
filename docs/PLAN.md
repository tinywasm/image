# tinywasm/image — Plan: Migrate to Typed Builders

> **Triggered by:** `tinywasm/dom` v0.11 removed `.Add(...any)` and the `Picture(...any)` pattern.
> **Scope:** `builders.go` only — one function to fix.

## Prerequisites

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

## Problem

`Picture(children ...any)` uses `.Add(children...)` which no longer exists in `dom.Element`:

```go
// builders.go:59 — BROKEN
func Picture(children ...any) *dom.Element {
    return dom.NewElement("picture").Add(children...)
}
```

## Change — `Picture` becomes no-arg

```go
func Picture() *dom.Element {
    return dom.NewElement("picture")
}
```

Callers compose via `.Child(...)` with typed `*dom.Element` children (`Source(...)`, `Img(...).AsElement()`):

```go
// Before (any):
Picture(Source(srcset, "image/webp"), Img(src, alt).AsElement())

// After (typed):
Picture().Child(Source(srcset, "image/webp"), Img(src, alt).AsElement())
```

## Consumers

Search all consumers of `Picture(` and update to `Picture().Child(...)`.

Known: none in this module's own tests — check `tinywasm/components`, `tinywasm/layout`, `tinywasm/user`.

## Tests

- `gotest` must pass with no `vet` errors.
- Add a stdlib test: `Picture().Child(Source(srcset, "image/webp")).String()` contains `<picture>` and `<source srcset=...>`.

## Done When

- No `Picture(...any)` constructor remains.
- `go vet ./...` passes.
- `gotest` passes.
