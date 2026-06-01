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
	got := Picture(
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
