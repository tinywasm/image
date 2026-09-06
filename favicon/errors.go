//go:build !wasm

package favicon

import "webtyp.com/fmt"

// Errores de validación de la fuente. Usan webtyp/fmt para mantener
// el binario sin dependencias pesadas.
var (
	ErrNoRaster    = fmt.Err("favicon: falta el raster")
	ErrUndecodable = fmt.Err("favicon: no se pudo decodificar la imagen")
	ErrNotSquare   = fmt.Err("favicon: el logo debe ser cuadrado")
	ErrTooSmall    = fmt.Err("favicon: el logo debe medir al menos 256x256")
)

// wrappedError une un mensaje exacto (el que ve el cliente) con el sentinel
// que permite errors.Is. Implementa Unwrap() []error para compatibilidad con
// Go 1.20+ y con el patrón de webtyp/fmt.wrappedErr.
type wrappedError struct {
	msg      string
	sentinel error
}

func (e *wrappedError) Error() string   { return e.msg }
func (e *wrappedError) Unwrap() []error { return []error{e.sentinel} }

func errNotSquare(w, h int) error {
	msg := fmt.Sprintf("favicon: el logo debe ser cuadrado: %dx%d", w, h)
	return &wrappedError{msg: msg, sentinel: ErrNotSquare}
}

func errTooSmall(w, h int) error {
	msg := fmt.Sprintf("favicon: el logo debe medir al menos %dx%d: %dx%d", MinEdge, MinEdge, w, h)
	return &wrappedError{msg: msg, sentinel: ErrTooSmall}
}
