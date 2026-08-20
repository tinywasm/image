//go:build !wasm

package image_test

import (
	"path/filepath"
	"testing"

	"github.com/tinywasm/image"
	"github.com/tinywasm/image/min"
)

func TestExtractImagesLiteral(t *testing.T) {
	env := newTestEnv(t)
	env.writeImageGo(`
package module
import "github.com/tinywasm/image"
func RenderImages() []image.Asset {
	return []image.Asset{
		{Path: "img/logo.png", Variants: image.VariantS | image.VariantM, Alt: "Logo"},
	}
}
`)

	assets, err := min.ExtractImages(env.ModuleDir)
	if err != nil {
		t.Fatalf("ExtractImages failed: %v", err)
	}

	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}

	if assets[0].BaseName != "logo" {
		t.Errorf("expected BaseName 'logo', got %q", assets[0].BaseName)
	}

	if assets[0].Variants != (image.VariantS | image.VariantM) {
		t.Errorf("expected Variants S|M, got %d", assets[0].Variants)
	}
}

func TestExtractImagesAllVariants(t *testing.T) {
	env := newTestEnv(t)
	env.writeImageGo(`
package module
import "github.com/tinywasm/image"
func RenderImages() []image.Asset {
	return []image.Asset{
		{Path: "hero.jpg", Variants: image.AllVariants},
	}
}
`)

	assets, _ := min.ExtractImages(env.ModuleDir)
	if len(assets) != 1 || assets[0].Variants != image.AllVariants {
		t.Errorf("failed to resolve AllVariants")
	}
}

func TestExtractImagesAltEmpty(t *testing.T) {
	env := newTestEnv(t)
	env.writeImageGo(`
package module
import "github.com/tinywasm/image"
func RenderImages() []image.Asset {
	return []image.Asset{
		{Path: "my-hero.jpg", Variants: image.VariantS},
	}
}
`)

	assets, _ := min.ExtractImages(env.ModuleDir)
	if assets[0].Alt != "my hero" {
		t.Errorf("expected alt 'my hero', got %q", assets[0].Alt)
	}
}

func TestExtractImagesNoRenderImages(t *testing.T) {
	env := newTestEnv(t)
	env.writeImageGo(`
package module
func Other() {}
`)

	assets, err := min.ExtractImages(env.ModuleDir)
	if err != nil {
		t.Fatalf("ExtractImages failed: %v", err)
	}
	if len(assets) != 0 {
		t.Errorf("expected 0 assets, got %d", len(assets))
	}
}

func TestExtractImagesNoImageFile(t *testing.T) {
	env := newTestEnv(t)
	assets, err := min.ExtractImages(env.ModuleDir)
	if err != nil {
		t.Fatalf("ExtractImages failed: %v", err)
	}
	if len(assets) != 0 {
		t.Errorf("expected 0 assets, got %d", len(assets))
	}
}

func TestExtractAbsPathResolution(t *testing.T) {
	env := newTestEnv(t)
	env.writeImageGo(`
package module
import "github.com/tinywasm/image"
func RenderImages() []image.Asset {
	return []image.Asset{
		{Path: "img/logo.png", Variants: image.VariantS},
	}
}
`)

	assets, _ := min.ExtractImages(env.ModuleDir)
	expected := filepath.Join(env.ModuleDir, "img/logo.png")
	if assets[0].AbsPath != expected {
		t.Errorf("expected AbsPath %q, got %q", expected, assets[0].AbsPath)
	}
}

func TestExtractImagesLocalVar(t *testing.T) {
	env := newTestEnv(t)
	env.writeImageGo(`
package module
import "github.com/tinywasm/image"
func RenderImages() []image.Asset {
	assets := []image.Asset{
		{Path: "img/logo.png", Variants: image.VariantS},
	}
	return assets
}
`)

	assets, _ := min.ExtractImages(env.ModuleDir)
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	if assets[0].BaseName != "logo" {
		t.Errorf("expected BaseName 'logo', got %q", assets[0].BaseName)
	}
}
