# webtyp/image
<img src="docs/img/badges.svg">

Todo lo relacionado con imágenes para WebTyp: builders HTML + pipeline de optimización WebP + compresión en cliente.

## Cuatro capas

1. **Builders** (`webtyp.com/image`) — `Img`, `Picture`, `Source`, construcción de elementos HTML. Compila para WASM y backend. Sin etiquetas de construcción (build tags).
2. **Pipeline** (`webtyp.com/image/min`) — `Handler`, `Config`, procesamiento WebP. Solo para backend (`//go:build !wasm`).
3. **Navegador** (`webtyp.com/image/browser`) — `Compress`, `CompressToFit`, compresión y redimensionado en el cliente antes de subir. Solo para WASM (`//go:build wasm`).
4. **Favicon** (`webtyp.com/image/favicon`) — `//go:build !wasm` — de un logo cuadrado al juego de iconos.

## Favicon (Backend)
```go
import "webtyp.com/image/favicon"

files, _ := favicon.Derive(favicon.Source{Raster: pngBytes, SVG: svgBytes})
for _, f := range files {
    os.WriteFile(filepath.Join(outDir, f.Name), f.Content, 0644)
}
```

> El nombre `image` sombrea el paquete estándar de Go **a propósito**: el stdlib `image` es demasiado pesado para TinyGo. Internamente, el pipeline aliasea el stdlib como `stdimage`.

## Compresión en el cliente (Frontend/WASM)
```go
import "webtyp.com/image/browser"

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
    import . "webtyp.com/image"

    // Variantes generadas: S=480px, M=1024px, L=1600px (calidad JPEG 62 por defecto).
    // sizes importa: sin el, el navegador asume 100vw y baja de mas en cualquier
    // imagen que no ocupe el ancho completo.

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
    import "webtyp.com/image"

    func RenderImages() []image.Asset {
        return []image.Asset{
            {Path: "img/hero.png", Variants: image.AllVariants, Alt: "Hero"},
        }
    }
> El nombre de archivo **debe** ser `image.go`. El pipeline lo detecta automáticamente.

## Pipeline (Backend)
    import "webtyp.com/image/min"

    handler := min.New(&min.Config{RootDir: ".", OutputDir: "web/public/img", Quality: 80})
    handler.LoadImages()

> La detección de módulos se delega a [webtyp/modfind](https://github.com/webtyp/modfind).

## Related Packages
- [webtyp/dom](https://github.com/webtyp/dom) — Element type
- [webtyp/html](https://github.com/webtyp/html) — HTML builders
- [webtyp/svg](https://github.com/webtyp/svg) — SVG builders + sprite
