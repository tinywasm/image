package image

import (
	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

// DefaultSizes is the default sizes attribute value for responsive images.
const DefaultSizes = "100vw"

// ImgElement wraps *dom.Element to provide a fluent image-specific API.
type ImgElement struct {
	el *dom.Element
}

// Img builds an <img> element with src and alt.
func Img(src, alt string) *ImgElement {
	el := dom.NewElement("img").
		Attr("src", src).
		Attr("alt", alt).
		NoCloseTag()
	return &ImgElement{el: el}
}

// Responsive construye un <img> con srcset para las tres variantes que el
// pipeline genera, a partir de la ruta BASE (sin sufijo de variante).
//
//	Responsive("/img/foto.jpg", "Fachada")
//
// emite:
//
//	<img src="/img/foto.M.jpg"
//	     srcset="/img/foto.S.jpg 480w, /img/foto.M.jpg 1024w, /img/foto.L.jpg 1600w"
//	     sizes="100vw" alt="Fachada">
//
// El src apunta a la variante M para que un navegador que ignore srcset reciba
// algo razonable en vez de la version de escritorio.
func Responsive(base, alt string) *ImgElement {
	prefix, ext, ok := splitBaseExt(base)
	if !ok {
		return Img(base, alt)
	}

	srcM := fmt.Sprintf("%s.%s%s", prefix, VariantM.Suffix(), ext)
	srcset := fmt.Sprintf("%s.%s%s %dw, %s.%s%s %dw, %s.%s%s %dw",
		prefix, VariantS.Suffix(), ext, VariantS.Width(),
		prefix, VariantM.Suffix(), ext, VariantM.Width(),
		prefix, VariantL.Suffix(), ext, VariantL.Width(),
	)

	el := dom.NewElement("img").
		Attr("src", srcM).
		Attr("srcset", srcset).
		Attr("sizes", DefaultSizes).
		Attr("alt", alt).
		NoCloseTag()

	return &ImgElement{el: el}
}

func splitBaseExt(base string) (prefix, ext string, ok bool) {
	lastDot := -1
	lastSlash := -1
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '.' && lastDot == -1 {
			lastDot = i
		} else if base[i] == '/' && lastSlash == -1 {
			lastSlash = i
		}
		if lastDot != -1 && lastSlash != -1 {
			break
		}
	}
	if lastDot == -1 || lastDot <= lastSlash {
		return "", "", false
	}
	return base[:lastDot], base[lastDot:], true
}

// Sizes declara al navegador que ancho ocupara la imagen en el layout, para
// que pueda elegir la variante ANTES de conocer el CSS.
func (i *ImgElement) Sizes(s string) *ImgElement {
	i.el.Attr("sizes", s)
	return i
}

// Srcset fija el atributo srcset a mano. Escape hatch para un consumidor con
// variantes que no siguen la convencion; Responsive es el camino normal.
func (i *ImgElement) Srcset(s string) *ImgElement {
	i.el.Attr("srcset", s)
	return i
}

// Lazy sets loading="lazy".
func (i *ImgElement) Lazy() *ImgElement {
	i.el.Attr("loading", "lazy")
	return i
}

// Size sets width and height (reduces CLS).
func (i *ImgElement) Size(w, h int) *ImgElement {
	i.el.Attr("width", fmt.Sprint(w))
	i.el.Attr("height", fmt.Sprint(h))
	return i
}

// Class adds CSS classes.
func (i *ImgElement) Class(classes ...string) *ImgElement {
	i.el.Class(classes...)
	return i
}

// Attr sets an arbitrary attribute.
func (i *ImgElement) Attr(key, val string) *ImgElement {
	i.el.Attr(key, val)
	return i
}

// AsElement returns the underlying *dom.Element for embedding in Render() trees.
func (i *ImgElement) AsElement() *dom.Element {
	return i.el
}

// String serializes the image element (satisfies dom.Component).
func (i *ImgElement) String() string {
	return i.el.String()
}

// Picture builds a <picture> element for responsive images.
func Picture() *dom.Element {
	return dom.NewElement("picture")
}

// Source builds a <source> for use inside <picture>.
// mediaOrType: media query "(max-width: 600px)" or MIME type "image/webp".
func Source(srcset, mediaOrType string) *dom.Element {
	el := dom.NewElement("source").Attr("srcset", srcset).NoCloseTag()
	if len(mediaOrType) > 0 {
		if mediaOrType[0] == '(' {
			el.Attr("media", mediaOrType)
		} else {
			el.Attr("type", mediaOrType)
		}
	}
	return el
}
