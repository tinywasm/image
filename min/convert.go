//go:build !wasm

package min

import (
	"fmt"
	stdimage "image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	"github.com/disintegration/imaging"
	"github.com/tinywasm/image"
)

func ProcessImage(src ParsedAsset, outputDir string, quality int, log func(...any)) ([]string, error) {
	if IsVector(src.AbsPath) {
		return copyVerbatim(src, outputDir)
	}

	img, err := imaging.Open(src.AbsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image %s: %w", src.AbsPath, err)
	}
	opaque := isOpaque(img)
	ext := extForOpacity(opaque)

	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	variants := []struct {
		v     image.Variant
		width int
	}{
		{image.VariantS, 640},
		{image.VariantM, 1024},
		{image.VariantL, 1920},
	}
	var outputFiles []string
	for _, vInfo := range variants {
		if src.Variants&vInfo.v != 0 {
			var processedImg stdimage.Image
			if originalWidth <= vInfo.width {
				log("image smaller than target, skipping resize", src.BaseName, vInfo.v)
				processedImg = img
			} else {
				processedImg = imaging.Resize(img, vInfo.width, 0, imaging.Lanczos)
			}
			outputName := fmt.Sprintf("%s.%s%s", src.BaseName, variantSuffix(vInfo.v), ext)
			outputPath := filepath.Join(outputDir, outputName)
			var err error
			if opaque {
				err = writeJPEG(processedImg, outputPath, quality)
			} else {
				err = writeWebP(processedImg, outputPath, quality)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", outputName, err)
			}
			outputFiles = append(outputFiles, outputName)
		}
	}
	return outputFiles, nil
}

// ExpectedExt is the extension ProcessImage will write for src — the single
// answer the whole pipeline asks, so that "is this output current?" and "is
// this output an orphan?" cannot disagree with what the encoder actually
// produces. Treating both extensions as equally valid is what let a lossless
// .webp written by an older release survive next to the .jpg that replaced it:
// each variant appeared up to date because SOME file matched, and cleanup
// spared the stale one because SOME asset could have claimed it.
//
// A JPEG source needs no decode: the format has no alpha channel, so it is
// opaque by construction. Anything else has to be read. An unreadable source
// returns "", meaning "unknown" — callers then fall back to accepting either
// extension rather than deleting output they cannot reason about.
func ExpectedExt(srcPath string) string {
	switch strings.ToLower(filepath.Ext(srcPath)) {
	case ".jpg", ".jpeg":
		return ".jpg"
	case vectorExt:
		return vectorExt
	}
	img, err := imaging.Open(srcPath)
	if err != nil {
		return ""
	}
	return extForOpacity(isOpaque(img))
}

func extForOpacity(opaque bool) string {
	if opaque {
		return ".jpg"
	}
	return ".webp"
}

func isOpaque(img stdimage.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xffff {
				return false
			}
		}
	}
	return true
}

func variantSuffix(v image.Variant) string {
	switch v {
	case image.VariantS:
		return "S"
	case image.VariantM:
		return "M"
	case image.VariantL:
		return "L"
	default:
		return "unknown"
	}
}

func writeJPEG(img stdimage.Image, path string, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if quality <= 0 {
		quality = 75
	}
	return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
}

func writeWebP(img stdimage.Image, path string, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	// Note: github.com/HugoSmits86/nativewebp currently only supports lossless WebP encoding (VP8L).
	// The quality parameter is accepted for future compatibility but not currently used by the encoder.
	return nativewebp.Encode(f, img, nil)
}

func deriveAlt(baseName string) string {
	return strings.ReplaceAll(baseName, "-", " ")
}

// vectorExt es la única extensión sin variantes rasterizadas: un vector ya
// escala, así que redimensionarlo no significa nada.
const vectorExt = ".svg"

// IsVector responde si la fuente se entrega tal cual en vez de codificarse.
//
// Existe porque un logo vectorial no tiene dónde vivir en un pipeline que solo
// sabía convertir mapas de bits: quedaba fuera de image.Asset y cada sitio se
// escribía su propio copiador de archivos. Ese copiador era invisible para el
// demonio, así que el mismo proyecto servía el logo en release y devolvía 404
// en desarrollo.
func IsVector(srcPath string) bool {
	return strings.EqualFold(filepath.Ext(srcPath), vectorExt)
}

// VectorOutputName es el nombre con el que un vector aterriza en la salida: el
// suyo, sin sufijo de variante. La página lo referencia por ese nombre.
func VectorOutputName(srcPath string) string {
	return filepath.Base(srcPath)
}

// copyVerbatim entrega el vector sin tocarlo. Variants se ignora a propósito:
// no hay tres anchos que generar.
func copyVerbatim(src ParsedAsset, outputDir string) ([]string, error) {
	content, err := os.ReadFile(src.AbsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vector %s: %w", src.AbsPath, err)
	}
	name := VectorOutputName(src.AbsPath)
	if err := os.WriteFile(filepath.Join(outputDir, name), content, 0644); err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", name, err)
	}
	return []string{name}, nil
}
