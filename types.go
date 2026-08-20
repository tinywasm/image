package image

// Variant represents a bitmask for responsive image variants.
type Variant uint8

const (
	VariantS Variant = 1 << iota // 1 — 640px  mobile
	VariantM                     // 2 — 1024px tablet
	VariantL                     // 4 — 1920px desktop
)

// AllVariants includes all responsive variants.
const AllVariants = VariantS | VariantM | VariantL

// Widths for responsive variants in pixels.
const (
	WidthS = 640
	WidthM = 1024
	WidthL = 1920
)

// Suffix es la marca que el pipeline intercala antes de la extension:
// "foto.jpg" con VariantM produce "foto.M.jpg".
//
// Vive aqui y no en min/ porque quien ESCRIBE los archivos y quien los
// DECLARA en el HTML tienen que compartir una sola definicion: min/ es
// backend, y el que emite el srcset renderiza tambien en wasm.
func (v Variant) Suffix() string {
	switch v {
	case VariantS:
		return "S"
	case VariantM:
		return "M"
	case VariantL:
		return "L"
	default:
		return ""
	}
}

// Width es el ancho en pixeles al que el pipeline redimensiona esta variante.
// Es el descriptor "w" que el srcset necesita para que el navegador elija.
func (v Variant) Width() int {
	switch v {
	case VariantS:
		return WidthS
	case VariantM:
		return WidthM
	case VariantL:
		return WidthL
	default:
		return 0
	}
}

// Asset represents an image declaration in a module.
type Asset struct {
	Path     string  // relative to the module directory: "img/logo.png"
	Variants Variant // e.g., AllVariants, VariantS|VariantM, VariantL
	Alt      string  // SEO alternative text; if empty derived from filename
}
