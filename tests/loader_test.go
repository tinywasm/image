package imagemin_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/imagemin"
)

func TestLoadImagesFromModule(t *testing.T) {
	env := newTestEnv(t)

	imgPath := "img/logo.png"
	env.copyTestImage(imgPath, "gopher.S.png")

	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: imgPath, Variants: imagemin.VariantS, Alt: "Gopher"},
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatalf("LoadImages failed: %v", err)
	}

	env.assertWebPExists("logo", imagemin.VariantS)
}

func TestReloadModuleNewImage(t *testing.T) {
	env := newTestEnv(t)

	env.copyTestImage("img/one.png", "gopher.S.png")
	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: "img/one.png", Variants: imagemin.VariantS},
	})
	env.Handler.ReloadModule(env.ModuleDir)
	env.assertWebPExists("one", imagemin.VariantS)

	env.copyTestImage("img/two.png", "gopher.S.png")
	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: "img/one.png", Variants: imagemin.VariantS},
		{Path: "img/two.png", Variants: imagemin.VariantS},
	})

	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule failed: %v", err)
	}

	env.assertWebPExists("one", imagemin.VariantS)
	env.assertWebPExists("two", imagemin.VariantS)
}

func TestReloadModuleRemovedImageDoesNotCleanup(t *testing.T) {
	env := newTestEnv(t)

	env.copyTestImage("img/one.png", "gopher.S.png")
	env.copyTestImage("img/two.png", "gopher.S.png")
	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: "img/one.png", Variants: imagemin.VariantS},
		{Path: "img/two.png", Variants: imagemin.VariantS},
	})
	env.Handler.ReloadModule(env.ModuleDir)
	env.assertWebPExists("one", imagemin.VariantS)
	env.assertWebPExists("two", imagemin.VariantS)

	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: "img/one.png", Variants: imagemin.VariantS},
	})

	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule failed: %v", err)
	}

	env.assertWebPExists("one", imagemin.VariantS)
	// ReloadModule no longer cleans orphans (global cleanup only in LoadImages)
	env.assertWebPExists("two", imagemin.VariantS)
}

func TestGlobalOrphanCleanup(t *testing.T) {
	env := newTestEnv(t)

	env.copyTestImage("img/one.png", "gopher.S.png")
	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: "img/one.png", Variants: imagemin.VariantS},
	})

	// Initial load
	env.Handler.LoadImages()
	env.assertWebPExists("one", imagemin.VariantS)

	// Add an orphan manually
	orphanPath := filepath.Join(env.OutputDir, "orphan.S.webp")
	os.WriteFile(orphanPath, []byte("garbage"), 0644)

	// Load again, should cleanup orphan
	env.Handler.LoadImages()
	if _, err := os.Stat(orphanPath); err == nil {
		t.Error("expected orphan to be removed")
	}
	env.assertWebPExists("one", imagemin.VariantS)

	// Remove image from SSR and load again
	env.writeSSRGoWithImages([]imagemin.Asset{})
	env.Handler.LoadImages()
	env.assertWebPNotExists("one", imagemin.VariantS)
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
	env.Handler = imagemin.New(&imagemin.Config{
		RootDir:   "",
		OutputDir: env.OutputDir,
	})
	err := env.Handler.LoadImages()
	if err == nil {
		t.Error("expected error for empty RootDir")
	}
}

func TestReloadModuleNoSSRFile(t *testing.T) {
	env := newTestEnv(t)
	// No SSR file in ModuleDir
	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule should not fail if no ssr.go exists: %v", err)
	}
}
