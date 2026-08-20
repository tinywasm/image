//go:build !wasm

package image_test

import (
	"testing"

	"github.com/tinywasm/image"
)

func TestVariantBitmask(t *testing.T) {
	if image.AllVariants != (image.VariantS | image.VariantM | image.VariantL) {
		t.Errorf("AllVariants should be a combination of S, M, and L")
	}
}

func TestVariantHasS(t *testing.T) {
	if image.AllVariants&image.VariantS == 0 {
		t.Errorf("AllVariants should include VariantS")
	}
	if image.VariantS&image.VariantM != 0 {
		t.Errorf("VariantS and VariantM should not overlap")
	}
}

func TestVariantZeroValue(t *testing.T) {
	v := image.Variant(0)
	if v&image.VariantS != 0 || v&image.VariantM != 0 || v&image.VariantL != 0 {
		t.Errorf("Zero value Variant should not match any variant")
	}
}

func TestVariantSuffix(t *testing.T) {
	if image.VariantS.Suffix() != "S" {
		t.Errorf("VariantS.Suffix() = %q, want %q", image.VariantS.Suffix(), "S")
	}
	if image.VariantM.Suffix() != "M" {
		t.Errorf("VariantM.Suffix() = %q, want %q", image.VariantM.Suffix(), "M")
	}
	if image.VariantL.Suffix() != "L" {
		t.Errorf("VariantL.Suffix() = %q, want %q", image.VariantL.Suffix(), "L")
	}
	if image.Variant(0).Suffix() != "" {
		t.Errorf("Variant(0).Suffix() = %q, want empty string", image.Variant(0).Suffix())
	}
}

func TestVariantWidth(t *testing.T) {
	if image.VariantS.Width() != 640 {
		t.Errorf("VariantS.Width() = %d, want 640", image.VariantS.Width())
	}
	if image.VariantM.Width() != 1024 {
		t.Errorf("VariantM.Width() = %d, want 1024", image.VariantM.Width())
	}
	if image.VariantL.Width() != 1920 {
		t.Errorf("VariantL.Width() = %d, want 1920", image.VariantL.Width())
	}
	if image.Variant(0).Width() != 0 {
		t.Errorf("Variant(0).Width() = %d, want 0", image.Variant(0).Width())
	}
}
