//go:build wasm

package browser

import (
	"github.com/tinywasm/fmt"
)

// ErrUnsupported indica que el navegador no ofrece OffscreenCanvas y por
// tanto no puede comprimir.
var ErrUnsupported = fmt.Errf("image/browser: el navegador no soporta OffscreenCanvas")
