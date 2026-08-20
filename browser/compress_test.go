//go:build wasm

package browser_test

import (
	"bytes"
	"syscall/js"
	"testing"

	"github.com/tinywasm/image/browser"
)

func setupJSMock(t *testing.T) func() {
	t.Helper()
	origCanvas := js.Global().Get("OffscreenCanvas")
	origBitmap := js.Global().Get("createImageBitmap")

	createPromise := func(resolveVal js.Value, rejectErr string) js.Value {
		promiseClass := js.Global().Get("Promise")
		if !promiseClass.Truthy() {
			t.Fatal("Promise class no encontrada en JS global")
		}
		var resolveFunc, rejectFunc js.Value
		executor := js.FuncOf(func(this js.Value, args []js.Value) any {
			resolveFunc = args[0]
			rejectFunc = args[1]
			return nil
		})
		p := promiseClass.New(executor)
		if rejectErr != "" {
			errObj := js.Global().Get("Error").New(rejectErr)
			rejectFunc.Invoke(errObj)
		} else {
			resolveFunc.Invoke(resolveVal)
		}
		return p
	}

	mockImageBitmap := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return createPromise(js.Undefined(), "file es requerido")
		}
		file := args[0]
		if file.Type() == js.TypeObject && file.Get("_invalid").Truthy() {
			return createPromise(js.Undefined(), "formato de archivo no valido")
		}
		w := 800
		h := 600
		if file.Type() == js.TypeObject && file.Get("_width").Truthy() {
			w = file.Get("_width").Int()
			h = file.Get("_height").Int()
		}

		bitmap := js.Global().Get("Object").New()
		bitmap.Set("width", w)
		bitmap.Set("height", h)
		bitmap.Set("close", js.FuncOf(func(this js.Value, args []js.Value) any {
			return nil
		}))
		return createPromise(bitmap, "")
	})
	js.Global().Set("createImageBitmap", mockImageBitmap)

	mockCanvasClass := js.FuncOf(func(this js.Value, args []js.Value) any {
		w := args[0].Int()
		h := args[1].Int()
		canvas := js.Global().Get("Object").New()
		canvas.Set("width", w)
		canvas.Set("height", h)

		canvas.Set("getContext", js.FuncOf(func(this js.Value, args []js.Value) any {
			ctx := js.Global().Get("Object").New()
			ctx.Set("drawImage", js.FuncOf(func(this js.Value, args []js.Value) any {
				return nil
			}))
			return ctx
		}))

		canvas.Set("convertToBlob", js.FuncOf(func(this js.Value, args []js.Value) any {
			opts := args[0]
			mimeType := opts.Get("type").String()
			quality := opts.Get("quality").Float()

			blob := js.Global().Get("Object").New()
			blob.Set("type", mimeType)

			blob.Set("arrayBuffer", js.FuncOf(func(this js.Value, args []js.Value) any {
				dataLen := int(float64(w*h) * quality * 0.01)
				if dataLen < 10 {
					dataLen = 10
				}
				u8 := js.Global().Get("Uint8Array").New(dataLen)
				u8.SetIndex(0, 'R')
				u8.SetIndex(1, 'I')
				u8.SetIndex(2, 'F')
				u8.SetIndex(3, 'F')
				return createPromise(u8.Get("buffer"), "")
			}))

			return createPromise(blob, "")
		}))

		return canvas
	})
	js.Global().Set("OffscreenCanvas", mockCanvasClass)

	return func() {
		js.Global().Set("OffscreenCanvas", origCanvas)
		js.Global().Set("createImageBitmap", origBitmap)
	}
}

func createMockFile(w, h int, invalid bool) js.Value {
	file := js.Global().Get("Object").New()
	file.Set("_width", w)
	file.Set("_height", h)
	file.Set("_invalid", invalid)
	return file
}

// 1. Imagen de 3000x2000, MaxEdge: 1920 -> resultado de 1920x1280
func TestCompress_ResizeLargeImage(t *testing.T) {
	cleanup := setupJSMock(t)
	defer cleanup()

	file := createMockFile(3000, 2000, false)
	res, err := browser.Compress(file, browser.Config{
		MaxEdge: 1920,
		Quality: 0.8,
		Type:    "image/webp",
	})
	if err != nil {
		t.Fatalf("error inesperado en Compress: %v", err)
	}
	if res.Width != 1920 || res.Height != 1280 {
		t.Fatalf("dimensiones esperadas 1920x1280, obtenidas %dx%d", res.Width, res.Height)
	}
}

// 2. Imagen de 800x600, MaxEdge: 1920 -> 800x600, no se amplía
func TestCompress_SmallImageNotEnlarged(t *testing.T) {
	cleanup := setupJSMock(t)
	defer cleanup()

	file := createMockFile(800, 600, false)
	res, err := browser.Compress(file, browser.Config{
		MaxEdge: 1920,
		Quality: 0.8,
		Type:    "image/webp",
	})
	if err != nil {
		t.Fatalf("error inesperado en Compress: %v", err)
	}
	if res.Width != 800 || res.Height != 600 {
		t.Fatalf("dimensiones esperadas 800x600, obtenidas %dx%d", res.Width, res.Height)
	}
}

// 3. Config{} vacío -> usa DefaultQuality, DefaultType, DefaultMaxEdge
func TestCompress_EmptyConfigUsesDefaults(t *testing.T) {
	cleanup := setupJSMock(t)
	defer cleanup()

	file := createMockFile(3000, 2000, false)
	res, err := browser.Compress(file, browser.Config{})
	if err != nil {
		t.Fatalf("error inesperado en Compress: %v", err)
	}
	if res.Type != browser.DefaultType {
		t.Errorf("tipo esperado '%s', obtenido '%s'", browser.DefaultType, res.Type)
	}
	if res.Width != 1920 || res.Height != 1280 {
		t.Errorf("dimensiones esperadas con DefaultMaxEdge (1920) = 1920x1280, obtenidas %dx%d", res.Width, res.Height)
	}
}

// 4. CompressToFit con limite holgado -> un solo intento
func TestCompressToFit_LooseLimit(t *testing.T) {
	cleanup := setupJSMock(t)
	defer cleanup()

	file := createMockFile(1000, 1000, false)
	res, err := browser.CompressToFit(file, 500000, browser.Config{})
	if err != nil {
		t.Fatalf("error inesperado en CompressToFit: %v", err)
	}
	if len(res.Data) > 500000 {
		t.Fatalf("tamaño %d supera el maximo de 500000", len(res.Data))
	}
}

// 5. CompressToFit con limite estrecho -> baja la calidad y cabe o devuelve error tras agotar intentos
func TestCompressToFit_TightLimit(t *testing.T) {
	cleanup := setupJSMock(t)
	defer cleanup()

	file := createMockFile(1000, 1000, false)
	_, err := browser.CompressToFit(file, 5, browser.Config{})
	if err == nil {
		t.Fatal("se esperaba error al no poder comprimir por debajo de 5 bytes, se obtuvo nil")
	}
}

// 6. OffscreenCanvas ausente -> ErrUnsupported devuelto SIN leer/acceder al archivo primero
func TestCompress_OffscreenCanvasMissingOrderCheck(t *testing.T) {
	origCanvas := js.Global().Get("OffscreenCanvas")
	js.Global().Set("OffscreenCanvas", js.Undefined())
	defer js.Global().Set("OffscreenCanvas", origCanvas)

	fileAccessed := false
	proxyHandler := js.Global().Get("Object").New()
	proxyHandler.Set("get", js.FuncOf(func(this js.Value, args []js.Value) any {
		fileAccessed = true
		return js.Undefined()
	}))

	fileProxy := js.Global().Get("Proxy").New(js.Global().Get("Object").New(), proxyHandler)

	_, err := browser.Compress(fileProxy, browser.Config{})
	if err == nil {
		t.Fatal("se esperaba ErrUnsupported, se obtuvo nil")
	}
	if err != browser.ErrUnsupported {
		t.Fatalf("se esperaba ErrUnsupported, se obtuvo: %v", err)
	}
	if fileAccessed {
		t.Fatal("el archivo fue accedido antes de verificar la disponibilidad de OffscreenCanvas")
	}
}

// 7. File que no es una imagen -> error, no panico
func TestCompress_InvalidImageFile(t *testing.T) {
	cleanup := setupJSMock(t)
	defer cleanup()

	file := createMockFile(0, 0, true)
	_, err := browser.Compress(file, browser.Config{})
	if err == nil {
		t.Fatal("se esperaba error para archivo de imagen invalido, se obtuvo nil")
	}
}

// 8. Result.Data devuelto -> bytes de una imagen decodificable
func TestCompress_ValidImageDataReturned(t *testing.T) {
	cleanup := setupJSMock(t)
	defer cleanup()

	file := createMockFile(100, 100, false)
	res, err := browser.Compress(file, browser.Config{})
	if err != nil {
		t.Fatalf("error inesperado en Compress: %v", err)
	}
	if len(res.Data) == 0 {
		t.Fatal("res.Data esta vacio")
	}
	if !bytes.HasPrefix(res.Data, []byte("RIFF")) {
		t.Fatalf("res.Data no contiene la cabecera esperada RIFF: %v", res.Data[:4])
	}
}
