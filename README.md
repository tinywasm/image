# tinywasm/image
<img src="docs/img/badges.svg">

Todo lo relacionado con imágenes para TinyWasm: builders HTML + pipeline de optimización WebP.

## Dos capas

1. **Builders** (`github.com/tinywasm/image`) — `Img`, `Picture`, `Source`, construcción de elementos HTML. Compila para WASM y backend. Sin etiquetas de construcción (build tags).
2. **Pipeline** (`github.com/tinywasm/image/min`) — `Handler`, `Config`, procesamiento WebP. Solo para backend.

> El nombre `image` sombrea el paquete estándar de Go **a propósito**: el stdlib `image` es demasiado pesado para TinyGo. Internamente, el pipeline aliasea el stdlib como `stdimage`.

## Render (Frontend/WASM)
    import . "github.com/tinywasm/image"

    func (c *Hero) Render() *dom.Element {
        return Img("/img/hero.M.webp", "Hero").Lazy().Size(1024, 512).AsElement()
    }

## Declaración para procesamiento (image.go, //go:build !wasm)
    package herosection
    import "github.com/tinywasm/image"

    func RenderImages() []image.Asset {
        return []image.Asset{
            {Path: "img/hero.png", Variants: image.AllVariants, Alt: "Hero"},
        }
    }
> El nombre de archivo **debe** ser `image.go`. El pipeline lo detecta automáticamente.

## Pipeline (Backend)
    import "github.com/tinywasm/image/min"

    handler := min.New(&min.Config{RootDir: ".", OutputDir: "web/public/img", Quality: 80})
    handler.InitDefaultLoader()
    handler.LoadImages()

## Related Packages
- [tinywasm/dom](https://github.com/tinywasm/dom) — Element type
- [tinywasm/html](https://github.com/tinywasm/html) — HTML builders
- [tinywasm/svg](https://github.com/tinywasm/svg) — SVG builders + sprite
