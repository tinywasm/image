# tinywasm/image
<img src="docs/img/badges.svg">

Todo lo relacionado con imágenes para TinyWasm: builders HTML + pipeline de optimización WebP + compresión en cliente.

## Tres capas

1. **Builders** (`github.com/tinywasm/image`) — `Img`, `Picture`, `Source`, construcción de elementos HTML. Compila para WASM y backend. Sin etiquetas de construcción (build tags).
2. **Pipeline** (`github.com/tinywasm/image/min`) — `Handler`, `Config`, procesamiento WebP. Solo para backend (`//go:build !wasm`).
3. **Navegador** (`github.com/tinywasm/image/browser`) — `Compress`, `CompressToFit`, compresión y redimensionado en el cliente antes de subir. Solo para WASM (`//go:build wasm`).

> El nombre `image` sombrea el paquete estándar de Go **a propósito**: el stdlib `image` es demasiado pesado para TinyGo. Internamente, el pipeline aliasea el stdlib como `stdimage`.

## Compresión en el cliente (Frontend/WASM)
```go
import "github.com/tinywasm/image/browser"

// En el handler del evento 'change' de un <input type="file">:
file := input.Get("files").Index(0)

// Comprimir a 1920px maxEdge, WebP, calidad 0.85
res, err := browser.Compress(file, browser.Config{
    MaxEdge: 1920,
    Quality: 0.85,
    Type:    "image/webp",
})
if err != nil {
    // Manejar error (p.ej. browser.ErrUnsupported)
}

// res.Data contiene los []byte listos para subir por HTTP
```

## Render (Frontend/WASM)
    import . "github.com/tinywasm/image"

    // Imagen responsiva con srcset y sizes (100vw por defecto):
    func (c *Card) Render() *dom.Element {
        return Responsive("/img/foto.jpg", "Fachada").
            Sizes("(max-width: 600px) 100vw, 33vw").
            Lazy().
            AsElement()
    }

    // Imagen simple sin srcset:
    func (c *Logo) Render() *dom.Element {
        return Img("/img/logo.png", "Logo").Size(200, 50).AsElement()
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
    handler.LoadImages()

> La detección de módulos se delega a [tinywasm/modfind](https://github.com/tinywasm/modfind).

## Related Packages
- [tinywasm/dom](https://github.com/tinywasm/dom) — Element type
- [tinywasm/html](https://github.com/tinywasm/html) — HTML builders
- [tinywasm/svg](https://github.com/tinywasm/svg) — SVG builders + sprite
