# Imagemin Architecture

`image` is a specialized image processing library for Go SSR (Server-Side Rendering) modules within the `tinywasm` ecosystem. Its primary goal is to automate the generation of responsive WebP images from declarations in module source code and to compress client-side uploaded images before sending them over the network.

## Package Structure

The package is split into three parts:

| Layer | Package | Build | When Executed |
|---|---|---|---|
| **Builders** | `github.com/tinywasm/image` | Tagless | Render, both server & client |
| **Pipeline** | `github.com/tinywasm/image/min` | `//go:build !wasm` | Build time, on server |
| **Browser** | `github.com/tinywasm/image/browser` | `//go:build wasm` | Client-side pre-upload |

- **`types.go`**: Contains core types like `Asset` and `Variant`, as well as suffix conventions (`Variant.Suffix()`) and pixel widths (`Variant.Width()`).
- **`builders.go`**: HTML builders like `Img()`, `Responsive()`, `Picture()`, and `Source()`.
- **`browser/compress.go`**: Client-side `Compress` and `CompressToFit` utilizing browser `OffscreenCanvas`.
- **`browser/js.go`**: Constant definitions for JS global/property names.
- **`browser/errors.go`**: Sentinel errors like `ErrUnsupported`.
- **`min/extract.go`**: Uses the `go/ast` and `go/parser` packages to analyze Go source code. It extracts the `RenderImages` function from `image.go` files.
- **`min/convert.go`**: Handles image transformation using `imaging` and `nativewebp`.
- **`min/loader.go`**: Orchestrates discovery via `go list`.
- **`min/cache.go`**: Implements mtime-based change detection.
- **`min/artifacts.go`**: Exposes processed images as in-memory `Artifact` values (`Handler.Artifacts()`) and resolves the default cache location (`DefaultCacheDir`).

## Core Concepts

### Browser Image Compression (`OffscreenCanvas`)
The client layer (`github.com/tinywasm/image/browser`) decompresses, resizes, and re-encodes user-selected files using the browser's native capabilities before uploading.

Using `OffscreenCanvas` and `convertToBlob()` guarantees all asynchronous operations return `Promise` objects, seamlessly integrating with `github.com/tinywasm/await`. This avoids DOM manipulation or callback-based APIs (`canvas.toBlob`), keeping client-side operations fast and decoupled from document elements.

### Responsive Images (`<img srcset>`) vs `<picture>`

Para **resolution switching** (servir la misma imagen en distintas resoluciones), el elemento HTML adecuado es `<img srcset sizes>`. Esto permite al navegador seleccionar la variante óptima considerando el ancho de pantalla, la densidad de píxeles (DPR) y las condiciones de red. `<picture>` se reserva para *art direction* (recortes o composiciones diferentes por tamaño de pantalla) o formatos alternativos.

La convención de sufijos (`.S`, `.M`, `.L`) y anchos (`WidthS = 640`, `WidthM = 1024`, `WidthL = 1920`) reside en `types.go` en el paquete raíz sin build tags. Esta única definición compartida permite que tanto el pipeline de optimización en backend (`min/`) como los renderizadores de HTML en frontend (WASM) compartan la misma convención sin duplicar cadenas ni depender de código específico de backend.

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
