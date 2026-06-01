package imagemin_test

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/imagemin"
)

type TestEnv struct {
	t         *testing.T
	ModuleDir string
	OutputDir string
	Handler   *imagemin.Handler
}

func newTestEnv(t *testing.T) *TestEnv {
	moduleDir := t.TempDir()
	outputDir := t.TempDir()

	config := &imagemin.Config{
		RootDir:   moduleDir,
		OutputDir: outputDir,
		Quality:   82,
	}

	handler := imagemin.New(config)
	handler.SetListModulesFn(func(rootDir string) ([]string, error) {
		return []string{moduleDir}, nil
	})

	return &TestEnv{
		t:         t,
		ModuleDir: moduleDir,
		OutputDir: outputDir,
		Handler:   handler,
	}
}

func (e *TestEnv) writeSSRGo(content string) {
	err := os.WriteFile(filepath.Join(e.ModuleDir, "ssr.go"), []byte(content), 0644)
	if err != nil {
		e.t.Fatalf("failed to write ssr.go: %v", err)
	}
}

func (e *TestEnv) writeSSRGoWithImages(assets []imagemin.Asset) {
	content := "//go:build !wasm\n\npackage module\n\nimport \"github.com/tinywasm/imagemin\"\n\nfunc RenderImages() []imagemin.Asset {\n\treturn []imagemin.Asset{\n"
	for _, asset := range assets {
		content += fmt.Sprintf("\t\t{Path: %q, Variants: imagemin.Variant(%d), Alt: %q},\n", asset.Path, asset.Variants, asset.Alt)
	}
	content += "\t}\n}\n"
	e.writeSSRGo(content)
}

func (e *TestEnv) copyTestImage(destRelPath, testdataFile string) {
	srcPath := filepath.Join("testdata", testdataFile)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		// Try to find it if we are in project root
		srcPath = filepath.Join("tests", "testdata", testdataFile)
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		e.t.Fatalf("failed to read test image %s: %v", testdataFile, err)
	}

	destPath := filepath.Join(e.ModuleDir, destRelPath)
	err = os.MkdirAll(filepath.Dir(destPath), 0755)
	if err != nil {
		e.t.Fatalf("failed to create dest dir: %v", err)
	}

	err = os.WriteFile(destPath, data, 0644)
	if err != nil {
		e.t.Fatalf("failed to write dest image: %v", err)
	}
}

func (e *TestEnv) assertWebPExists(name string, v imagemin.Variant) {
	path := filepath.Join(e.OutputDir, fmt.Sprintf("%s.%s.webp", name, variantName(v)))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		e.t.Errorf("expected WebP variant %s for %s to exist", variantName(v), name)
	}
}

func (e *TestEnv) assertWebPNotExists(name string, v imagemin.Variant) {
	path := filepath.Join(e.OutputDir, fmt.Sprintf("%s.%s.webp", name, variantName(v)))
	if _, err := os.Stat(path); err == nil {
		e.t.Errorf("expected WebP variant %s for %s NOT to exist", variantName(v), name)
	}
}

func variantName(v imagemin.Variant) string {
	switch v {
	case imagemin.VariantS:
		return "S"
	case imagemin.VariantM:
		return "M"
	case imagemin.VariantL:
		return "L"
	default:
		return "unknown"
	}
}

func createTestImage(path string, width, height int) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 0, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, nil)
}

func createTestPNG(path string, width, height int, alpha bool) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	a := uint8(255)
	if alpha {
		a = 128
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 0, a})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
