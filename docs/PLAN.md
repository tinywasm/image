---
PLAN: "fix: fuente unica de anchos, escalera 480/1024/1600 y calidad por defecto 62"
EXECUTOR: jules
REVIEWER: none
---

> Este plan se despacha con el flujo CodeJob. Ver skill: agents-workflow.
>
> Corrige dos defectos que sobrevivieron al PR #8 (`Responsive()` con `srcset`)
> y ajusta la escalera de anchos. **Este plan es autocontenido**: no necesitas
> leer el PR anterior ni otros repos, y no debes tocar otros repos desde aquí.

# Plan — `tinywasm/image`: una sola fuente de anchos, y una escalera que pesa lo que debe

El PR #8 subió la convención de **sufijos** (`.S`, `.M`, `.L`) a `types.go` para
que el pipeline y el renderizador compartieran una definición. Los **anchos** se
quedaron atrás: `types.go` los declara y `min/convert.go` los repite como
literales. Este plan cierra esa duplicación y, ya con una sola fuente de verdad,
cambia la escalera.

## Los tres problemas, medidos

**1. Los anchos están duplicados.** `types.go` define `WidthS/M/L` y
`min/convert.go:36-38` los repite hardcodeados. Cambiar uno sin el otro produce
archivos de un ancho y un `srcset` que declara otro — el navegador elige mal y
nada falla ruidosamente.

**2. La escalera está mal calibrada.** El navegador elige calculando
`ancho_layout_css × DPR`. A `sizes="100vw"` ningún teléfono moderno pide menos
de 720 px reales, así que la variante S (640) **nunca se elige**; y un teléfono
DPR 3 pide 1179 y cae en **L**, descargando ~385 KB a la calidad actual.

**3. El descriptor `w` puede mentir.** `ProcessImage` no amplía: una fuente de
1280 px escribe un `foto.L.jpg` de 1280 px que el `srcset` declara `1600w`.

Números medidos con el propio encoder del pipeline (foto densa, peor caso):

| ancho | q62 | q75 |
|---|---|---|
| 480 | 28 KB | 36 KB |
| 1024 | 106 KB | 138 KB |
| 1600 | ~202 KB | ~267 KB |
| 1920 | ~284 KB | ~385 KB |

El porqué completo de `480/1024/1600` y de `q62` ya está escrito en
[`docs/ARCHITECTURE.md`](ARCHITECTURE.md) → "La escalera de anchos" y "Calidad
JPEG por defecto". **No lo re-deduzcas ni lo cambies**: este plan sólo lo
implementa.

---

## 0. Reglas de desarrollo

### 0.1 El paquete raíz compila para ambos lados

`builders.go` y `types.go` **no llevan build tags** y deben seguir sin llevarlos:
los consumen paquetes que renderizan en el navegador y en el build. Todo lo que
agregues al paquete raíz tiene que compilar con TinyGo.

- **Sin `fmt`, `errors`, `strconv`, `strings`, `log`** de la stdlib. Usa
  `github.com/tinywasm/fmt` — es lo que ya hace `builders.go`.
- **Sin `map[K]V`.** **Sin `reflect`, sin `encoding/json`.**
- **Sin el paquete `image` de la stdlib** fuera de `min/` (donde se aliasa como
  `stdimage`). El nombre de este módulo lo sombrea a propósito.

### 0.2 Anti-footguns

- **`min/` SÍ es backend** (`//go:build !wasm`) y usa la stdlib legítimamente.
  No "corrijas" sus imports aplicándole las reglas del paquete raíz.
- **`browser/compress.go` NO se toca.** Tiene `DefaultMaxEdge = 1920`, que es
  otra cosa: el ancho máximo al que el navegador comprime una foto **antes de
  subirla**. No es la escalera de render. Dejar en 1920.
  Igual para `browser/compress_test.go`, que asserta 1920 en ese contexto.
- **No se toca** `Img`, `Lazy`, `Size`, `Class`, `Attr`, `AsElement`, `String`,
  `Picture`, `Source`, `Responsive`, `Sizes` ni `Srcset` en sus firmas.
  Este plan cambia valores y elimina duplicación, no la API.

### 0.3 Sin strings mágicos

Todo string repetido es una constante nombrada. Literales en la lógica:
prohibidos.

### 0.4 Idioma

Código en inglés; documentación y comentarios de prosa en español.

---

## 1. `types.go` — los anchos nuevos

Cambia **sólo los valores** de las constantes y los comentarios que los citan:

```go
const (
	VariantS Variant = 1 << iota // 1 — 480px  grillas y tarjetas
	VariantM                     // 2 — 1024px telefono a pantalla completa, tablet
	VariantL                     // 4 — 1600px escritorio
)

// Widths for responsive variants in pixels.
const (
	WidthS = 480
	WidthM = 1024
	WidthL = 1600
)
```

`Variant.Suffix()` y `Variant.Width()` no cambian de cuerpo: `Width()` ya
devuelve estas constantes.

---

## 2. `min/convert.go` — consumir `Variant.Width()`, no repetirlo

### 2.1 Borrar la tabla con anchos literales

Hoy, en `ProcessImage`:

```go
variants := []struct {
	v     image.Variant
	width int
}{
	{image.VariantS, 640},
	{image.VariantM, 1024},
	{image.VariantL, 1920},
}
```

Reemplázala por una lista de variantes sin anchos, y toma el ancho de la propia
variante dentro del bucle:

```go
variants := []image.Variant{image.VariantS, image.VariantM, image.VariantL}

for _, v := range variants {
	if src.Variants&v == 0 {
		continue
	}
	targetWidth := v.Width()
	// ...
}
```

Ajusta el resto del cuerpo del bucle a la variable `v` (hoy es `vInfo.v`) y a
`targetWidth` (hoy `vInfo.width`). La línea que arma el nombre queda:

```go
outputName := fmt.Sprintf("%s.%s%s", src.BaseName, v.Suffix(), ext)
```

**Verificación:** `grep -n "640\|1920" min/convert.go` → vacío.

### 2.2 El aviso de fuente angosta

La rama que hoy omite el resize:

```go
if originalWidth <= vInfo.width {
	log("image smaller than target, skipping resize", src.BaseName, vInfo.v)
	processedImg = img
}
```

pasa a avisar del problema real — que el `srcset` va a sobredeclarar el archivo.
Ojo con la condición: se **omite** el resize cuando la fuente no es más ancha
(`<=`), pero sólo se **avisa** cuando es estrictamente más angosta (`<`); si es
exactamente igual el descriptor es honesto y no hay nada que reportar.

```go
if originalWidth <= targetWidth {
	if originalWidth < targetWidth {
		log(MsgSourceNarrowerThanVariant, src.BaseName, v.Suffix(), originalWidth, targetWidth)
	}
	processedImg = img
} else {
	processedImg = imaging.Resize(img, targetWidth, 0, imaging.Lanczos)
}
```

Declara la constante en `min/convert.go`, exportada, con este texto **literal**:

```go
// MsgSourceNarrowerThanVariant avisa que la fuente no alcanza el ancho de la
// variante declarada. El archivo se escribe igual, pero Responsive() lo va a
// declarar en el srcset con el ancho nominal de la variante, no con el real:
// el navegador entonces la elige creyendo que recibe mas pixeles de los que hay.
//
// Es un defecto del contenido (una fuente demasiado chica), no del render, y
// por eso se ataja aqui: el builder compila a WASM y no puede medir el archivo.
const MsgSourceNarrowerThanVariant = "source narrower than variant width; srcset will overstate it"
```

El string `"image smaller than target, skipping resize"` **desaparece**.
**Verificación:** `grep -rn "skipping resize" .` → vacío.

### 2.3 Calidad por defecto 62

En `writeJPEG`:

```go
if quality <= 0 {
	quality = 75
}
```

pasa a usar una constante exportada del paquete `min`:

```go
// DefaultQuality es la calidad JPEG cuando Config.Quality no la fija.
//
// Es 62 y no 75 porque con la escalera 480/1024/1600 ninguna variante se
// muestra 1:1 en una pantalla moderna: siempre hay downscale (DPR >= 2) o un
// leve upscale (un monitor de 1920 recibiendo la variante de 1600). Ese
// remuestreo destruye los artefactos JPEG antes de que el ojo los alcance, y
// la calidad es la palanca de peso mas grande que existe — mayor que cualquier
// ajuste de ancho.
const DefaultQuality = 62
```

```go
if quality <= 0 {
	quality = DefaultQuality
}
```

**Verificación:** `grep -n "= 75" min/convert.go` → vacío.

---

## 3. Tests

### 3.1 Actualizar los que fijan los anchos viejos

| Archivo | Línea aprox. | Cambio |
|---|---|---|
| `tests/types_test.go` | 49-57 | `640` → `480`, `1920` → `1600` (valores y mensajes `want`) |
| `tests/builders_test.go` | 67 | `srcset='/img/foto.S.jpg 480w, /img/foto.M.jpg 1024w, /img/foto.L.jpg 1600w'` |
| `tests/builders_test.go` | 93 | `srcset='/img/v1.2/foto.S.jpg 480w, /img/v1.2/foto.M.jpg 1024w, /img/v1.2/foto.L.jpg 1600w'` |
| `builders.go` | 33 | el ejemplo del comentario de `Responsive`: `480w`, `1024w`, `1600w` |

En `tests/convert_test.go`, `TestConvertPhotographicCompression` sólo tiene
comentarios obsoletos — corrige `// L variant (1920px)` → `(1600px)` y
`// S variant (640px)` → `(480px)`. **No cambies los umbrales de 300 KB y
100 KB ni la calidad 82 que pasa el test**: con anchos menores a la misma
calidad los archivos sólo pueden ser más chicos, así que el test sigue en verde
y sigue siendo un techo válido.

### 3.2 Tests nuevos

Van en `tests/convert_test.go`, siguiendo el estilo de los que ya están ahí
(usan `newTestEnv(t)`, `createTestImage(path, w, h)` y `min.ProcessImage`).

**`TestVariantWidthIsSingleSourceOfTruth`** — el guardián del defecto #1.
Procesa una fuente de 3000x2000 con `image.AllVariants`, abre cada archivo de
salida con `imaging.Open` y verifica que su ancho real es exactamente
`image.VariantS.Width()`, `image.VariantM.Width()` y `image.VariantL.Width()`.
Debe leer los anchos esperados **desde los métodos**, nunca de literales: así,
si alguien vuelve a hardcodear anchos en `convert.go`, el test falla.

**`TestConvertWarnsOnNarrowSource`** — procesa una fuente de 800x600 con
`image.VariantL` y captura los mensajes del `log` en un slice. Verifica que
alguno contiene `min.MsgSourceNarrowerThanVariant`. Con una fuente de 3000x2000
y `image.VariantS` el mensaje **no** debe aparecer.

**`TestDefaultQualityApplied`** — procesa la misma fuente dos veces: una con
`quality` 0 y otra con `quality` 95, a directorios de salida distintos. El
archivo generado con 0 debe ser **estrictamente menor** que el de 95, lo que
prueba que el default se aplicó y es más agresivo que 95. No asserta un tamaño
absoluto (depende de la imagen sintética).

---

## 4. Documentación

- **`docs/ARCHITECTURE.md`**: el contenido ya está escrito. Tu única tarea es
  **borrar la nota provisional** que dice
  `> STATUS (eliminar esta nota cuando la escalera esté implementada): ...`
  (dos líneas, blockquote, bajo el título "La escalera de anchos: 480 / 1024 / 1600").
  Verifica además que las tablas y los anchos del documento coinciden con el
  código que dejaste; si algo no calza, **el código manda sobre el plan pero el
  documento manda sobre ambos** — avisa en el PR en vez de cambiar el diseño.
  **Verificación:** `grep -n "STATUS (eliminar" docs/ARCHITECTURE.md` → vacío.

- **`README.md`**: en la sección `## Render (Frontend/WASM)`, después del ejemplo
  de `Responsive`, agrega una línea que documente la escalera y el default:

  ```
  // Variantes generadas: S=480px, M=1024px, L=1600px (calidad JPEG 62 por defecto).
  // sizes importa: sin el, el navegador asume 100vw y baja de mas en cualquier
  // imagen que no ocupe el ancho completo.
  ```

  No toques la sección de `browser` (líneas ~21-24), que habla de `MaxEdge` y es
  otro concepto.

- Si escribes diagramas: **nunca uses `subgraph`** (rompe el renderizado en el
  TUI). `flowchart TD` y `<br/>` para los saltos.

---

## 5. Etapas

| # | Etapa | Archivos | Resultado |
|---|---|---|---|
| 1 | Anchos nuevos | `types.go` | `WidthS=480`, `WidthL=1600`, comentarios al día |
| 2 | Fuente única de verdad | `min/convert.go` | sin literales de ancho; usa `v.Width()` |
| 3 | Aviso de fuente angosta | `min/convert.go` | `MsgSourceNarrowerThanVariant` |
| 4 | Calidad por defecto | `min/convert.go` | `DefaultQuality = 62` |
| 5 | Tests actualizados | `tests/types_test.go`, `tests/builders_test.go`, `tests/convert_test.go`, `builders.go` | verdes con los anchos nuevos |
| 6 | Tests nuevos | `tests/convert_test.go` | 3 tests de §3.2 |
| 7 | Documentación | `docs/ARCHITECTURE.md`, `README.md` | nota STATUS borrada, README al día |

---

## 6. Criterios de aceptación

- [ ] `go vet ./...` limpio.
- [ ] `go test ./tests/...` en verde, incluidos los 3 tests nuevos.
- [ ] `GOOS=js GOARCH=wasm go build ./...` sin errores.
- [ ] `go build ./...` limpio.
- [ ] `grep -n "640\|1920" min/convert.go` → vacío.
- [ ] `grep -rn "skipping resize" .` → vacío.
- [ ] `grep -n "= 75" min/convert.go` → vacío.
- [ ] `grep -n "STATUS (eliminar" docs/ARCHITECTURE.md` → vacío.
- [ ] `grep -n "WidthS = 480\|WidthM = 1024\|WidthL = 1600" types.go` → tres líneas.
- [ ] `grep -n "DefaultMaxEdge = 1920" browser/compress.go` → **sigue existiendo**
      (no se tocó el compresor del navegador).
- [ ] `head -1 types.go builders.go` → **ninguno** empieza con `//go:build`.
- [ ] `grep -n "\"fmt\"\|\"strings\"\|\"errors\"\|\"strconv\"\|\"log\"" types.go builders.go` → vacío.
- [ ] `grep -n "map\[" types.go builders.go` → vacío.
- [ ] `git diff --stat` no muestra cambios en las firmas de `Img`, `Responsive`,
      `Sizes`, `Srcset`, `Picture` ni `Source`.

## 7. Fuera de alcance

- Tocar `browser/compress.go` o su `DefaultMaxEdge` — es compresión pre-subida,
  no la escalera de render.
- Calidad por variante (S/M más agresiva que L): se evaluó y se descartó a favor
  de un único default, porque con esta escalera ninguna variante se muestra 1:1.
- Agregar un cuarto tier, AVIF, art direction, o `<picture>`.
- Hacer que `Responsive()` conozca el ancho real de los archivos: requiere
  metadata en tiempo de render que hoy no cruza a WASM. El aviso de §2.2 es la
  contención acordada.
