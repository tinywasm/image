# PLAN — Migrate image/min module discovery to `tinywasm/modfind`

> `image/min` runs its own `go list -m -json all` in `listModulesReal`
> ([loader.go:15](../loader.go#L15)) to get module dirs, then extracts images from each. That exact
> block is byte-identical to copies in `ssr` and `imagemin`. This plan replaces it with the shared
> **`tinywasm/modfind`** finder so `go list` runs **once** per dev session, cached and shared.
>
> **Self-contained, single-module plan** (`tinywasm/image/min`, package `min`). Prerequisite:
> `tinywasm/modfind` published. Its contract (inlined so this plan needs no external file):
> `modfind.New() *Finder`; `(*Finder).Dirs(rootDir) ([]string, error)` runs `go list -m -json all`
> once and caches; `(*Finder).Seed(rootDir, []Module)` pre-loads the cache for tests;
> `(*Finder).Discover` returns `[]Module{Path, Dir, …; Writable() bool}`.
>
> **This is a clean breaking change.** The duplicated `go list` AND the `listModulesFn` /
> `InitDefaultLoader` injection seam are **deleted** — no deprecated fallback. webp output unchanged.
> (The sibling repo `tinywasm/imagemin` carries an identical `listModulesReal`; same migration via its
> own plan.)

---

## 1. Development Rules (constraints copied for execution context)

- **REUSE `modfind` — do NOT reimplement discovery.** The `go list -m -json all` logic already exists,
  consolidated and tested, in the **published** `github.com/tinywasm/modfind`. This task is a
  **deletion + wiring** correction, **not** a from-scratch implementation. Do not write any `go list`,
  `os/exec`, or module-walk code here — import modfind and call `Dirs`. The only copy of this
  algorithm in the ecosystem must be modfind's.
- **Same webp output, single discovery source.** `LoadImages` must process the same module dirs and
  produce the same webp output. Only the module-list *source* changes.
- **Delete the old seam — no deprecated code.** Remove `listModulesFn`, `SetListModulesFn`, and
  `InitDefaultLoader` (the inline `go list` wiring) entirely. Replace with `SetFinder` + lazy init.
  Tests inject a seeded `*modfind.Finder`, not a `func`.
- **`//go:build !wasm`.** `loader.go` already has it; keep it. `modfind` is `!wasm` too.
- **Minimal dep.** `modfind` adds only stdlib + `tinywasm/fmt`. No heavy transitive deps.
- **Documentation first.**

---

## 2. Problem

`listModulesReal` ([loader.go:15-40](../loader.go#L15)) is the third copy of the canonical `go list -m
-json all` → `[]dir` loop (ssr + imagemin have the others). Three independent `go list` runs at
startup for the same project; no shared cache. Contradicts the light/fast dev-loop goal.

---

## 3. Decision

Delete `listModulesReal` + the `listModulesFn`/`InitDefaultLoader` seam; source dirs from a
`*modfind.Finder`:

```go
// handler.go
func (h *Handler) SetFinder(f *modfind.Finder) { h.finder = f } // app injects the shared one

// loader.go — LoadImages sources dirs directly from the finder
func (h *Handler) moduleDirs() ([]string, error) {
    if h.finder == nil {
        h.finder = modfind.New() // lazy: standalone use still works
    }
    return h.finder.Dirs(h.config.RootDir)
}
```

- `Handler` gains `finder *modfind.Finder` + `SetFinder`. Remove `listModulesFn`, `SetListModulesFn`,
  `InitDefaultLoader`, and `listModulesReal`.
- `LoadImages` ([loader.go:47](../loader.go#L47)) calls `h.moduleDirs()` instead of `h.listModulesFn(
  h.config.RootDir)`; the per-dir extraction loop ([loader.go:62-76](../loader.go#L62)) is untouched.
- **Tests** that called `SetListModulesFn(fakeFn)` now seed:
  `f := modfind.New(); f.Seed(root, mods); h.SetFinder(f)`.

---

## 4. Implementation Steps

### Step 1 — Bump modfind
`go get github.com/tinywasm/modfind@vX`.

### Step 2 — Finder field + injector; remove old seam
[handler.go](../handler.go): add `finder *modfind.Finder` to `Handler` + `SetFinder`. **Delete**
`listModulesFn`, `SetListModulesFn`, `InitDefaultLoader`.

### Step 3 — Replace the loader
[loader.go](../loader.go): **delete** `listModulesReal` (the inline `go list`); add `moduleDirs()`
(§3) and call it from `LoadImages`. Drop now-unused imports (`bytes`, `encoding/json`, `os/exec`).

### Step 4 — Migrate tests + update app call sites
Tests that used `SetListModulesFn` → seed a finder (§3). **Note for app:** `app` currently calls
`h.ImageHandler.InitDefaultLoader()` and `SetListModulesFn`; those calls are removed in app's plan and
replaced by `SetFinder(shared)`.

### Step 5 — Documentation
[README.md](../README.md) (or docs): note module discovery delegated to `modfind`. Link it.

---

## 5. Edge Cases

- **Seeded finder (tests)** → `Dirs` returns seeded dirs; modfind `go list` not called.
- **Replace module dir** → treated like any dir; images extracted in place. No special handling.
- **`go list` fails in modfind** → `Dirs` returns error; `LoadImages` already log-and-continues
  ([loader.go:57](../loader.go#L57)).
- **Shared finder** → single `go list`; image hits cache after ssr/ormc warm it (or vice-versa).

---

## 6. Test Strategy

`gotest`. Existing image-processing tests are the regression guard. Add:

| # | Case | Assert |
|---|------|--------|
| I1 | seeded finder injected | modfind `go list` not invoked; seeded dirs used |
| I2 | no injected finder | lazy modfind discovers real module dirs |
| I3 | injected shared finder | dirs match `finder.Dirs` |
| I4 | webp output | unchanged vs pre-refactor (regression) |

---

## 7. Out of Scope

- `modfind` implementation — its own plan.
- `imagemin` sibling migration — its own plan (identical pattern).
- Any schema/ssr concern — different handlers.
