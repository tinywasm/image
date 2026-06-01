# tinywasm/image
<img src="docs/img/badges.svg">

Todo lo relacionado con imágenes para TinyWasm: builders HTML + pipeline de optimización WebP.

## Dos capas

1. **Builders** (`Img`, `Picture`, `Source`) — construcción de elementos HTML. Compila para wasm y backend.
2. **Pipeline** (`Handler`, `Asset`, `Variant`) — declara imágenes en `ssr.go`, genera variantes WebP S/M/L. Solo backend (`//go:build !wasm`).

> El nombre `image` sombrea el paquete estándar de Go **a propósito**: el stdlib `image` es demasiado pesado para TinyGo. Internamente, el pipeline aliasea el stdlib como `stdimage`.

## Render
    import . "github.com/tinywasm/image"

    func (c *Hero) Render() *dom.Element {
        return Img("/img/hero.M.webp", "Hero").Lazy().Size(1024, 512).AsElement()
    }

## Declaración para procesamiento (ssr.go, //go:build !wasm)
    func RenderImages() []image.Asset {
        return []image.Asset{
            {Path: "img/hero.png", Variants: image.AllVariants, Alt: "Hero"},
        }
    }

## Pipeline
    handler := image.New(&image.Config{RootDir: ".", OutputDir: "web/public/img", Quality: 80})
    handler.InitDefaultLoader()
    handler.LoadImages()

## Related Packages
- [tinywasm/dom](https://github.com/tinywasm/dom) — Element type
- [tinywasm/html](https://github.com/tinywasm/html) — HTML builders
- [tinywasm/svg](https://github.com/tinywasm/svg) — SVG builders + sprite
