//go:build !wasm

package image_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tinywasm/image"
)

func TestMtimeSkipsUnchanged(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/logo.png")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 100, 100)
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/logo.png", Variants: image.VariantS, Alt: "Logo"},
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(env.OutputDir, "logo.S.jpg")
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
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/logo.png", Variants: image.VariantS, Alt: "Logo"},
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(env.OutputDir, "logo.S.jpg")

	// Set output file mtime to past
	past := time.Now().Add(-1 * time.Hour)
	err = os.Chtimes(outputPath, past, past)
	if err != nil {
		t.Fatal(err)
	}

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
		t.Errorf("expected mtime (%v) to be updated after source change (was %v)", stat2.ModTime(), stat1.ModTime())
	}
}

func TestMtimeMissingVariant(t *testing.T) {
	env := newTestEnv(t)
	imgPath := filepath.Join(env.ModuleDir, "img/logo.png")
	os.MkdirAll(filepath.Dir(imgPath), 0755)
	createTestImage(imgPath, 100, 100)
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/logo.png", Variants: image.VariantS | image.VariantM, Alt: "Logo"},
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatal(err)
	}

	env.assertJPGExists("logo", image.VariantS)
	env.assertJPGExists("logo", image.VariantM)

	// Delete one variant
	err = os.Remove(filepath.Join(env.OutputDir, "logo.M.jpg"))
	if err != nil {
		t.Fatal(err)
	}

	// Second pass should regenerate the missing one
	err = env.Handler.LoadImages()
	if err != nil {
		t.Fatal(err)
	}

	env.assertJPGExists("logo", image.VariantM)
}
