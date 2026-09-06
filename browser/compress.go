//go:build wasm

package browser

import (
	"syscall/js"

	"webtyp.com/await"
	"webtyp.com/fmt"
)

const (
	DefaultQuality = 0.85
	DefaultType    = "image/webp"
	DefaultMaxEdge = 1920
)

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
func Compress(file js.Value, c Config) (Result, error) {
	if !js.Global().Get(jsOffscreenCanvas).Truthy() {
		return Result{}, ErrUnsupported
	}

	if c.MaxEdge == 0 && c.Quality == 0 && c.Type == "" {
		c.MaxEdge = DefaultMaxEdge
	}
	if c.Quality <= 0 {
		c.Quality = DefaultQuality
	}
	if c.Type == "" {
		c.Type = DefaultType
	}

	bitmapVal, err := await.Promise(js.Global().Call(jsCreateImageBitmap, file))
	if err != nil {
		return Result{}, fmt.Errf("image/browser: error al decodificar imagen: %v", err)
	}
	defer bitmapVal.Call(jsClose)

	origW := bitmapVal.Get(jsWidth).Int()
	origH := bitmapVal.Get(jsHeight).Int()

	w, h := origW, origH
	if c.MaxEdge > 0 {
		maxOrig := origW
		if origH > maxOrig {
			maxOrig = origH
		}
		if maxOrig > c.MaxEdge {
			if origW >= origH {
				w = c.MaxEdge
				h = (origH * c.MaxEdge) / origW
			} else {
				h = c.MaxEdge
				w = (origW * c.MaxEdge) / origH
			}
		}
	}

	canvas := js.Global().Get(jsOffscreenCanvas).New(w, h)
	ctx := canvas.Call(jsGetContext, js2D)
	ctx.Call(jsDrawImage, bitmapVal, 0, 0, w, h)

	opts := js.Global().Get("Object").New()
	opts.Set(jsType, c.Type)
	opts.Set(jsQuality, c.Quality)

	blobVal, err := await.Promise(canvas.Call(jsConvertToBlob, opts))
	if err != nil {
		return Result{}, fmt.Errf("image/browser: error al convertir canvas a blob: %v", err)
	}

	bufVal, err := await.Promise(blobVal.Call(jsArrayBuffer))
	if err != nil {
		return Result{}, fmt.Errf("image/browser: error al obtener arrayBuffer: %v", err)
	}

	u8 := js.Global().Get(jsUint8Array).New(bufVal)
	out := make([]byte, u8.Get(jsLength).Int())
	js.CopyBytesToGo(out, u8)

	resType := blobVal.Get(jsType).String()
	if resType == "" {
		resType = c.Type
	}

	return Result{
		Data:   out,
		Type:   resType,
		Width:  w,
		Height: h,
	}, nil
}

// CompressToFit repite Compress bajando la calidad hasta que el resultado
// quepa en maxBytes, o hasta agotar los intentos.
func CompressToFit(file js.Value, maxBytes int, c Config) (Result, error) {
	if c.MaxEdge == 0 && c.Quality == 0 && c.Type == "" {
		c.MaxEdge = DefaultMaxEdge
	}
	if c.Quality <= 0 {
		c.Quality = DefaultQuality
	}

	currentQuality := c.Quality
	minQuality := 0.10
	step := 0.15

	for {
		cfg := c
		cfg.Quality = currentQuality
		res, err := Compress(file, cfg)
		if err != nil {
			return Result{}, err
		}

		if len(res.Data) <= maxBytes {
			return res, nil
		}

		if currentQuality <= minQuality {
			return Result{}, fmt.Errf("image/browser: no se pudo comprimir por debajo de %d bytes (ultimo tamaño: %d bytes)", maxBytes, len(res.Data))
		}

		currentQuality -= step
		if currentQuality < minQuality {
			currentQuality = minQuality
		}
	}
}
