package imagemin_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tinywasm/imagemin"
)

func TestMtimeSkipsUnchanged(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/logo.png")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 100, 100)
	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: "img/logo.png", Variants: imagemin.VariantS, Alt: "Logo"},
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(env.OutputDir, "logo.S.webp")
	stat1, err := os.Stat(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	// Second pass should skip
	err = env.Handler.LoadImages()
	if err != nil {
		t.Fatal(err)
	}

	stat2, err := os.Stat(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	if stat1.ModTime() != stat2.ModTime() {
		t.Error("expected mtime to be identical after skip")
	}
}

func TestMtimeReprocessesOnChange(t *testing.T) {
	env := newTestEnv(t)
	srcPath := filepath.Join(env.ModuleDir, "img/logo.png")
	os.MkdirAll(filepath.Dir(srcPath), 0755)
	createTestImage(srcPath, 100, 100)
	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: "img/logo.png", Variants: imagemin.VariantS, Alt: "Logo"},
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(env.OutputDir, "logo.S.webp")
	stat1, err := os.Stat(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	// Touch source file in the future
	future := time.Now().Add(1 * time.Hour)
	err = os.Chtimes(srcPath, future, future)
	if err != nil {
		t.Fatal(err)
	}

	// Second pass should reprocess
	err = env.Handler.LoadImages()
	if err != nil {
		t.Fatal(err)
	}

	stat2, err := os.Stat(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	if !stat2.ModTime().After(stat1.ModTime()) {
		t.Error("expected mtime to be updated after source change")
	}
}

func TestMtimeMissingVariant(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/logo.png")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 100, 100)
	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: "img/logo.png", Variants: imagemin.VariantS | imagemin.VariantM, Alt: "Logo"},
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatal(err)
	}

	env.assertWebPExists("logo", imagemin.VariantS)
	env.assertWebPExists("logo", imagemin.VariantM)

	// Delete one variant
	err = os.Remove(filepath.Join(env.OutputDir, "logo.M.webp"))
	if err != nil {
		t.Fatal(err)
	}

	// Second pass should regenerate the missing one
	err = env.Handler.LoadImages()
	if err != nil {
		t.Fatal(err)
	}

	env.assertWebPExists("logo", imagemin.VariantM)
}
