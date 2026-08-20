//go:build !wasm

package image_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/image"
	"github.com/tinywasm/image/min"
	"github.com/tinywasm/modfind"
)

func TestConvertJPGToJPG(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/test.jpg")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 200, 100)

	asset := min.ParsedAsset{
		AbsPath:  imgPath,
		Variants: image.VariantL,
		Alt:      "Test",
		BaseName: "test",
	}

	outputs, err := min.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	if len(outputs) != 1 || outputs[0] != "test.L.jpg" {
		t.Errorf("expected output [test.L.jpg], got %v", outputs)
	}

	env.assertJPGExists("test", image.VariantL)
}

func TestConvertNoUpscale(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/small.jpg")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 100, 100)

	asset := min.ParsedAsset{
		AbsPath:  imgPath,
		Variants: image.VariantL,
		Alt:      "Small",
		BaseName: "small",
	}

	skipped := false
	_, err := min.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {
		for _, msg := range m {
			if msg == "image smaller than target, skipping resize" {
				skipped = true
			}
		}
	})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	if !skipped {
		t.Error("expected resize to be skipped for small image")
	}
	env.assertJPGExists("small", image.VariantL)
}

func TestConvertVariantSubset(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/multi.jpg")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 200, 100)

	asset := min.ParsedAsset{
		AbsPath:  imgPath,
		Variants: image.VariantS | image.VariantM,
		Alt:      "Multi",
		BaseName: "multi",
	}

	_, err := min.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	env.assertJPGExists("multi", image.VariantS)
	env.assertJPGExists("multi", image.VariantM)
	env.assertJPGNotExists("multi", image.VariantL)
}

func TestConvertOutputNaming(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/naming.jpg")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 100, 100)

	asset := min.ParsedAsset{
		AbsPath:  imgPath,
		Variants: image.VariantS,
		BaseName: "my-custom-name",
	}

	min.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})

	expected := filepath.Join(env.OutputDir, "my-custom-name.S.jpg")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Errorf("expected output file %s to exist", expected)
	}
}

func TestConvertAltDerivedFromFilename(t *testing.T) {
	env := newTestEnv(t)
	// This is also tested in extract_test, but let's verify it works in integration
	imgPath := filepath.Join(env.ModuleDir, "img/my-hero.jpg")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 100, 100)

	// We need writeImageGo for ExtractImages to work in this env
	env.writeImageGo(`
package m
import "github.com/tinywasm/image"
func RenderImages() []image.Asset {
	return []image.Asset{{Path: "img/my-hero.jpg", Variants: image.VariantS}}
}
`)
	assets, _ := min.ExtractImages(env.ModuleDir)
	if len(assets) == 0 {
		t.Fatalf("expected 1 asset, got 0")
	}
	if assets[0].Alt != "my hero" {
		t.Errorf("expected alt 'my hero', got %q", assets[0].Alt)
	}
}

func TestConvertOutputDirCreated(t *testing.T) {
	env := newTestEnv(t)
	newOutputDir := filepath.Join(t.TempDir(), "deep/path/img")
	// newOutputDir does not exist yet

	imgPath := filepath.Join(env.ModuleDir, "test.jpg")
	createTestImage(imgPath, 100, 100)

	// We need to ensure LoadImages or ReloadModule creates it, or ProcessImage
	env.Handler = min.New(&min.Config{
		RootDir:   env.ModuleDir,
		OutputDir: newOutputDir,
		Quality:   82,
	})
	f := modfind.New()
	f.Seed(env.ModuleDir, []modfind.Module{{Dir: env.ModuleDir, Path: "m"}})
	env.Handler.SetFinder(f)

	env.writeImageGoWithImages([]image.Asset{{Path: "test.jpg", Variants: image.VariantS}})

	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule failed: %v", err)
	}

	if _, err := os.Stat(newOutputDir); os.IsNotExist(err) {
		t.Error("expected OutputDir to be created automatically")
	}
}

func TestConvertPNGTransparency(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "transparent.png")
	createTestPNG(imgPath, 100, 100, true)

	asset := min.ParsedAsset{
		AbsPath:  imgPath,
		Variants: image.VariantS,
		BaseName: "transparent",
	}

	outputs, err := min.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	if len(outputs) != 1 || outputs[0] != "transparent.S.webp" {
		t.Errorf("expected transparent PNG output [transparent.S.webp], got %v", outputs)
	}

	env.assertWebPExists("transparent", image.VariantS)
}

func TestConvertQualityRange(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "quality.jpg")
	createTestImage(imgPath, 500, 500)

	asset := min.ParsedAsset{
		AbsPath:  imgPath,
		Variants: image.VariantS,
		BaseName: "quality",
	}

	// Just verify it doesn't error with different qualities
	for _, q := range []int{50, 82} {
		_, err := min.ProcessImage(asset, env.OutputDir, q, func(m ...any) {})
		if err != nil {
			t.Errorf("ProcessImage failed for quality %d: %v", q, err)
		}
	}
}

func TestConvertCorruptImage(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "corrupt.jpg")
	os.WriteFile(imgPath, []byte("not an image"), 0644)

	asset := min.ParsedAsset{
		AbsPath:  imgPath,
		Variants: image.VariantS,
		BaseName: "corrupt",
	}

	_, err := min.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err == nil {
		t.Error("expected error for corrupt image")
	}
}

func TestConvertPhotographicCompression(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "photo.jpg")
	// Create a high resolution 3000x2000 photographic test image
	createTestImage(imgPath, 3000, 2000)

	asset := min.ParsedAsset{
		AbsPath:  imgPath,
		Variants: image.VariantS | image.VariantL,
		Alt:      "Photo",
		BaseName: "photo",
	}

	_, err := min.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// L variant (1920px) should be under 300 KB
	lStat, err := os.Stat(filepath.Join(env.OutputDir, "photo.L.jpg"))
	if err != nil {
		t.Fatalf("photo.L.jpg not found: %v", err)
	}
	if lStat.Size() >= 300*1024 {
		t.Errorf("expected photo.L.jpg size < 300 KB, got %d bytes", lStat.Size())
	}

	// S variant (640px) should be under 100 KB
	sStat, err := os.Stat(filepath.Join(env.OutputDir, "photo.S.jpg"))
	if err != nil {
		t.Fatalf("photo.S.jpg not found: %v", err)
	}
	if sStat.Size() >= 100*1024 {
		t.Errorf("expected photo.S.jpg size < 100 KB, got %d bytes", sStat.Size())
	}
}
