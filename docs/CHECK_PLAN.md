# PLAN: tinywasm/image — Builders + Pipeline (fusión de imagemin)

## Repositorio
`github.com/tinywasm/image` — path local: `tinywasm/image/`

Estado actual:
- go.mod: `module github.com/cdvelop/image` (a renombrar)
- `image.go`: stub `type Image struct{}` + `func New() *Image` (a eliminar)

## Dependencias de ejecución
```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

## Prerequisito
Ejecutar **después** de que `tinywasm/dom` haya publicado su versión con `Img()` eliminado de `dom/element.go` y `NewElement()` / `NoCloseTag()` disponibles.

---

## Objetivo y decisión arquitectónica

**Una sola librería para TODO lo de imágenes.** Se fusiona `tinywasm/imagemin` dentro de `tinywasm/image` y se archiva imagemin.

`tinywasm/image` provee dos capas separadas por build tags:

| Capa | Archivos | Build tag | Depende de |
|------|----------|-----------|------------|
| **Builders HTML** | `builders.go` | sin tag (wasm + backend) | `tinywasm/dom` |
| **Declaración** | `types.go` | sin tag | — |
| **Pipeline (procesamiento)** | `convert.go`, `extract.go`, `cache.go`, `loader.go`, `handler.go` | `//go:build !wasm` | `imaging`, `nativewebp` |

**Por qué funciona en WASM:** al compilar para wasm, los archivos `//go:build !wasm` se excluyen → `imaging`/`nativewebp` nunca entran al binario, aunque sigan en `go.mod`. Go solo linkea código alcanzable.

**Colisión con stdlib `image` — INTENCIONAL:** el paquete se llama `image` a propósito para sombrear el `image` estándar (pesado, inviable en TinyGo). Así un componente no puede importar por error el stdlib. Internamente, los archivos del pipeline que necesitan el stdlib lo aliasan: `import stdimage "image"`.

---

## go.mod

```
module github.com/tinywasm/image

go 1.25

require (
    github.com/tinywasm/dom v<nueva-version>
    github.com/HugoSmits86/nativewebp v1.2.1
    github.com/disintegration/imaging v1.6.2
)

require golang.org/x/image v0.24.0 // indirect
```

> `imaging`/`nativewebp` son deps de build (solo `!wasm`). No afectan el binario WASM.

---

## Paso 1: Eliminar el stub

```bash
rm tinywasm/image/image.go
```

Contiene `type Image struct{}` + `func New() *Image` — se reemplaza por builders y el Handler del pipeline.

---

## Paso 2: Mover el pipeline de imagemin

Copiar estos archivos desde `tinywasm/imagemin/` a `tinywasm/image/`, cambiando `package imagemin` → `package image`:

- `types.go`     → `Asset`, `Variant`, constantes `VariantS/M/L`, `AllVariants` (SIN build tag, los necesita el render)
- `convert.go`   → `//go:build !wasm` — procesamiento WebP
- `extract.go`   → `//go:build !wasm` — AST de `RenderImages() []Asset`
- `cache.go`     → `//go:build !wasm` — `IsUpToDate` mtime
- `loader.go`    → `//go:build !wasm` — `LoadImages`, `ReloadModule`, `go list`
- `handler.go`   → `//go:build !wasm` — `Handler`, `New(*Config)`, devwatch (era `imagemin.go`)

### Cambio obligatorio en `convert.go` — aliasar stdlib

**Buscar:**
```go
import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	"github.com/disintegration/imaging"
)
```

**Reemplazar con:**
```go
import (
	"fmt"
	stdimage "image"
	"os"
	"path/filepath"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	"github.com/disintegration/imaging"
)
```

**Y la única referencia al tipo (línea ~37):**
```go
var processedImg image.Image   →   var processedImg stdimage.Image
```

> Revisar el resto de archivos del pipeline por si alguno más referencia el stdlib `image`. Solo `convert.go` lo hace actualmente.

---

## Paso 3: Crear `builders.go` (sin build tag)

```go
package image

import (
    "github.com/tinywasm/dom"
    "github.com/tinywasm/fmt"
)

// ImgElement wraps *dom.Element to provide a fluent image-specific API.
type ImgElement struct {
    el *dom.Element
}

// Img builds an <img> element with src and alt.
func Img(src, alt string) *ImgElement {
    el := dom.NewElement("img").
        Attr("src", src).
        Attr("alt", alt).
        NoCloseTag()
    return &ImgElement{el: el}
}

// Lazy sets loading="lazy".
func (i *ImgElement) Lazy() *ImgElement {
    i.el.Attr("loading", "lazy")
    return i
}

// Size sets width and height (reduces CLS).
func (i *ImgElement) Size(w, h int) *ImgElement {
    i.el.Attr("width", fmt.Sprint(w))
    i.el.Attr("height", fmt.Sprint(h))
    return i
}

// Class adds CSS classes.
func (i *ImgElement) Class(classes ...string) *ImgElement {
    i.el.Class(classes...)
    return i
}

// Attr sets an arbitrary attribute.
func (i *ImgElement) Attr(key, val string) *ImgElement {
    i.el.Attr(key, val)
    return i
}

// AsElement returns the underlying *dom.Element for embedding in Render() trees.
func (i *ImgElement) AsElement() *dom.Element {
    return i.el
}

// String serializes the image element (satisfies dom.Component).
func (i *ImgElement) String() string {
    return i.el.String()
}

// Picture builds a <picture> element for responsive images.
func Picture(children ...any) *dom.Element {
    return dom.NewElement("picture").Add(children...)
}

// Source builds a <source> for use inside <picture>.
// mediaOrType: media query "(max-width: 600px)" or MIME type "image/webp".
func Source(srcset, mediaOrType string) *dom.Element {
    el := dom.NewElement("source").Attr("srcset", srcset).NoCloseTag()
    if len(mediaOrType) > 0 {
        if mediaOrType[0] == '(' {
            el.Attr("media", mediaOrType)
        } else {
            el.Attr("type", mediaOrType)
        }
    }
    return el
}
```

---

## Paso 4: Mover tests

Mover desde `tinywasm/imagemin/tests/` a `tinywasm/image/tests/`:
- `convert_test.go`, `extract_test.go`, `handler_test.go`, `loader_test.go`, `mtime_test.go`, `concurrency_test.go`, `types_test.go`, `setup_test.go`
- `testdata/` (gatos.M.jpeg, gopher.S.png, perros.L.jpg)

Transformaciones en cada test:
- `package imagemin_test` → `package image_test`
- import `github.com/tinywasm/imagemin` → `github.com/tinywasm/image`
- referencias `imagemin.` → `image.`

Agregar tests de builders en `tinywasm/image/builders_test.go`:

```go
package image_test

import (
    "strings"
    "testing"
    . "github.com/tinywasm/image"
)

func TestImg_Basic(t *testing.T) {
    got := Img("/logo.png", "Logo").AsElement().String()
    if !strings.Contains(got, "src='/logo.png'") { t.Error("expected src") }
    if !strings.Contains(got, "alt='Logo'") { t.Error("expected alt") }
}

func TestImg_Void(t *testing.T) {
    got := Img("/a.png", "A").AsElement().String()
    if strings.Contains(got, "</img>") { t.Error("img should be void") }
}

func TestImg_LazySize(t *testing.T) {
    got := Img("/a.png", "A").Lazy().Size(200, 100).AsElement().String()
    if !strings.Contains(got, "loading='lazy'") { t.Error("expected lazy") }
    if !strings.Contains(got, "width='200'") { t.Error("expected width") }
}

func TestPicture_WithSources(t *testing.T) {
    got := Picture(
        Source("hero.webp", "image/webp"),
        Img("/hero.png", "Hero").AsElement(),
    ).String()
    if !strings.Contains(got, "<picture") { t.Error("expected picture") }
    if !strings.Contains(got, "type='image/webp'") { t.Error("expected webp source") }
}
```

---

## Paso 5: Mover documentación

Mover `tinywasm/imagemin/docs/ARCHITECTURE.md` → `tinywasm/image/docs/ARCHITECTURE.md` (actualizar referencias `imagemin` → `image`, agregar sección de builders).

Reescribir `tinywasm/image/README.md` (ver sección final).

---

## Paso 6: Archivar imagemin

Una vez que `tinywasm/image` compile y pase tests:
- Marcar `tinywasm/imagemin` como archivado (ver `tinywasm/imagemin/docs/ARCHIVE.md`).
- Actualizar `tinywasm/app` para importar `github.com/tinywasm/image` en lugar de `github.com/tinywasm/imagemin` (el `ImageHandler` ahora es `image.New(*Config)`).

---

## Contrato unificado (uso en componentes)

### `mycomponent.go` (wasm + backend) — render
```go
import (
    . "github.com/tinywasm/image"  // Img, Picture, Source
    . "github.com/tinywasm/html"   // Section, Div...
    "github.com/tinywasm/dom"
)

func (c *Hero) Render() *dom.Element {
    return Section(
        Img("/img/hero.M.webp", "Hero").Lazy().Size(1024, 512).AsElement(),
    ).Class("hero")
}
```

### `ssr.go` (`//go:build !wasm`) — declarar para procesar
```go
//go:build !wasm

package herosection

import "github.com/tinywasm/image"

func RenderImages() []image.Asset {
    return []image.Asset{
        {Path: "img/hero.png", Variants: image.AllVariants, Alt: "Hero"},
    }
}
```

> Una sola declaración por imagen. El extractor AST de `extract.go` ya parsea este literal `[]image.Asset{...}`. **No** se usa builder fluido para la declaración (el AST no resuelve cadenas de métodos).

---

## Preload — POSPUESTO

El plan original proponía `*ImageAsset` + `ImageProvider.RenderImages() *ImageAsset` + inyección de `<link rel=preload>` vía assetmin. **Se descarta:**
- Colisionaba con el `RenderImages() []Asset` de imagemin.
- assetmin no está probado en producción.
- El preload es un subconjunto de la declaración; si se necesita, se agrega un campo `Preload bool` a `image.Asset` más adelante (el extractor AST solo requiere una línea extra), sin paquete nuevo.

El "Cambio 6" del plan de assetmin queda eliminado.

---

## Verificación

```bash
cd tinywasm/image
go mod tidy
go build ./...            # backend: compila pipeline + builders
GOOS=js GOARCH=wasm go build ./...   # wasm: solo builders + types (sin imaging/nativewebp)
gotest
```

Verificar que el build wasm NO incluye `imaging`/`nativewebp`:
```bash
GOOS=js GOARCH=wasm go list -deps ./... | grep -E "imaging|nativewebp" && echo "FALLO: deps de build en wasm" || echo "OK: wasm limpio"
```

---

## README.md (reescribir)

```markdown
# tinywasm/image

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
```

Ver `tinywasm/docs/MASTER_PLAN.md` para el orden global de ejecución.
