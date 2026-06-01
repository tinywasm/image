package image_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/image"
	"github.com/tinywasm/image/min"
)

func TestLoadImagesFromModule(t *testing.T) {
	env := newTestEnv(t)

	imgPath := "img/logo.png"
	env.copyTestImage(imgPath, "gopher.S.png")

	env.writeImageGoWithImages([]image.Asset{
		{Path: imgPath, Variants: image.VariantS, Alt: "Gopher"},
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatalf("LoadImages failed: %v", err)
	}

	env.assertWebPExists("logo", image.VariantS)
}

func TestReloadModuleNewImage(t *testing.T) {
	env := newTestEnv(t)

	env.copyTestImage("img/one.png", "gopher.S.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
	})
	env.Handler.ReloadModule(env.ModuleDir)
	env.assertWebPExists("one", image.VariantS)

	env.copyTestImage("img/two.png", "gopher.S.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
		{Path: "img/two.png", Variants: image.VariantS},
	})

	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule failed: %v", err)
	}

	env.assertWebPExists("one", image.VariantS)
	env.assertWebPExists("two", image.VariantS)
}

func TestReloadModuleRemovedImageDoesNotCleanup(t *testing.T) {
	env := newTestEnv(t)

	env.copyTestImage("img/one.png", "gopher.S.png")
	env.copyTestImage("img/two.png", "gopher.S.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
		{Path: "img/two.png", Variants: image.VariantS},
	})
	env.Handler.ReloadModule(env.ModuleDir)
	env.assertWebPExists("one", image.VariantS)
	env.assertWebPExists("two", image.VariantS)

	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
	})

	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule failed: %v", err)
	}

	env.assertWebPExists("one", image.VariantS)
	// ReloadModule no longer cleans orphans (global cleanup only in LoadImages)
	env.assertWebPExists("two", image.VariantS)
}

func TestGlobalOrphanCleanup(t *testing.T) {
	env := newTestEnv(t)

	env.copyTestImage("img/one.png", "gopher.S.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
	})

	// Initial load
	env.Handler.LoadImages()
	env.assertWebPExists("one", image.VariantS)

	// Add an orphan manually
	orphanPath := filepath.Join(env.OutputDir, "orphan.S.webp")
	os.WriteFile(orphanPath, []byte("garbage"), 0644)

	// Load again, should cleanup orphan
	env.Handler.LoadImages()
	if _, err := os.Stat(orphanPath); err == nil {
		t.Error("expected orphan to be removed")
	}
	env.assertWebPExists("one", image.VariantS)

	// Remove image from SSR and load again
	env.writeImageGoWithImages([]image.Asset{})
	env.Handler.LoadImages()
	env.assertWebPNotExists("one", image.VariantS)
}

func TestLoadImagesGoListFails(t *testing.T) {
	env := newTestEnv(t)
	env.Handler.SetListModulesFn(func(rootDir string) ([]string, error) {
		return nil, os.ErrPermission
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatalf("LoadImages should not return error on go list failure, only log warning. Got: %v", err)
	}
}

func TestLoadImagesRootDirEmpty(t *testing.T) {
	env := newTestEnv(t)
	env.Handler = min.New(&min.Config{
		RootDir:   "",
		OutputDir: env.OutputDir,
	})
	err := env.Handler.LoadImages()
	if err == nil {
		t.Error("expected error for empty RootDir")
	}
}

func TestReloadModuleNoImageFile(t *testing.T) {
	env := newTestEnv(t)
	// No image.go file in ModuleDir
	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule should not fail if no image.go exists: %v", err)
	}
}
