---
PLAN: "feat: derivar el juego de iconos de un sitio desde su logo"
TAG: v0.1.0
EXECUTOR: jules
REVIEWER: none
---

> Plan autocontenido: todo lo necesario para ejecutarlo está aquí.
> Se despacha con el flujo CodeJob. Ver skill: agents-workflow.

# Plan — `image/favicon`: un cuadrado entra, el juego de iconos sale

**Requisito previo**, porque este entorno no lo trae instalado:

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

## 1. El problema, con el caso real que lo destapó

`veltylabs/misitio` es un panel donde un cliente administra su sitio web. El
cliente sube **un** logo, y de ahí tienen que salir el logo de la plantilla y el
icono de la pestaña del navegador. Hoy no sale ninguno: nadie en el ecosistema
sabe convertir un logo en un icono.

Y no es sólo el sitio del cliente. Una aplicación cualquiera construida con este
ecosistema tampoco puede declarar su icono — `sitec` regenera un `favicon.svg`
vacío en cada build y enlaza ese archivo — así que **todas** las páginas salen
con el icono por defecto del navegador.

Este repo es el dueño natural de la pieza que falta: ya decodifica, redimensiona
y codifica imágenes (`min`), y ya comprime en el navegador (`browser`). Falta la
tercera operación: **de un cuadrado grande, los tamaños que pide un navegador**.

Este plan hace **sólo** eso. Quién lo llama y quién emite el `<head>` es asunto
de `sitec`, en su propio plan.

## 2. La decisión de producto, ya tomada

Está razonada en
[`misitio/docs/DESIGN.md` §12](https://github.com/veltylabs/misitio/blob/main/docs/DESIGN.md).
Lo que importa aquí:

- La fuente es un **ráster cuadrado de al menos 256×256** (PNG, WebP o JPEG).
  Un vector no se puede derivar de un ráster, pero todos los tamaños sí.
- **Nunca se amplía.** Un icono borroso es peor que ningún icono.
- El SVG es **opcional y adicional**, nunca la fuente única: Safari quiere un
  `apple-touch-icon` PNG y los previsualizadores de enlaces piden ráster.

## 3. La API — paquete nuevo `favicon`

Archivos nuevos: `favicon/favicon.go`, `favicon/ico.go`, `favicon/errors.go`.
**Todos con `//go:build !wasm`**: esto decodifica imágenes con la stdlib y jamás
debe llegar a un binario del navegador.

```go
package favicon

// MinEdge es el lado mínimo de la fuente. Cubre todos los tamaños que se emiten
// sin ampliar ninguno: el mayor es el icono de Android, 192.
const MinEdge = 256

// Source es la marca que declara un proyecto.
type Source struct {
	// Raster es el logo cuadrado. Obligatorio. PNG, WebP o JPEG.
	Raster []byte

	// SVG es el mismo logo en vectorial. Opcional: mejora la nitidez en
	// pantallas grandes y no reemplaza al ráster.
	SVG []byte
}

// File es un artefacto listo para escribirse en la salida del sitio.
type File struct {
	Name      string // "icon-32.png"
	Content   []byte
	Mediatype string // "image/png"
	Rel       string // "icon", "apple-touch-icon" — vacío: no se enlaza
	Sizes     string // "32x32" — vacío: sin atributo sizes
	Type      string // valor del atributo type del <link> — vacío: sin atributo
}

// Derive produce el juego completo a partir de la fuente.
func Derive(s Source) ([]File, error)
```

### 3.1 — Validación, en este orden y con estos errores

`favicon/errors.go`, con `github.com/tinywasm/fmt`:

| Comprobación | Error |
|---|---|
| `len(s.Raster) == 0` | `ErrNoRaster` |
| no se puede decodificar | `ErrUndecodable` |
| ancho ≠ alto | `ErrNotSquare` |
| lado < `MinEdge` | `ErrTooSmall` |

Los dos últimos mensajes **llevan las medidas reales**: quien los lea es un
cliente al que hay que decirle qué le pasa a su archivo, no un programador.
Textual:

```
favicon: el logo debe ser cuadrado: 800x600
favicon: el logo debe medir al menos 256x256: 128x128
```

### 3.2 — Lo que emite

En este orden exacto:

| Archivo | Lado | `Rel` | `Sizes` | `Type` |
|---|---|---|---|---|
| `icon-32.png` | 32 | `icon` | `32x32` | `image/png` |
| `icon-192.png` | 192 | `icon` | `192x192` | `image/png` |
| `apple-touch-icon.png` | 180 | `apple-touch-icon` | `180x180` | *(vacío)* |
| `favicon.ico` | 32 | *(vacío)* | *(vacío)* | *(vacío)* |
| `favicon.svg` | — | `icon` | *(vacío)* | `image/svg+xml` |

- `favicon.svg` sale **sólo** si `Source.SVG` no está vacío, y se copia
  **tal cual**: este paquete no sanea SVG. Lo hace `github.com/tinywasm/svg`, y
  el que sanea es quien recibe archivos de terceros, no quien los redimensiona.
- `favicon.ico` no lleva `Rel` a propósito: **nadie lo enlaza**. Los navegadores
  viejos lo piden solos a la raíz del sitio; ese es todo su mecanismo.
- Redimensionado con `github.com/disintegration/imaging` —ya es dependencia de
  este repo— con `imaging.Lanczos`, que es lo que conserva el borde limpio de un
  logo al bajar a 32 px.
- Codificación con `image/png` de la stdlib. **No** uses WebP aquí: un icono
  WebP no lo pide ningún navegador y el `.ico` no lo admite.

**Nunca amplíes.** Con `MinEdge = 256` los cuatro tamaños son menores que
cualquier fuente válida, así que hoy no puede pasar; escribe igual la guarda —
`if size > srcEdge { continue }`— porque el día que alguien agregue un icono de
512 esa línea es la diferencia entre omitirlo y publicar algo borroso.

### 3.3 — El `.ico`, byte a byte

`favicon/ico.go`. No hace falta un codificador: el formato admite un PNG
embebido, así que son 22 bytes de cabecera y el PNG que ya generaste. Todo en
**little-endian**:

```
ICONDIR      (6 bytes)
  uint16  0        reservado, siempre 0
  uint16  1        tipo: 1 = icono
  uint16  1        cantidad de imágenes

ICONDIRENTRY (16 bytes)
  uint8   32       ancho
  uint8   32       alto
  uint8   0        colores de la paleta: 0 = sin paleta
  uint8   0        reservado
  uint16  1        planos
  uint16  32       bits por pixel
  uint32  len(png) tamaño de la imagen
  uint32  22       desplazamiento donde empieza el PNG

<bytes del PNG de 32x32>
```

## 4. Tests — `tests/favicon_test.go`

`gotest`, nunca `go test`. En `tests/`, sólo aserciones de la stdlib.

Construye las imágenes de prueba **en memoria** con `image.NewRGBA` y
`png.Encode` — nada de archivos binarios nuevos en `testdata/`, que no se pueden
revisar en un diff.

| Test | Qué fija |
|---|---|
| `TestDeriveEmitsTheFullSet` | Cinco archivos con un SVG, cuatro sin él, con los nombres exactos de §3.2. |
| `TestDeriveSizes` | Cada PNG emitido mide lo que dice su nombre: se decodifica y se comprueban ancho y alto. |
| `TestDeriveRejectsNonSquare` | 800×600 → `ErrNotSquare`, y **el mensaje contiene "800x600"**. |
| `TestDeriveRejectsTooSmall` | 128×128 → `ErrTooSmall`, mensaje con "128x128". |
| `TestDeriveRejectsEmptyAndGarbage` | Ráster vacío → `ErrNoRaster`; bytes que no son imagen → `ErrUndecodable`. |
| `TestDeriveNeverUpscales` | Con una fuente de exactamente 256, ningún archivo emitido supera los 256 px. |
| `TestDeriveCopiesSVGVerbatim` | El `favicon.svg` emitido es byte a byte el que entró: este paquete no lo toca. |
| `TestIcoWrapsPNG` | La cabecera del `.ico` es la de §3.3 —tipo 1, una imagen, desplazamiento 22— y a partir del byte 22 hay un PNG decodificable de 32×32. |

## 5. Documentación

- [`README.md`](../README.md): una cuarta capa en la tabla de paquetes —
  `image/favicon`, `!wasm`, "de un logo cuadrado al juego de iconos"— y un
  ejemplo de cinco líneas.
- [`docs/ARCHITECTURE.md`](ARCHITECTURE.md): la sección de estructura de
  paquetes menciona el nuevo, con la regla de que **valida y deriva, pero no
  sanea**: un SVG de un tercero se limpia en `tinywasm/svg`.

Ningún documento debe citar `docs/PLAN.md`: este archivo se borra al publicar.

## 6. Criterios de aceptación

- [ ] `gotest` en verde.
- [ ] `gofmt -l .` vacío.
- [ ] `head -1 favicon/*.go` → todos empiezan con `//go:build !wasm`.
- [ ] `grep -rn "favicon" browser/ min/ *.go` → vacío: el paquete nuevo no se
      cuela en el compresor del navegador ni en la tubería de imágenes.
- [ ] `go.mod` **sin dependencias nuevas**: `imaging` ya está y el resto es
      stdlib.
- [ ] Los ocho tests de §4 existen y pasan.

## 7. Anti-footguns

1. **`//go:build !wasm` en todo el paquete.** Este repo compila también para el
   navegador; `image/png` y `imaging` en un binario TinyGo son cientos de KB que
   no caben en ningún presupuesto.
2. **No toques `min/` ni `browser/`.** Son la tubería de imágenes del sitio y la
   compresión previa a la subida: otro problema, otras reglas.
3. **No sanees el SVG aquí.** Se copia tal cual. Limpiar SVG de terceros es de
   `tinywasm/svg`, y mezclarlo aquí pondría un parser XML dentro de un
   redimensionador.
4. **No inventes formatos de salida.** Nada de WebP ni de PNG de 512: el juego
   es el de §3.2, ni uno más.
5. `docs/PLAN.md` (este archivo) no se renombra ni se borra, y su frontmatter
   —`PLAN`, `TAG`, `EXECUTOR`, `STATUS`, `SESSION`, `PR`— **no se edita a mano**.
