# PLAN: tinywasm/image — Separar subpaquete `image/min`

## Repositorio
`github.com/tinywasm/image` — path local: `tinywasm/image/`

## Estado actual (ejecutado desde CHECK_PLAN.md)

El paquete raíz ya tiene:
- `builders.go` — `Img()`, `Picture()`, `Source()` (sin build tag)
- `types.go` — `Asset`, `Variant`, constantes (sin build tag)
- `handler.go`, `convert.go`, `extract.go`, `loader.go`, `cache.go` — `//go:build !wasm`
- `go.mod` — incluye `imaging`, `nativewebp`, `dom`, `fmt`
- `tests/` — builders_test + pipeline tests

## Problema

Los archivos `//go:build !wasm` del paquete raíz propagan complejidad hacia los importadores:
- `assetmin` importa `image.Handler` → debe gestionar el mismo build tag
- La API visible en `import . "github.com/tinywasm/image"` incluye `Handler`, `Config`, `ParsedAsset` — ruido para quien solo usa builders

## Solución: subpaquete `image/min`

Mover toda la capa de procesamiento a `tinywasm/image/min` (mismo módulo, go.mod compartido).

**Resultado:**
- `github.com/tinywasm/image` → solo builders + tipos. Sin build tags.
- `github.com/tinywasm/image/min` → Handler, pipeline WebP. Sin build tags — nunca se importa desde wasm.
- `assetmin` importa `"github.com/tinywasm/image/min"` — limpio, sin build tags en ningún lado.

```
tinywasm/image/
├── builders.go       # Img(), Picture(), Source()
├── types.go          # Asset, Variant
├── go.mod            # module github.com/tinywasm/image (compartido)
├── min/
│   ├── handler.go    # Handler, Config, New()
│   ├── convert.go    # ConvertToWebP
│   ├── extract.go    # ExtractImages, ParsedAsset (parsea image.go)
│   ├── loader.go     # LoadImages, ReloadModule
│   └── cache.go      # IsUpToDate (mtime fuente vs output; sin manifest)
└── tests/
    ├── builders_test.go
    ├── convert_test.go
    ├── extract_test.go
    ├── handler_test.go
    ├── loader_test.go
    ├── mtime_test.go
    ├── concurrency_test.go
    ├── types_test.go
    ├── setup_test.go
    └── testdata/
```

---

## Paso 1: Crear directorio `min/` y mover archivos de procesamiento

Mover desde `tinywasm/image/` a `tinywasm/image/min/`:
- `handler.go`
- `convert.go`
- `extract.go`
- `loader.go`
- `cache.go`

En cada archivo:
1. Eliminar la línea `//go:build !wasm`
2. Cambiar declaración de paquete: `package image  →  package min`

`ParsedAsset` se queda definido en `min/` — es interno al pipeline, por lo que dentro de `min`
la firma es `func ExtractImages(...) ([]ParsedAsset, error)` (sin prefijo).
`Asset` y `Variant` permanecen en el paquete raíz — los usan tanto builders como el pipeline.

### Ajuste de imports en `min/`

Los archivos de `min/` que referencian los tipos públicos del paquete raíz deben importarlo:

```go
import "github.com/tinywasm/image"
// image.Asset, image.Variant, image.AllVariants
```

> En `extract.go`, el literal declarado por el componente es `[]image.Asset{...}`; al parsearlo
> por AST, `ExtractImages` lo convierte a su tipo interno `[]ParsedAsset`.

### `extract.go`: parsear `image.go`, no `ssr.go` (convención `X.go`)

El extractor actual lee `ssr.go` (deprecado). Con la convención `X.go`, `RenderImages()` vive
en `image.go`. Cambiar en `min/extract.go`:
```go
// ANTES:
ssrPath := filepath.Join(moduleDir, "ssr.go")
// DESPUÉS:
imagePath := filepath.Join(moduleDir, "image.go")
```
Ajustar también el nombre de la variable y los mensajes de error que mencionen `ssr.go`.

### Alias stdlib en `min/convert.go`

El import de stdlib `image` choca con el paquete raíz. Usar alias:
```go
import (
    stdimage "image"
    "github.com/tinywasm/image"  // para image.Asset, image.Variant
    // ...
)
var processedImg stdimage.Image
```

---

## Paso 2: Actualizar imports en `tests/`

Los tests **no se mueven** — todos siguen en `tinywasm/image/tests/` con `package image_test`.
Un directorio de tests-only es un paquete externo independiente: puede importar tanto el
paquete raíz como el subpaquete `min` a la vez. No hay conflicto de nombre de paquete.

Tests del pipeline (`handler_test.go`, `convert_test.go`, etc.):
```go
package image_test  // se mantiene

import "github.com/tinywasm/image/min"
// min.Handler, min.New(), min.Config, min.ParsedAsset
// image.Asset, image.Variant — siguen del root (image)
```

`builders_test.go` no cambia — importa solo el paquete raíz `image`.

---

## Paso 3: Limpiar paquete raíz

Después de mover los archivos de procesamiento, el paquete raíz queda solo con:
- `builders.go`
- `types.go`

Verificar que no queden referencias a `handler.go`, `convert.go` etc. en el root.

---

## Paso 4: `image/min` implementa `assetmin.ImageProcessor`

Decisión de arquitectura: `image/min` **no** se registra como handler separado de devwatch.
Se **inyecta** en `assetmin` vía la interfaz `assetmin.ImageProcessor`. assetmin reconoce
`image.go` y delega; `tinywasm/app` cablea el `image/min` real.

`image/min.Handler` **ya implementa** la interfaz estructuralmente — sus métodos coinciden:

```go
// assetmin.ImageProcessor (definida en assetmin, NO importa image/min):
LoadImages() error                    // ya existe en Handler
ReloadModule(moduleDir string) error  // ya existe en Handler
UnobservedFiles() []string            // ya existe en Handler
```

No hay ciclo: `image/min` no importa `assetmin`; `assetmin` no importa `image/min`.
Solo `tinywasm/app` importa ambos y los conecta.

Cambio menor: `Handler.Name()` devuelve `"IMAGEMIN"` → cambiar a `"IMAGE"`.

La integración concreta (go.mod, construcción, `SetImageProcessor`) vive en
`tinywasm/app/docs/PLAN.md`. assetmin define la interfaz en `tinywasm/assetmin/docs/PLAN.md`.

---

## Paso 5: Detección de cambio — mtime contra outputs (sin manifest)

**Decisión:** **no** hay manifest ni hash persistente. Los `.webp` generados **son** el cache.
`cache.go` mantiene su lógica actual (`IsUpToDate`: el output existe y el fuente no es más nuevo
que el output). Solo se le quita `//go:build !wasm` al mover a `min/` (el subpaquete nunca entra
a wasm).

### Por qué mtime basta aquí
- Los outputs `.webp` **deben estar gitignored** (son artefactos de build). Git no los toca.
- Con outputs gitignored, mtime es robusto: `git checkout` solo actualiza el mtime de los
  *fuentes* cambiados; los outputs conservan el suyo → "fuente más nuevo que output" detecta
  exactamente lo que cambió. Tras `clone` no hay outputs → se generan todos (correcto).
- Estado persistente = los propios `.webp` en disco. No se necesita manifest ni memoria.

### Requisito
- `OutputDir` (ej. `web/public/img/*.webp`) debe estar en `.gitignore`. Documentarlo en el
  README de image y en la verificación.

### Hueco conocido (aceptable)
Restaurar un fuente preservando su mtime viejo (`cp -p`, extracción de `tar`, restore de backup)
puede saltar un cambio. Raro; workaround: borrar los outputs y dejar que se regeneren.

### Tests
- `tests/mtime_test.go` se mantiene (prueba la lógica mtime; solo ajustar import a `image/min`).
- `tests/extract_test.go` → los fixtures que escriben `ssr.go` deben escribir `image.go`.

---

## Paso 6: Verificación

```bash
# paquete raíz — solo builders (sin imaging/nativewebp en el build)
cd tinywasm/image
go build ./...
GOOS=js GOARCH=wasm go build .    # solo builders.go + types.go, limpio

# subpaquete min — pipeline completo
go build ./min/...

# tests (todos en tests/, package image_test, importan image + image/min)
cd tinywasm/image/tests
gotest
```

---

## Contrato de uso (sin cambios)

### Componente — render (builders)
```go
import . "github.com/tinywasm/image"   // solo Img, Picture, Source, Asset, Variant

func (c *Hero) Render() *dom.Element {
    return Img("/img/hero.M.webp", "Hero").Lazy().Size(1024, 512).AsElement()
}
```

### Componente — declaración (archivo `image.go`)
```go
// image.go
//go:build !wasm
package herosection

import "github.com/tinywasm/image"

func RenderImages() []image.Asset {
    return []image.Asset{
        {Path: "img/hero.png", Variants: image.AllVariants, Alt: "Hero"},
    }
}
```
> El nombre de archivo **debe** ser `image.go` — assetmin lo detecta por nombre y `image/min`
> parsea `RenderImages()` de ese archivo.

### App — construye el pipeline e inyecta en assetmin
```go
import "github.com/tinywasm/image/min"

imgProc := min.New(&min.Config{RootDir: ".", OutputDir: "web/public/img", Quality: 80})
assetsHandler.SetImageProcessor(imgProc)  // assetmin enruta image.go → imgProc
```

---

## go.mod — sin cambios

El go.mod raíz sigue teniendo `imaging` y `nativewebp` porque `image/min` es parte del mismo módulo. La separación de dependencias no aplica aquí — el beneficio es de **API y propagación de build tags**, no de go.mod.

Si en el futuro se necesita go.mod separado (ej: para publicar versiones independientes), `image/min` puede extraerse como módulo `github.com/tinywasm/imagemin` en ese momento.

---

Ver `tinywasm/docs/MASTER_PLAN.md` para el orden global de ejecución.
