package image

import (
	"github.com/tinywasm/dom"
	"github.com/tinywasm/fmt"
)

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
