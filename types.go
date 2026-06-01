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

// Asset represents an image declaration in a module.
type Asset struct {
	Path     string  // relative to the module directory: "img/logo.png"
	Variants Variant // e.g., AllVariants, VariantS|VariantM, VariantL
	Alt      string  // SEO alternative text; if empty derived from filename
}
