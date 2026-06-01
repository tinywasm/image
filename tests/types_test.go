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
