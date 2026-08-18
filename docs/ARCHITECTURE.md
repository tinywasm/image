# Imagemin Architecture

`image` is a specialized image processing library for Go SSR (Server-Side Rendering) modules within the `tinywasm` ecosystem. Its primary goal is to automate the generation of responsive WebP images from declarations in module source code.

## Package Structure

The package is split into two parts:
1. **`github.com/tinywasm/image` (Root)**: Contains types and HTML builders. It has no build tags and is safe to import in WASM.
2. **`github.com/tinywasm/image/min` (Subpackage)**: Contains the processing pipeline (Handler, Converter, Extractor). It is only imported by the backend.

- **`types.go`**: Contains core types like `Asset` and `Variant`.
- **`builders.go`**: HTML builders like `Img()`, `Picture()`, and `Source()`.
- **`min/extract.go`**: Uses the `go/ast` and `go/parser` packages to analyze Go source code. It extracts the `RenderImages` function from `image.go` files.
- **`min/convert.go`**: Handles image transformation using `imaging` and `nativewebp`.
- **`min/loader.go`**: Orchestrates discovery via `go list`.
- **`min/cache.go`**: Implements mtime-based change detection.
- **`min/artifacts.go`**: Exposes processed images as in-memory `Artifact` values (`Handler.Artifacts()`) and resolves the default cache location (`DefaultCacheDir`).

## Core Concepts

### Asset Declarations
Instead of manually managing images, modules declare their requirements in a standard `image.go` file. By implementing `RenderImages() []image.Asset`, a module tells the system which images it needs and what responsive variants (Small, Medium, Large) should be generated.

### Stat-Based Caching (mtime)
To ensure fast development cycles, `image` avoids reprocessing images that haven't changed. Instead of maintaining a separate database or JSON file with image hashes, it uses the **Modification Time (mtime)** provided by the operating system's file system.

**The logic is simple and efficient:**
1. For every declared asset, the system checks if the output WebP files already exist.
2. If they exist, it compares the `mtime` of the source file (e.g., `.png`) with the `mtime` of the output files (e.g., `.S.webp`).
3. If the source file is newer than any of its outputs, the image is reprocessed.
4. If the outputs are newer or equal, the system skips them.

### Serving From Memory: `Artifacts()`

`Config.OutputDir` is where `LoadImages` writes converted files, but a
development server that serves a site from memory has no other way to obtain
that content — without `Artifacts()`, the consumer would have to keep a real
output directory inside the user's project just to have something to serve.

`Handler.Artifacts()` returns one `Artifact{Path, Mediatype, Content}` per file
that `LoadImages`/`ReloadModule` last produced, read back from `OutputDir`.
`Path` is `img/<name>` and doubles as the URL; `Mediatype` is derived from the
file extension (`image/jpeg`, `image/webp`, `image/svg+xml`). A source whose
output can't be read is skipped silently — the conversion failure was already
logged.

### Cache vs. Output: `DefaultCacheDir`

`OutputDir` plays two roles that this design keeps separate: it is the
**cache** that avoids re-encoding unchanged sources between runs, and it can
also be the **output** a site publishes. A consumer that only needs the former
(because it serves via `Artifacts()`) should not point `OutputDir` inside the
user's project. `DefaultCacheDir(rootDir)` returns a per-project directory
under the OS user cache (`os.UserCacheDir()`), keyed by a truncated SHA-256 of
`rootDir`, so the cache never lives inside — or gets confused with — a
project's publishable output.

## SEO Considerations

- **WebP Format**: Modern compression for faster page loads.
- **Responsive Variants**: Serve the smallest appropriate image for the user's device.
- **Alt Text**: Declared in `image.go` for use in HTML generation.

## Workflow

1. **Discovery**: `image/min` finds all modules via `go list`.
2. **Extraction**: For each module, it parses `image.go` to find image declarations.
3. **Validation**: It checks `mtime` to see which images actually need processing.
4. **Processing**: It resizes and encodes the necessary variants.
5. **Cleanup**: It removes any WebP files that are no longer declared by any module.
