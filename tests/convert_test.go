package imagemin_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/imagemin"
)

func TestConvertJPGToWebP(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/test.jpg")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 200, 100)

	asset := imagemin.ParsedAsset{
		AbsPath:  imgPath,
		Variants: imagemin.VariantL,
		Alt:      "Test",
		BaseName: "test",
	}

	_, err := imagemin.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	env.assertWebPExists("test", imagemin.VariantL)
}

func TestConvertNoUpscale(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/small.jpg")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 100, 100)

	asset := imagemin.ParsedAsset{
		AbsPath:  imgPath,
		Variants: imagemin.VariantL,
		Alt:      "Small",
		BaseName: "small",
	}

	skipped := false
	_, err := imagemin.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {
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
	env.assertWebPExists("small", imagemin.VariantL)
}

func TestConvertVariantSubset(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/multi.jpg")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 200, 100)

	asset := imagemin.ParsedAsset{
		AbsPath:  imgPath,
		Variants: imagemin.VariantS | imagemin.VariantM,
		Alt:      "Multi",
		BaseName: "multi",
	}

	_, err := imagemin.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	env.assertWebPExists("multi", imagemin.VariantS)
	env.assertWebPExists("multi", imagemin.VariantM)
	env.assertWebPNotExists("multi", imagemin.VariantL)
}

func TestConvertOutputNaming(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/naming.jpg")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 100, 100)

	asset := imagemin.ParsedAsset{
		AbsPath:  imgPath,
		Variants: imagemin.VariantS,
		BaseName: "my-custom-name",
	}

	imagemin.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})

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

	assets, _ := imagemin.ExtractImages(env.ModuleDir)
	// We need writeSSRGo for ExtractImages to work in this env
	env.writeSSRGo(`
package m
import "github.com/tinywasm/imagemin"
func RenderImages() []imagemin.Asset {
	return []imagemin.Asset{{Path: "img/my-hero.jpg", Variants: imagemin.VariantS}}
}
`)
	assets, _ = imagemin.ExtractImages(env.ModuleDir)
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
	// Actually, let's see where it's created. 
	// ProcessImage calls writeWebP which calls os.Create. os.Create doesn't create directories.
	// ReloadModule should probably create it.
	
	err := env.Handler.ReloadModule(env.ModuleDir)
	// Wait, we need to set OutputDir in config
	env.Handler = imagemin.New(&imagemin.Config{
		RootDir: env.ModuleDir,
		OutputDir: newOutputDir,
		Quality: 82,
	})
	env.Handler.SetListModulesFn(func(rootDir string) ([]string, error) {
		return []string{env.ModuleDir}, nil
	})
	
	env.writeSSRGoWithImages([]imagemin.Asset{{Path: "test.jpg", Variants: imagemin.VariantS}})
	
	// We need to make sure the directory is created.
	// Let's modify ReloadModule to create the output dir if it doesn't exist.
	
	err = env.Handler.ReloadModule(env.ModuleDir)
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

	asset := imagemin.ParsedAsset{
		AbsPath:  imgPath,
		Variants: imagemin.VariantS,
		BaseName: "transparent",
	}

	_, err := imagemin.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	env.assertWebPExists("transparent", imagemin.VariantS)
}

func TestConvertQualityRange(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "quality.jpg")
	createTestImage(imgPath, 500, 500)

	asset := imagemin.ParsedAsset{
		AbsPath:  imgPath,
		Variants: imagemin.VariantS,
		BaseName: "quality",
	}

	// Just verify it doesn't error with different qualities
	for _, q := range []int{50, 82} {
		_, err := imagemin.ProcessImage(asset, env.OutputDir, q, func(m ...any) {})
		if err != nil {
			t.Errorf("ProcessImage failed for quality %d: %v", q, err)
		}
	}
}

func TestConvertCorruptImage(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "corrupt.jpg")
	os.WriteFile(imgPath, []byte("not an image"), 0644)

	asset := imagemin.ParsedAsset{
		AbsPath:  imgPath,
		Variants: imagemin.VariantS,
		BaseName: "corrupt",
	}

	_, err := imagemin.ProcessImage(asset, env.OutputDir, 82, func(m ...any) {})
	if err == nil {
		t.Error("expected error for corrupt image")
	}
}
