package image_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/image"
	"github.com/tinywasm/image/min"
	"github.com/tinywasm/modfind"
)

func TestConvertJPGToWebP(t *testing.T) {
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

	_, err := min.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	env.assertWebPExists("test", image.VariantL)
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
	env.assertWebPExists("small", image.VariantL)
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

	env.assertWebPExists("multi", image.VariantS)
	env.assertWebPExists("multi", image.VariantM)
	env.assertWebPNotExists("multi", image.VariantL)
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

	expected := filepath.Join(env.OutputDir, "my-custom-name.S.webp")
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

	_, err := min.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
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
