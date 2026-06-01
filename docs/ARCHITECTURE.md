# Imagemin Architecture

`image` is a specialized image processing library for Go SSR (Server-Side Rendering) modules within the `tinywasm` ecosystem. Its primary goal is to automate the generation of responsive WebP images from declarations in module source code.

## Core Concepts

### Asset Declarations
Instead of manually managing images, modules declare their requirements in a standard `ssr.go` file. By implementing `RenderImages() []image.Asset`, a module tells the system which images it needs and what responsive variants (Small, Medium, Large) should be generated.

### Stat-Based Caching (mtime)
To ensure fast development cycles, `image` avoids reprocessing images that haven't changed. Instead of maintaining a separate database or JSON file with image hashes, it uses the **Modification Time (mtime)** provided by the operating system's file system.

**The logic is simple and efficient:**
1. For every declared asset, the system checks if the output WebP files already exist.
2. If they exist, it compares the `mtime` of the source file (e.g., `.png`) with the `mtime` of the output files (e.g., `.S.webp`).
3. If the source file is newer than any of its outputs, the image is reprocessed.
4. If the outputs are newer or equal, the system skips them.

This approach is extremely fast because reading file metadata takes microseconds compared to reading and hashing megabytes of image data. It also keeps the project clean by not generating extra hidden state files.

## Package Structure

- **`types.go`**: Contains the public API and core types. It has no build tags, ensuring it can be imported by any module (even those compiling to WASM) without bringing in heavy dependencies.
- **`extract.go`**: Uses the `go/ast` and `go/parser` packages to analyze Go source code. It extracts the `RenderImages` function body without actually executing the code, making it safe and fast.
- **`convert.go`**: Handles the heavy lifting of image transformation.
    - **Resizing**: Uses `imaging` with the Lanczos filter for high-quality downscaling.
    - **Encoding**: Uses `nativewebp` for WebP generation.
    - **Safety**: Never upscales images; if a source image is smaller than the target variant width, the resize step is skipped to preserve quality.
- **`loader.go`**: Orchestrates discovery.
    - Uses `go list -m -json all` to find all modules in the project.
    - Iterates through modules to trigger extraction and conversion.
    - Implements a global **Orphan Cleanup** strategy: during initial load, any `.webp` file in the output directory that doesn't correspond to a currently declared asset is deleted.

## SEO Considerations

- **WebP Format**: Provides modern compression for faster page loads, a key factor in Core Web Vitals.
- **Responsive Variants**: Allows the frontend to serve the smallest appropriate image for the user's device (Mobile, Tablet, Desktop) using `<picture>` or `srcset`.
- **Descriptive Naming**: Images are named after their base name and variant (e.g., `hero.M.webp`), which is search-engine friendly.
- **Alt Text**: Alt text is declared alongside the image in `ssr.go`. While not embedded in the WebP file (due to encoder limitations), it is available during the extraction phase for use in HTML generation.

## Workflow

1. **Discovery**: `image` finds all modules via `go list`.
2. **Extraction**: For each module, it parses `ssr.go` to find image declarations.
3. **Validation**: It checks `mtime` to see which images actually need processing.
4. **Processing**: It resizes and encodes the necessary variants.
5. **Cleanup**: It removes any WebP files that are no longer declared by any module.
