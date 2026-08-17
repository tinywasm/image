---
PLAN: "fix: variantes de fotografías salen en WebP sin pérdida — 10-20x más grandes de lo esperado"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 3599838635258186320
---

> Se despacha con el flujo CodeJob. Ver skill: agents-workflow.

# Plan — codificación con pérdida para contenido fotográfico

## El bug, medido

`veltylabs/mjosefa-website` declaró 7 fotografías vía `RenderImages()` con
`image.AllVariants` y `Quality: 82`. El resultado, medido en disco tras
`sitec build`:

| Fichero | Tamaño |
|---|---|
| `PA180022.L.webp` | **5.5 MB** |
| `PA180022.M.webp` | 1.58 MB |
| `PA180022.S.webp` | 602 KB |
| ... (6 fotos más, mismo patrón) | |

Ninguna variante de ninguna foto baja de 300 KB — ni siquiera la `S` (640px).
El directorio `img/` completo pesa **45 MB**, prácticamente lo mismo que los
7 JPEG originales sin optimizar (29 MB) que este pipeline existe para
reemplazar.

## Causa raíz

`min/convert.go`, función `writeWebP`:

```go
// Note: github.com/HugoSmits86/nativewebp currently only supports lossless
// WebP encoding (VP8L). The quality parameter is accepted for future
// compatibility but not currently used by the encoder.
return nativewebp.Encode(f, img, nil)
```

`nativewebp` (la única librería WebP del `go.mod`) **sólo sabe codificar sin
pérdida**. El campo `Config.Quality` que `ProcessImage` recibe se ignora por
completo — nunca llegó a usarse. Codificación sin pérdida sobre una fotografía
JPEG real (ruido de sensor, detalle continuo) no comprime casi nada frente al
original: es la razón matemática exacta por la que `PA180022.L.webp` (5.5 MB)
es casi del mismo tamaño que `PA180022.jpg` (5.3 MB) pese a estar redimensionada
a 1920px.

**El resize sí funciona.** `imaging.Resize(img, vInfo.width, 0, imaging.Lanczos)`
corre correctamente — el problema es exclusivamente el códec de salida.

## Por qué no es un ajuste de una línea

No existe una bandera "modo con pérdida" en `nativewebp` — no es un parámetro
que falte pasar, es una capacidad que la librería no tiene (confirmado leyendo
su código fuente: `Encode`/`EncodeAll` no aceptan ningún control de calidad).
Cerrar este hueco significa cambiar de librería o de estrategia de
codificación para el caso fotográfico, con las implicaciones que eso trae para
`convert.go`, `loader.go` (nombres de fichero `.webp`) y quien consuma esos
nombres aguas abajo (`sitec`'s asset routing, las etiquetas `<img>`/`<picture>`
que arman `components`).

## Recomendación: JPEG con pérdida para fotografías, WebP sin pérdida se queda donde ya tiene sentido

`min` corre **sólo en build time**, en la máquina del desarrollador o el
servidor de publicación — nunca se compila a WASM (no hay ningún `//go:build
wasm` en este árbol; de hecho, ni siquiera hay `//go:build !wasm`, lo cual vale
la pena añadir en esta misma ronda como higiene, para que quede imposible que
un futuro cambio arrastre esto al binario del navegador por error).

Esto significa que **cgo no está prohibido aquí** de la misma forma que lo
está en el cliente WASM. Pero antes de alcanzar una dependencia de sistema
(libwebp vía cgo), hay una opción más barata y ya disponible: la stdlib de Go
trae `image/jpeg`, con un `Quality` que **sí funciona**. JPEG con pérdida a
calidad 75-85 es, para fotografías, 10-20x más pequeño que WebP sin pérdida, y
es exactamente lo que casi todo sitio web sirve hoy para fotos reales — WebP
gana sobre JPEG con pérdida, pero la brecha (~25-35%) es irrelevante comparada
con la que hay que cerrar primero (10-20x).

**No se propone tocar el caso donde WebP sin pérdida SÍ tiene sentido**: PNG
con transparencia, capturas de pantalla, iconografía — contenido con bordes
duros y pocas transiciones de color, donde sin-pérdida no penaliza tanto y la
transparencia importa. Ese camino se queda como está.

Diseño propuesto:

1. **Detectar si la fuente tiene transparencia real** (canal alpha con algún
   píxel no-opaco) al decodificar con `imaging.Open`. Si la tiene → sigue el
   camino actual (WebP sin pérdida, `nativewebp`). Si no la tiene (el caso de
   toda fotografía JPEG, que nunca tiene alpha) → nuevo camino con pérdida.
2. **Nuevo camino con pérdida**: `image/jpeg.Encode(w, img, &jpeg.Options{Quality: quality})`
   — reutiliza el `Config.Quality` que ya existe y que hoy se ignora. Nombre de
   fichero: mismo patrón (`{base}.{S|M|L}`) pero con extensión `.jpg` en vez de
   `.webp` para este camino, para que sea imposible abrir un `.webp` con bytes
   JPEG dentro por error.
3. **`IsUpToDate` y `cleanOrphans`** (`loader.go`) deben reconocer ambas
   extensiones al decidir si una variante ya existe o quedó huérfana — hoy
   asumen `.webp` a secas.
4. Quien arma las etiquetas `<img>`/`<picture>` (en `components`, aguas abajo)
   necesita saber qué extensión emitió cada variante. Revisa cómo
   `ProcessImage` comunica el nombre de fichero (`outputFiles []string`, ya
   devuelve el nombre completo con extensión) — si eso ya basta para que
   `sitec`/`components` no tengan que adivinar la extensión, no hace falta
   tocarlos; si algo asume `.webp` a mano en otro punto de la cadena, repórtalo
   aquí en vez de parchear allá.

Si al implementar esto aparece una razón concreta para preferir cgo+libwebp
sobre JPEG con pérdida (por ejemplo, que la brecha de tamaño extra sí importe
para este caso de uso), coméntalo en el PR con la medición — no lo decidas
tú solo silenciosamente; es una dependencia de sistema nueva y vale la pena
discutirla.

## Restricciones

- No cambies el resize (`imaging.Resize`, los anchos 640/1024/1920) — funciona
  correctamente, el bug está sólo en la codificación final.
- No toques `sitec` ni `components` para "hacerlo funcionar" — si necesitan un
  cambio, ese cambio es tuyo que hacer en sus propios repos con su propio PR,
  nunca un parche vendoriado. Si el hueco es de otro repo, repórtalo en el PR
  de este plan y detente ahí.
- Añade `//go:build !wasm` a los ficheros de este paquete que no lo tengan —
  documenta explícitamente que nunca se compila al binario del navegador.

## Verificación

- Test con una fotografía real (sin alpha) de tamaño comparable a
  `PA180022.jpg` (varios MB, resolución de cámara real — usa una imagen de
  prueba generada o una libre de derechos, no la del sitio): la variante `L`
  (1920px) debe pesar **menos de 300 KB**, y la `S` (640px) menos de 100 KB.
- Test con una fuente que sí tiene transparencia real: sigue saliendo por el
  camino WebP sin pérdida existente, sin regresión.
- `IsUpToDate`/`cleanOrphans` reconocen ambas extensiones: una variante `.jpg`
  ya generada no se re-procesa en cada build, y no se marca como huérfana.
- Suite completa verde (`go build`, `go vet`, `go test -race`).
