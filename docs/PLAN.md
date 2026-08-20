---
PLAN: "feat(browser): compresion de imagenes en el navegador antes de subir"
EXECUTOR: jules
REVIEWER: none
---

> Este plan se despacha con el flujo CodeJob. Ver skill: agents-workflow.

# Plan — `tinywasm/image/browser`

Agregar una **tercera capa** a `tinywasm/image`: comprimir y redimensionar una
imagen **en el navegador**, antes de subirla, usando el `canvas` del propio
navegador.

## Por qué, y por qué aquí

`tinywasm/image` ya tiene dos capas, y la nueva completa el mapa:

| Capa | Paquete | Build | Cuándo corre |
|---|---|---|---|
| Builders | `github.com/tinywasm/image` | sin tags | render, en ambos lados |
| Pipeline | `github.com/tinywasm/image/min` | `//go:build !wasm` | build, en el servidor |
| **Navegador** | `github.com/tinywasm/image/browser` | `//go:build wasm` | **antes de subir, en el cliente** |

El hueco es real y concreto: `min` optimiza lo que ya está en el árbol al
construir, así que **entre que el usuario elige un archivo y que el build lo
procesa, no hay nada**. Una app que recibe fotos de teléfono (3–6 MB) tiene que
subirlas enteras, pagar el ancho de banda y almacenarlas antes de poder
reducirlas — o rechazarlas, que es peor producto.

El navegador ya sabe decodificar y volver a codificar imágenes. Esta capa expone
esa capacidad como una función de Go.

**Es aditivo**: no toca `image` ni `image/min`, no cambia ninguna firma
existente, y su archivo lleva `//go:build wasm`, así que no puede entrar en un
build de backend ni por accidente.

---

## 0. Reglas de desarrollo

### 0.1 Este paquete es **solo wasm**

Todo archivo no-test de `browser/` empieza con:

```go
//go:build wasm
```

`syscall/js` está permitido y es necesario — es lo que ya hace
`tinywasm/dom` (`dom/element_wasm.go`, `dom/dom_frontend.go`). No es una
excepción que haya que justificar.

### 0.2 Sin stdlib pesada

- **Sin `fmt`, `errors`, `strconv`, `strings`, `log`.** Usa
  `github.com/tinywasm/fmt`: `fmt.Errf(...)` para construir errores.
- **Sin `encoding/json`, sin `reflect`.**
- **Sin el paquete `image` de la stdlib.** El nombre de este módulo lo sombrea a
  propósito: el `image` estándar es demasiado pesado para TinyGo. Toda la
  decodificación y codificación la hace el navegador, no Go.
- **Sin `map[K]V`.**

### 0.3 Sin strings mágicos

Todo string repetido (nombre de método JS, tipo MIME, clave de opción) es una
constante nombrada. Literales en la lógica: prohibidos.

### 0.4 Idioma y estructura

Código en inglés; documentación y comentarios de prosa en español. Archivos de
menos de 500 líneas. Tests bajo `browser/` con `//go:build wasm`, ejecutables con
el arnés wasm del ecosistema.

---

## 1. La API

Archivo `browser/compress.go`.

```go
// Config son los parametros de compresion.
type Config struct {
	// MaxEdge es el lado mas largo en pixeles despues de redimensionar,
	// manteniendo la proporcion. Cero = no redimensionar.
	MaxEdge int

	// Quality es la calidad del codificador, de 0 a 1. Cero usa DefaultQuality.
	Quality float64

	// Type es el MIME de salida. Vacio usa DefaultType.
	Type string
}

// Result es la imagen ya comprimida.
type Result struct {
	Data   []byte // los bytes listos para subir
	Type   string // el MIME que el navegador produjo realmente
	Width  int    // dimensiones finales
	Height int
}

// Compress decodifica el archivo, lo redimensiona si hace falta y lo vuelve a
// codificar con los parametros dados.
//
// file es un js.Value de un objeto File o Blob — tipicamente
// input.Get("files").Index(0) desde el evento change de un <input type=file>.
func Compress(file js.Value, c Config) (Result, error)

// CompressToFit repite Compress bajando la calidad hasta que el resultado
// quepa en maxBytes, o hasta agotar los intentos.
//
// Existe porque el tamaño de salida de un codificador no es predecible desde
// la entrada: una foto con mucho ruido puede pesar el triple que una plana con
// las mismas dimensiones y la misma calidad. Un consumidor con un limite duro
// de subida necesita esto, no una calidad fija.
func CompressToFit(file js.Value, maxBytes int, c Config) (Result, error)
```

Constantes exportadas:

```go
const (
	DefaultQuality = 0.85
	DefaultType    = "image/webp"
	DefaultMaxEdge = 1920
)
```

`DefaultMaxEdge = 1920` porque es el ancho al que un sitio sirve su imagen más
grande; guardar más píxeles de los que se van a mostrar es peso que nadie ve.

---

## 2. Cómo funciona — todo por Promises

Los cuatro pasos son asíncronos y **los cuatro devuelven Promises**, así que los
cuatro se resuelven con `github.com/tinywasm/await`:

| # | Llamada JS | Devuelve |
|---|---|---|
| 1 | `createImageBitmap(file)` | Promise de `ImageBitmap` |
| 2 | `new OffscreenCanvas(w, h)` + `getContext("2d").drawImage(...)` | síncrono |
| 3 | `canvas.convertToBlob({type, quality})` | Promise de `Blob` |
| 4 | `blob.arrayBuffer()` | Promise de `ArrayBuffer` |

`await.Promise(v)` bloquea la goroutine hasta que la Promise resuelve y libera
sus callbacks al salir. **No escribas tu propio puente con `js.FuncOf` y un
canal**: eso es exactamente lo que `await` ya hace sin fugas.

**Anti-footgun — usa `OffscreenCanvas`, no `<canvas>`.** El `canvas` del DOM
ofrece `toBlob`, que recibe un **callback**, no una Promise, y obligaría a
construir a mano el puente que `await` evita. `OffscreenCanvas.convertToBlob()`
devuelve una Promise y encaja con el resto sin código extra. Además no toca el
DOM, así que la compresión no depende de que haya un documento.

### 2.1 Del `ArrayBuffer` a `[]byte`

```go
u8 := js.Global().Get(jsUint8Array).New(buf)
out := make([]byte, u8.Get("length").Int())
js.CopyBytesToGo(out, u8)
```

`js.CopyBytesToGo` copia en un solo paso. **No recorras el `Uint8Array` con
`.Index(i)` en un bucle**: para una imagen de un megabyte eso es un millón de
cruces de la frontera JS↔Go y congela la pestaña.

### 2.2 El cálculo de dimensiones

Se preserva la proporción y **nunca se amplía**: si el lado más largo ya es menor
que `MaxEdge`, las dimensiones no cambian. Ampliar una imagen chica sólo agrega
peso sin agregar detalle.

---

## 3. Navegadores sin `OffscreenCanvas`

Si `OffscreenCanvas` no existe en `js.Global()`, `Compress` devuelve
`ErrUnsupported` **de inmediato**, antes de leer el archivo.

```go
// ErrUnsupported indica que el navegador no ofrece OffscreenCanvas y por
// tanto no puede comprimir. El consumidor decide que hacer: pedir un archivo
// mas chico, o subir sin comprimir si su limite lo permite.
var ErrUnsupported = fmt.Errf("image/browser: el navegador no soporta OffscreenCanvas")
```

**No implementes un camino alternativo con `<canvas>` y `toBlob`.** Duplicaría
el código para cubrir navegadores de hace tres años, y el consumidor ya tiene una
salida sensata: pedir un archivo más chico. Un error claro vale más que un
segundo camino que casi nadie ejecuta y que nadie prueba.

---

## 4. Estructura de archivos

```
browser/compress.go     // Config, Result, Compress, CompressToFit
browser/js.go           // constantes de nombres JS y helpers de js.Value
browser/errors.go       // ErrUnsupported y demas centinelas
browser/compress_test.go
```

Todos con `//go:build wasm`.

---

## 5. Tests — `browser/`, con `//go:build wasm`

Corren en el arnés wasm del ecosistema (`tinywasm/wasmbrowsertest`), no con
`go test` normal.

| # | Caso | Espera |
|---|---|---|
| 1 | imagen de 3000×2000, `MaxEdge: 1920` | resultado de 1920×1280, proporción intacta |
| 2 | imagen de 800×600, `MaxEdge: 1920` | 800×600 — **no se amplía** |
| 3 | `Config{}` vacío | usa `DefaultQuality`, `DefaultType`, `DefaultMaxEdge` |
| 4 | `CompressToFit` con un límite holgado | un solo intento |
| 5 | `CompressToFit` con un límite estrecho | baja la calidad y cabe, o devuelve error tras agotar intentos |
| 6 | `OffscreenCanvas` ausente (stub) | `ErrUnsupported`, **sin leer el archivo** |
| 7 | `file` que no es una imagen | error, no pánico |
| 8 | el `Result.Data` devuelto | son los bytes de una imagen decodificable por el navegador |

El caso 2 es el que la gente rompe al escribir el cálculo de dimensiones: un
`escala = MaxEdge / lado` sin tope amplía las imágenes chicas.

El caso 6 verifica **el orden**: el chequeo va antes de leer el archivo, no
después. Leer 6 MB para luego descubrir que no se puede comprimir es trabajo
tirado.

---

## 6. Documentación

- `README.md` — agregar la **tercera capa** a la sección "Dos capas" (que pasa a
  ser tres), con un ejemplo de uso desde un `<input type=file>`.
- `docs/ARCHITECTURE.md` — la tabla de capas del §"Por qué, y por qué aquí" de
  este plan, y por qué `OffscreenCanvas` y no `<canvas>`.
- Si escribes diagramas: **nunca uses `subgraph`** (rompe el renderizado en el
  TUI). `flowchart TD` y `<br/>` para los saltos.

---

## 7. Criterios de aceptación

- [ ] `go vet ./...` limpio.
- [ ] `GOOS=js GOARCH=wasm go build ./...` sin errores.
- [ ] `go build ./...` (backend) **sigue funcionando**: el paquete `browser` no
      entra porque todos sus archivos llevan `//go:build wasm`.
- [ ] Los 8 casos en verde en el arnés wasm.
- [ ] `head -1 browser/*.go` → todos empiezan con `//go:build wasm`.
- [ ] `grep -rn "\"fmt\"\|\"errors\"\|\"strings\"\|\"strconv\"\|\"log\"\|encoding/json\|\"reflect\"\|\"image\"" browser/` → vacío.
- [ ] `grep -rn "map\[" browser/` → vacío.
- [ ] `grep -rn "js.FuncOf" browser/` → vacío: **se usa `tinywasm/await`**.
- [ ] `grep -rn "toBlob\|createElement" browser/` → vacío: **`OffscreenCanvas`**.
- [ ] `grep -rn "\.Index(" browser/` → vacío en el camino de copia de bytes:
      **`js.CopyBytesToGo`**.
- [ ] `git diff --stat` no muestra cambios en `builders.go`, `types.go` ni `min/`.
- [ ] `README.md` y `docs/ARCHITECTURE.md` actualizados.

## 8. Fuera de alcance

- Recortes, rotación, filtros o corrección de orientación EXIF.
- Cualquier decodificación o codificación **en Go**: la hace el navegador.
- Subir el archivo: esta capa devuelve bytes, no habla con la red.
- Políticas de tamaño máximo: `CompressToFit` recibe el límite como parámetro;
  decidir cuál es es del consumidor.
