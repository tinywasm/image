package image_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/image"
	"github.com/tinywasm/image/min"
	"github.com/tinywasm/modfind"
)

func TestLoadImagesFromModule(t *testing.T) {
	env := newTestEnv(t)

	imgPath := "img/logo.png"
	env.createTinyImage(imgPath)

	env.writeImageGoWithImages([]image.Asset{
		{Path: imgPath, Variants: image.VariantS, Alt: "Gopher"},
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatalf("LoadImages failed: %v", err)
	}

	env.assertJPGExists("logo", image.VariantS)
}

func TestReloadModuleNewImage(t *testing.T) {
	env := newTestEnv(t)

	env.createTinyImage("img/one.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
	})
	env.Handler.ReloadModule(env.ModuleDir)
	env.assertJPGExists("one", image.VariantS)

	env.createTinyImage("img/two.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
		{Path: "img/two.png", Variants: image.VariantS},
	})

	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule failed: %v", err)
	}

	env.assertJPGExists("one", image.VariantS)
	env.assertJPGExists("two", image.VariantS)
}

func TestReloadModuleRemovedImageDoesNotCleanup(t *testing.T) {
	env := newTestEnv(t)

	env.createTinyImage("img/one.png")
	env.createTinyImage("img/two.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
		{Path: "img/two.png", Variants: image.VariantS},
	})
	env.Handler.ReloadModule(env.ModuleDir)
	env.assertJPGExists("one", image.VariantS)
	env.assertJPGExists("two", image.VariantS)

	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
	})

	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule failed: %v", err)
	}

	env.assertJPGExists("one", image.VariantS)
	// ReloadModule no longer cleans orphans (global cleanup only in LoadImages)
	env.assertJPGExists("two", image.VariantS)
}

func TestGlobalOrphanCleanup(t *testing.T) {
	env := newTestEnv(t)

	env.createTinyImage("img/one.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
	})

	// Initial load
	env.Handler.LoadImages()
	env.assertJPGExists("one", image.VariantS)

	// Add orphans manually (.jpg and .webp)
	orphanJpgPath := filepath.Join(env.OutputDir, "orphan.S.jpg")
	os.WriteFile(orphanJpgPath, []byte("garbage"), 0644)
	orphanWebpPath := filepath.Join(env.OutputDir, "orphan.S.webp")
	os.WriteFile(orphanWebpPath, []byte("garbage"), 0644)

	// Load again, should cleanup orphans
	env.Handler.LoadImages()
	if _, err := os.Stat(orphanJpgPath); err == nil {
		t.Error("expected JPG orphan to be removed")
	}
	if _, err := os.Stat(orphanWebpPath); err == nil {
		t.Error("expected WebP orphan to be removed")
	}
	env.assertJPGExists("one", image.VariantS)

	// Remove image from SSR and load again
	env.writeImageGoWithImages([]image.Asset{})
	env.Handler.LoadImages()
	env.assertJPGNotExists("one", image.VariantS)
}

func TestLoadImagesGoListFails(t *testing.T) {
	env := newTestEnv(t)
	f := modfind.New()

	// We want to simulate a go list failure.
	// We can just use an invalid directory to trigger an error in modfind.Dirs
	handler := min.New(&min.Config{
		RootDir:   "/non-existent-dir-at-all-12345",
		OutputDir: env.OutputDir,
	})
	handler.SetFinder(f)
	handler.SetLog(func(messages ...any) {
		// Just to capture that it's called
	})

	err := handler.LoadImages()
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
