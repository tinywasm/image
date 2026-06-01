package imagemin_test

import (
	"testing"

	"github.com/tinywasm/imagemin"
)

func TestVariantBitmask(t *testing.T) {
	if imagemin.AllVariants != (imagemin.VariantS | imagemin.VariantM | imagemin.VariantL) {
		t.Errorf("AllVariants should be a combination of S, M, and L")
	}
}

func TestVariantHasS(t *testing.T) {
	if imagemin.AllVariants&imagemin.VariantS == 0 {
		t.Errorf("AllVariants should include VariantS")
	}
	if imagemin.VariantS&imagemin.VariantM != 0 {
		t.Errorf("VariantS and VariantM should not overlap")
	}
}

func TestVariantZeroValue(t *testing.T) {
	v := imagemin.Variant(0)
	if v&imagemin.VariantS != 0 || v&imagemin.VariantM != 0 || v&imagemin.VariantL != 0 {
		t.Errorf("Zero value Variant should not match any variant")
	}
}
