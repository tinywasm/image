//go:build !wasm

package image_test

import (
	"strings"
	"testing"

	. "github.com/tinywasm/image"
)

func TestImg_Basic(t *testing.T) {
	got := Img("/logo.png", "Logo").AsElement().String()
	if !strings.Contains(got, "src='/logo.png'") {
		t.Error("expected src")
	}
	if !strings.Contains(got, "alt='Logo'") {
		t.Error("expected alt")
	}
}

func TestImg_Void(t *testing.T) {
	got := Img("/a.png", "A").AsElement().String()
	if strings.Contains(got, "</img>") {
		t.Error("img should be void")
	}
}

func TestImg_LazySize(t *testing.T) {
	got := Img("/a.png", "A").Lazy().Size(200, 100).AsElement().String()
	if !strings.Contains(got, "loading='lazy'") {
		t.Error("expected lazy")
	}
	if !strings.Contains(got, "width='200'") {
		t.Error("expected width")
	}
}

func TestPicture_WithSources(t *testing.T) {
	got := Picture().Child(
		Source("hero.webp", "image/webp"),
		Img("/hero.png", "Hero").AsElement(),
	).String()
	if !strings.Contains(got, "<picture") {
		t.Error("expected picture")
	}
	if !strings.Contains(got, "type='image/webp'") {
		t.Error("expected webp source")
	}
}

func TestPicture_Typed(t *testing.T) {
	got := Picture().Child(Source("hero.webp", "image/webp")).String()
	if !strings.Contains(got, "<picture") {
		t.Error("expected <picture>")
	}
	if !strings.Contains(got, "<source srcset='hero.webp'") {
		t.Error("expected <source srcset=...>")
	}
}

func TestResponsive_Basic(t *testing.T) {
	got := Responsive("/img/foto.jpg", "alt").String()
	if !strings.Contains(got, "src='/img/foto.M.jpg'") {
		t.Errorf("expected src='/img/foto.M.jpg', got %s", got)
	}
	expectedSrcset := "srcset='/img/foto.S.jpg 480w, /img/foto.M.jpg 1024w, /img/foto.L.jpg 1600w'"
	if !strings.Contains(got, expectedSrcset) {
		t.Errorf("expected %s, got %s", expectedSrcset, got)
	}
	if !strings.Contains(got, "sizes='100vw'") {
		t.Errorf("expected default sizes='100vw', got %s", got)
	}
	if !strings.Contains(got, "alt='alt'") {
		t.Errorf("expected alt='alt', got %s", got)
	}
}

func TestResponsive_CustomSizes(t *testing.T) {
	custom := "(max-width: 600px) 100vw, 33vw"
	got := Responsive("/img/foto.jpg", "alt").Sizes(custom).String()
	expectedSizes := "sizes='(max-width: 600px) 100vw, 33vw'"
	if !strings.Contains(got, expectedSizes) {
		t.Errorf("expected %s, got %s", expectedSizes, got)
	}
}

func TestResponsive_DotsInDirectory(t *testing.T) {
	got := Responsive("/img/v1.2/foto.jpg", "alt").String()
	if !strings.Contains(got, "src='/img/v1.2/foto.M.jpg'") {
		t.Errorf("expected src='/img/v1.2/foto.M.jpg', got %s", got)
	}
	if !strings.Contains(got, "srcset='/img/v1.2/foto.S.jpg 480w, /img/v1.2/foto.M.jpg 1024w, /img/v1.2/foto.L.jpg 1600w'") {
		t.Errorf("expected srcset with v1.2, got %s", got)
	}
}

func TestResponsive_NoExtension(t *testing.T) {
	got := Responsive("/img/foto", "alt").String()
	if !strings.Contains(got, "src='/img/foto'") {
		t.Errorf("expected src='/img/foto', got %s", got)
	}
	if strings.Contains(got, "srcset") {
		t.Errorf("expected no srcset attribute when extension is missing, got %s", got)
	}
}

func TestResponsive_Chaining(t *testing.T) {
	got := Responsive("/img/foto.jpg", "alt").Lazy().Class("x").String()
	if !strings.Contains(got, "loading='lazy'") {
		t.Errorf("expected loading='lazy', got %s", got)
	}
	if !strings.Contains(got, "class='x'") {
		t.Errorf("expected class='x', got %s", got)
	}
}

func TestImg_NoSrcsetRegression(t *testing.T) {
	got := Img("/img/foto.jpg", "alt").String()
	if strings.Contains(got, "srcset") {
		t.Errorf("Img() should not produce srcset, got %s", got)
	}
}
