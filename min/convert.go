package min

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	stdimage "image"

	"github.com/HugoSmits86/nativewebp"
	"github.com/disintegration/imaging"
	"github.com/tinywasm/image"
)

func ProcessImage(src ParsedAsset, outputDir string, quality int, log func(...any)) ([]string, error) {
	img, err := imaging.Open(src.AbsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image %s: %w", src.AbsPath, err)
	}
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
			outputName := fmt.Sprintf("%s.%s.webp", src.BaseName, variantSuffix(vInfo.v))
			outputPath := filepath.Join(outputDir, outputName)
			err := writeWebP(processedImg, outputPath, quality)
			if err != nil {
				return nil, fmt.Errorf("failed to write webp %s: %w", outputName, err)
			}
			outputFiles = append(outputFiles, outputName)
		}
	}
	return outputFiles, nil
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
