---
PLAN: "feat(min): las imágenes procesadas se exponen como artefactos servibles"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 17032959379592064886
PR: https://github.com/tinywasm/image/pull/6
---

> Este plan se despacha con el flujo CodeJob. Ver skill: agents-workflow.
> Es autocontenido: no necesitas leer nada fuera de este repositorio.

# Plan — las imágenes procesadas se exponen como artefactos servibles

## Contexto: qué hace este repo

`github.com/tinywasm/image` declara imágenes y `image/min` (build tag
`//go:build !wasm`) las convierte. Un módulo declara sus imágenes en un
`image.go` con un literal compuesto que se lee por AST:

```go
func RenderImages() []image.Asset {
	return []image.Asset{
		{Path: "img/foto.jpg", Variants: image.VariantM, Alt: "Una foto"},
		{Path: "img/logo.svg", Alt: "El logo"},
	}
}
```

`min.Handler.LoadImages()` recorre los módulos, convierte cada origen y escribe
el resultado en `Config.OutputDir`. Un origen ráster produce un archivo por
variante declarada, con sufijo (`foto.M.jpg`); un vector se entrega tal cual,
con su propio nombre (`logo.svg`).

**Este repo es herramienta de backend y usa la biblioteca estándar
legítimamente.** No "corrijas" los imports a `tinywasm/fmt` ni equivalentes:
`min` nunca se compila a WASM.

## El problema que resuelve este plan

El consumidor de `min` es un servidor de desarrollo que sirve el sitio **desde
memoria**. Hoy no puede servir imágenes: `min` las escribe a disco y no ofrece
ninguna forma de obtener su contenido. El consumidor termina obligado a crear
un directorio de salida dentro del proyecto del usuario solo para que exista un
archivo que servir.

Además `Config.OutputDir` cumple hoy dos papeles a la vez: es la **caché** de
conversión (evita re-codificar en cada arranque) y es la **salida del sitio**.
Son cosas distintas y este plan las separa.

## Paso 1 — `outputNames`, una sola fuente para los nombres de salida

En `min/loader.go`, `cleanOrphans` deriva hoy los nombres de salida de un asset
en línea. Extrae esa derivación a una función y usa la función en `cleanOrphans`.

Crea en `min/loader.go`:

```go
// outputNames son los archivos que un asset produce en la salida. Es la única
// derivación de nombres del paquete: si "¿qué archivos vigentes tiene este
// asset?" y "¿qué archivos hay que servir?" respondieran por separado, un
// archivo podría estar vigente para la limpieza e invisible para el servidor.
func outputNames(asset ParsedAsset) []string {
	if IsVector(asset.AbsPath) {
		return []string{VectorOutputName(asset.AbsPath)}
	}

	ext := ExpectedExt(asset.AbsPath)
	if ext == "" {
		return nil
	}

	variantInfos := []struct {
		v image.Variant
		s string
	}{
		{image.VariantS, "S"},
		{image.VariantM, "M"},
		{image.VariantL, "L"},
	}
	var out []string
	for _, vi := range variantInfos {
		if asset.Variants&vi.v != 0 {
			out = append(out, fmt.Sprintf("%s.%s%s", asset.BaseName, vi.s, ext))
		}
	}
	return out
}
```

Reescribe `cleanOrphans` para que construya `activeFiles` llamando a
`outputNames`. **Conserva** su comportamiento actual cuando `ExpectedExt`
devuelve `""` (origen ilegible): en ese caso hoy acepta `.jpg` y `.webp` para
no borrar salida sobre la que no puede razonar. Mantén esa rama tal cual dentro
de `cleanOrphans`, no dentro de `outputNames`.

Criterio: `grep -n "VariantS, \"S\"" min/loader.go` devuelve como máximo dos
apariciones (una en `outputNames`, una en la rama de origen ilegible).

## Paso 2 — `Artifact` y `Handler.Artifacts()`

Crea el archivo nuevo `min/artifacts.go`:

```go
//go:build !wasm

package min

// Artifact es una imagen ya procesada, lista para servirse o para escribirse.
// Path es la ruta relativa a la raíz del sitio y ES la URL: "img/foto.M.jpg".
type Artifact struct {
	Path      string
	Mediatype string
	Content   []byte
}

// imgDir es el prefijo bajo el que se sirve toda imagen procesada.
const imgDir = "img"

// Mediatypes de las tres codificaciones que este paquete produce.
const (
	MediatypeJPEG = "image/jpeg"
	MediatypeWebP = "image/webp"
	MediatypeSVG  = "image/svg+xml"
)

// Artifacts devuelve el contenido de cada imagen producida por el último
// LoadImages/ReloadModule.
//
// Existe porque el consumidor de desarrollo sirve el sitio desde memoria. Sin
// esto, la única forma de servir una imagen era que este paquete la dejara en
// un directorio dentro del proyecto del usuario, y ese directorio pasaba a ser
// una segunda salida del sitio que nadie pidió.
//
// Un origen cuya salida no se puede leer se omite en silencio: no es un error
// del consumidor, y el log ya reportó el fallo de conversión.
func (h *Handler) Artifacts() []Artifact { ... }
```

Implementación:

- `Handler` gana un campo `assets []ParsedAsset` protegido por el `h.mu`
  existente. `LoadImages` lo **reemplaza** con la lista completa que ya
  acumula en su variable `allAssets`. `ReloadModule` **mezcla**: quita los
  assets cuyo `AbsPath` esté bajo el `moduleDir` recargado y añade los nuevos.
- `Artifacts()` toma el lock de lectura, y para cada asset y cada nombre de
  `outputNames(asset)` lee `filepath.Join(h.config.OutputDir, name)`. Si
  `os.ReadFile` falla, omite ese archivo y continúa.
- `Path` es `imgDir + "/" + name` (usa `path.Join`, no `filepath.Join`: es una
  URL, no una ruta del sistema).
- `Mediatype` sale de la extensión del nombre mediante una función
  `mediatypeFor(name string) string`: `.jpg` → `MediatypeJPEG`, `.webp` →
  `MediatypeWebP`, `.svg` → `MediatypeSVG`, cualquier otra → `""`. Un artefacto
  con mediatype `""` **no se devuelve**.

## Paso 3 — `DefaultCacheDir`: la caché sale del proyecto

Crea en `min/artifacts.go`:

```go
// cacheNamespace es el subdirectorio del caché de usuario donde viven las
// imágenes convertidas de todos los proyectos.
const cacheNamespace = "tinywasm/images"

// DefaultCacheDir es dónde guardar las conversiones de un proyecto cuando la
// salida NO es un entregable: un directorio por proyecto bajo el caché del
// usuario, nunca dentro del proyecto.
//
// La conversión es cara y merece caché entre arranques; lo que no merece es
// que esa caché viva dentro del repositorio del usuario, donde se confunde con
// la salida publicable del sitio.
func DefaultCacheDir(rootDir string) (string, error) { ... }
```

Implementación: `os.UserCacheDir()`, y dentro
`cacheNamespace/<clave>/img`, donde `<clave>` es el SHA-256 hexadecimal de
`rootDir` truncado a 16 caracteres (evita colisiones y rutas ilegales sin
depender de la forma de la ruta). Si `os.UserCacheDir` falla, devuelve el error
tal cual envuelto: `fmt.Err("min: no hay directorio de caché de usuario:", err)`.

> `fmt` aquí es `github.com/tinywasm/fmt`, que este repo ya usa en otros
> archivos; si el archivo importa el `fmt` estándar, usa `fmt.Errorf` con el
> mismo texto. Resuelve cuál es por el bloque de imports del archivo, no por el
> texto del selector.

## Paso 4 — test con forma de consumidor

Una API no está publicada hasta que un test **con la forma del consumidor**,
dentro de esta librería, la demuestra. El consumidor real hace: declarar
imágenes → `LoadImages()` → servir `Artifacts()` por HTTP.

Añade a `tests/loader_test.go`:

```go
// TestArtifactsSirvenElSitioSinTocarElProyecto tiene la forma exacta del
// consumidor: un servidor de desarrollo que responde peticiones desde memoria.
// Si Artifacts() no devuelve el contenido, la única alternativa del consumidor
// es escribir un directorio de salida dentro del proyecto del usuario.
func TestArtifactsSirvenElSitioSinTocarElProyecto(t *testing.T) { ... }
```

El test debe:

1. Crear un origen ráster (usa el helper existente `env.createTinyImage`) y un
   origen vectorial (escribe un `.svg` a mano, como hace
   `TestDeclaredVectorReachesOutput`).
2. Declararlos con `env.writeImageGoWithImages`.
3. Llamar a `env.Handler.LoadImages()`.
4. Montar un `http.ServeMux` cuyo handler busque el path de la petición entre
   los `Artifacts()`, escriba `Content-Type` desde `Mediatype` y responda el
   `Content`; usar `httptest.NewServer`.
5. Pedir `/img/<nombre ráster>` y `/img/<nombre vector>` y afirmar, para cada
   uno: código 200, `Content-Type` correcto (`image/jpeg` y `image/svg+xml`), y
   cuerpo **byte a byte igual** al archivo que hay en `env.OutputDir`.

Añade además:

```go
// TestDefaultCacheDirQuedaFueraDelProyecto
func TestDefaultCacheDirQuedaFueraDelProyecto(t *testing.T) { ... }
```

Afirma que `min.DefaultCacheDir(root)` devuelve una ruta que **no** tiene a
`root` como prefijo, y que dos raíces distintas dan rutas distintas.

## Reglas de código

- **Nada de literales repetidos.** Todo string que aparezca más de una vez
  (`"img"`, los mediatypes, `"tinywasm/images"`) es una constante con nombre.
  Ya están declaradas arriba: úsalas, no repitas el literal.
- **No inventes API nueva** fuera de lo que este plan nombra: `Artifact`,
  `Artifacts()`, `DefaultCacheDir`, `outputNames`, `mediatypeFor`,
  `MediatypeJPEG/WebP/SVG`, `cacheNamespace`, `imgDir`.
- **No cambies el comportamiento en disco.** `LoadImages` sigue escribiendo en
  `Config.OutputDir` exactamente como hoy. Este plan **añade** una forma de
  leer lo producido; no quita la escritura.
- **No toques `ProcessImage`, `IsUpToDate`, `ExpectedExt`, `IsVector` ni
  `VectorOutputName`.** Son correctos y hay tests que los cubren.
- Todo el código nuevo lleva `//go:build !wasm`, igual que el resto de `min/`.
- Comentarios y documentación en inglés no: **este repo comenta en español en
  los archivos que ya lo hacen y en inglés en los que ya lo hacen**. Sigue el
  idioma del archivo que estés tocando; para archivos nuevos (`min/artifacts.go`)
  usa español, como en los bloques citados arriba.

## Etapas

| # | archivo | entrega | criterio de aceptación |
|---|---|---|---|
| 1 | `min/loader.go` | `outputNames` + `cleanOrphans` la usa | los tests existentes de limpieza siguen verdes |
| 2 | `min/artifacts.go` | `Artifact`, `Artifacts()`, mediatypes | `go build ./...` |
| 3 | `min/artifacts.go` | `DefaultCacheDir` | `grep -rn "UserCacheDir" min/` devuelve una sola aparición |
| 4 | `tests/loader_test.go` | los dos tests nuevos | `go test ./...` verde |

Cierre: `go vet ./...` y `go test -race ./...` en verde, y `Artifacts()`
devuelve exactamente un elemento por archivo que `LoadImages` dejó en
`OutputDir`.
