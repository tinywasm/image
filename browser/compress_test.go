//go:build wasm

package browser_test

import (
	"syscall/js"
	"testing"

	"github.com/tinywasm/image/browser"
)

// Helper para crear un canvas y devolverlo como Blob simulado/real en el JS environment (p.ej. nodejs / browser).
func createTestImageBlob(width, height int, mimeType string) js.Value {
	// Usamos HTMLCanvasElement o OffscreenCanvas para generar un blob sintético si está disponible
	offscreen := js.Global().Get("OffscreenCanvas")
	if offscreen.Truthy() {
		canvas := offscreen.New(width, height)
		ctx := canvas.Call("getContext", "2d")
		ctx.Set("fillStyle", "red")
		ctx.Call("fillRect", 0, 0, width, height)
		// Si convertToBlob está disponible de forma síncrona/mock en Node or env
		// Pero en test podemos usar canvas
		_ = canvas
	}
	return js.Undefined()
}

func TestCompress_OffscreenCanvasMissing(t *testing.T) {
	origCanvas := js.Global().Get("OffscreenCanvas")
	js.Global().Set("OffscreenCanvas", js.Undefined())
	defer js.Global().Set("OffscreenCanvas", origCanvas)

	badFile := js.ValueOf("not_a_file")
	_, err := browser.Compress(badFile, browser.Config{})
	if err == nil {
		t.Fatal("se esperaba error ErrUnsupported, se obtuvo nil")
	}
	if err != browser.ErrUnsupported {
		t.Fatalf("se esperaba ErrUnsupported, se obtuvo: %v", err)
	}
}

func TestCompress_ConfigDefaults(t *testing.T) {
	// Verificar que las constantes estén definidas correctamente
	if browser.DefaultQuality != 0.85 {
		t.Errorf("DefaultQuality esperado 0.85, obtenido %f", browser.DefaultQuality)
	}
	if browser.DefaultType != "image/webp" {
		t.Errorf("DefaultType esperado 'image/webp', obtenido '%s'", browser.DefaultType)
	}
	if browser.DefaultMaxEdge != 1920 {
		t.Errorf("DefaultMaxEdge esperado 1920, obtenido %d", browser.DefaultMaxEdge)
	}
}
