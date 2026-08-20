---
PLAN: "feat: constructor de imagenes responsivas con srcset"
EXECUTOR: jules
REVIEWER: none
---

> Este plan se despacha con el flujo CodeJob. Ver skill: agents-workflow.
>
> **Fase A (puerta)** de un cambio en tres repos: aquí nace el constructor;
> después `tinywasm/components` (`herobanner`) y `tinywasm/layout` (`landing`)
> lo consumen, en planes propios. **Este plan es autocontenido**: no necesitas
> ver los otros dos, y no debes tocar esos repos desde aquí.

# Plan — `tinywasm/image`: `srcset` para todo el ecosistema

Dar a este paquete la capacidad de emitir una imagen **responsiva**: un `<img>`
con `srcset` y `sizes` que declara las tres variantes que el pipeline ya genera.

## El problema, medido

Sobre `veltylabs/mjosefa-website` en producción: el pipeline reduce las fotos de
~5 MB a 47–110 KB, pero el HTML **no tiene ni un `srcset`**. Un teléfono de
400 px descarga los mismos 110 KB que un escritorio, cuando le bastaban ~25 KB.

Las variantes ya se generan. Lo que falta es **declararlas**, y hoy nadie puede:
la convención de sufijos vive en `min/convert.go`, que es `//go:build !wasm`, y
quien tiene que emitir el HTML renderiza también en wasm.

---

## 0. Reglas de desarrollo

### 0.1 El paquete raíz **compila para ambos lados**

`builders.go` y `types.go` **no llevan build tags** y deben seguir sin llevarlos:
los consumen `landing` y `components`, que renderizan en el navegador y en el
build. Todo lo que agregues al paquete raíz tiene que compilar con TinyGo.

- **Sin `fmt`, `errors`, `strconv`, `strings`, `log`** de la stdlib. Usa
  `github.com/tinywasm/fmt` — es lo que ya hace `builders.go`.
- **Sin `map[K]V`.**
- **Sin `reflect`, sin `encoding/json`.**
- **Sin el paquete `image` de la stdlib** fuera de `min/` (donde se aliasa como
  `stdimage`). El nombre de este módulo lo sombrea a propósito.

**Anti-footgun:** `min/` **sí** es backend (`//go:build !wasm`) y usa la stdlib
legítimamente. No "corrijas" sus imports aplicándole las reglas del paquete raíz.

### 0.2 Sin strings mágicos

Todo string repetido es una constante nombrada. Literales en la lógica:
prohibidos.

### 0.3 Idioma

Código en inglés; documentación y comentarios de prosa en español.

---

## 1. Subir la convención de sufijos a `types.go`

Hoy `min/convert.go` tiene esto, privado y sólo backend:

```go
func variantSuffix(v image.Variant) string {
	switch v {
	case image.VariantS:
		return "S"
	case image.VariantM:
		return "M"
	case image.VariantL:
		return "L"
	default:
		return "unknown"
	}
}
```

**Es el problema de raíz**: el productor de los archivos conoce la convención y
el consumidor que debe declararlos no puede verla.

Muévela a `types.go` (paquete raíz, sin build tags), como método del tipo que ya
vive ahí:

```go
// Suffix es la marca que el pipeline intercala antes de la extension:
// "foto.jpg" con VariantM produce "foto.M.jpg".
//
// Vive aqui y no en min/ porque quien ESCRIBE los archivos y quien los
// DECLARA en el HTML tienen que compartir una sola definicion: min/ es
// backend, y el que emite el srcset renderiza tambien en wasm.
func (v Variant) Suffix() string

// Width es el ancho en pixeles al que el pipeline redimensiona esta variante.
// Es el descriptor "w" que el srcset necesita para que el navegador elija.
func (v Variant) Width() int
```

Anchos, que ya están documentados en `types.go`:

```go
const (
	WidthS = 640
	WidthM = 1024
	WidthL = 1920
)
```

Luego **borra `variantSuffix` de `min/convert.go`** y usa `v.Suffix()`.
Verificación: `grep -rn "func variantSuffix" .` → vacío.

El caso `default` devuelve cadena vacía, no `"unknown"`: un sufijo `"unknown"`
producía nombres de archivo `foto.unknown.jpg` que no fallan y no sirven. Con
cadena vacía el llamador puede detectar el caso.

---

## 2. El constructor — `builders.go`

```go
// Responsive construye un <img> con srcset para las tres variantes que el
// pipeline genera, a partir de la ruta BASE (sin sufijo de variante).
//
//	Responsive("/img/foto.jpg", "Fachada")
//
// emite:
//
//	<img src="/img/foto.M.jpg"
//	     srcset="/img/foto.S.jpg 640w, /img/foto.M.jpg 1024w, /img/foto.L.jpg 1920w"
//	     sizes="100vw" alt="Fachada">
//
// El src apunta a la variante M para que un navegador que ignore srcset reciba
// algo razonable en vez de la version de escritorio.
func Responsive(base, alt string) *ImgElement

// Sizes declara al navegador que ancho ocupara la imagen en el layout, para
// que pueda elegir la variante ANTES de conocer el CSS.
//
// Sin esto el navegador asume 100vw y baja de mas en cualquier imagen que no
// ocupe el ancho completo — una tarjeta en una grilla de tres columnas es el
// caso tipico.
func (i *ImgElement) Sizes(s string) *ImgElement

// Srcset fija el atributo srcset a mano. Escape hatch para un consumidor con
// variantes que no siguen la convencion; Responsive es el camino normal.
func (i *ImgElement) Srcset(s string) *ImgElement
```

Constante:

```go
const DefaultSizes = "100vw"
```

### 2.1 `<img srcset>`, **no** `<picture>`

`Picture()` y `Source()` ya existen en este paquete y **no se tocan**: sirven
para *art direction* (recortes distintos por pantalla) y para ofrecer formatos
alternativos.

Lo que hace falta aquí es otra cosa — **resolution switching**: la misma imagen
en tres tamaños. Para eso el elemento correcto es `<img srcset sizes>`, que deja
que el navegador considere también la densidad de pantalla y su propia
heurística de red. Un `<picture>` con `media` fuerza la decisión desde el
servidor y desperdicia esa información.

### 2.2 Cómo se arma la ruta de una variante

```
base = "/img/foto.jpg"  →  "/img/foto" + "." + v.Suffix() + ".jpg"
```

Se corta por el **último** punto de la ruta. **Anti-footgun:** un directorio con
punto (`/img/v1.2/foto.jpg`) rompe cualquier implementación que corte por el
primero. Y si la ruta **no tiene** extensión, `Responsive` devuelve un `<img>
src=base` sin `srcset` en vez de generar rutas inválidas — degradar es correcto,
inventar archivos que no existen no.

---

## 3. Estructura de archivos

```
types.go        // modificar — Variant.Suffix(), Variant.Width(), WidthS/M/L
builders.go     // modificar — Responsive(), Sizes(), Srcset(), DefaultSizes
min/convert.go  // modificar — borrar variantSuffix, usar v.Suffix()
tests/          // los casos de §4
```

No se crean archivos nuevos. No se toca `Img`, `Lazy`, `Size`, `Class`, `Attr`,
`AsElement`, `String`, `Picture` ni `Source`: **este plan es puramente aditivo
para los consumidores actuales.**

---

## 4. Tests — `tests/`

| # | Caso | Espera |
|---|---|---|
| 1 | `VariantS.Suffix()`, `M`, `L` | `"S"`, `"M"`, `"L"` |
| 2 | `Variant(0).Suffix()` | cadena vacía, **no** `"unknown"` |
| 3 | `VariantS.Width()`, `M`, `L` | 640, 1024, 1920 |
| 4 | `Responsive("/img/foto.jpg", "alt")` | `src` es `.M.jpg`; `srcset` trae las tres con `640w`, `1024w`, `1920w`, en ese orden |
| 5 | `Responsive` sin `.Sizes(...)` | `sizes="100vw"` |
| 6 | `.Sizes("(max-width: 600px) 100vw, 33vw")` | ese valor exacto |
| 7 | `Responsive("/img/v1.2/foto.jpg", "alt")` | corta por el **último** punto: `/img/v1.2/foto.M.jpg` |
| 8 | `Responsive("/img/foto", "alt")` | `<img>` con `src="/img/foto"` y **sin** `srcset` |
| 9 | `Responsive(...).Lazy().Class("x")` | los métodos existentes siguen encadenando |
| 10 | `Img("/img/foto.jpg", "alt")` | **sin** `srcset`: el comportamiento viejo no cambia |

El caso 10 es el que protege a los consumidores actuales: este plan agrega un
camino, no cambia el que ya existe.

---

## 5. Documentación

- `README.md` — `Responsive` en la sección de render, con el ejemplo de `sizes`
  para una tarjeta en grilla (que es donde `100vw` se queda corto).
- `docs/ARCHITECTURE.md` — por qué la convención de sufijos vive en `types.go` y
  no en `min/`, y por qué `<img srcset>` y no `<picture>`.
- Si escribes diagramas: **nunca uses `subgraph`** (rompe el renderizado en el
  TUI). `flowchart TD` y `<br/>` para los saltos.

---

## 6. Criterios de aceptación

- [ ] `go vet ./...` limpio; `go test ./tests/...` en verde con los 10 casos.
- [ ] `GOOS=js GOARCH=wasm go build ./...` sin errores: el paquete raíz sigue
      compilando para el navegador.
- [ ] `go build ./...` limpio.
- [ ] `grep -rn "func variantSuffix" .` → vacío.
- [ ] `grep -rn "\"S\"\|\"M\"\|\"L\"" min/` → vacío: la convención está en
      `types.go` y `min/` la consume.
- [ ] `head -1 types.go builders.go` → **ninguno** empieza con `//go:build`.
- [ ] `grep -n "\"fmt\"\|\"strings\"\|\"errors\"\|\"strconv\"\|\"log\"" types.go builders.go` → vacío.
- [ ] `grep -n "map\[" types.go builders.go` → vacío.
- [ ] `git diff --stat` no muestra cambios en las firmas de `Img`, `Picture` ni
      `Source`.

## 7. Fuera de alcance

- Cambiar `herobanner` o `landing` para que usen `Responsive` — son las fases B
  y C del master plan, en sus propios repos.
- Tocar `loadingFor` de `herobanner`: **ya es correcto** (capa 0 `eager`, resto
  `lazy`).
- Cambiar el pipeline de conversión: funciona. Esto es un problema de
  **declaración**, no de conversión.
- AVIF, art direction, o generar variantes nuevas.
