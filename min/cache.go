//go:build !wasm

package min

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tinywasm/image"
)

func IsUpToDate(srcPath string, variants image.Variant, outputDir string) bool {
	srcStat, err := os.Stat(srcPath)
	if err != nil {
		return false
	}
	srcMtime := srcStat.ModTime()

	// Un vector no tiene variantes: su salida es un único archivo con el mismo
	// nombre. Preguntar por sufijos S/M/L aquí lo daría por vigente siempre,
	// porque ninguna variante estaría declarada y el bucle no miraría nada.
	if IsVector(srcPath) {
		outStat, err := os.Stat(filepath.Join(outputDir, VectorOutputName(srcPath)))
		return err == nil && !srcMtime.After(outStat.ModTime())
	}

	variantInfos := []struct {
		v image.Variant
		s string
	}{
		{image.VariantS, "S"},
		{image.VariantM, "M"},
		{image.VariantL, "L"},
	}
	baseName := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))

	// Only the extension this source will actually be encoded to counts. If a
	// previous release wrote the other one, this asset is NOT up to date — it
	// still has to be re-encoded, and the leftover is cleanOrphans' problem.
	extensions := []string{".jpg", ".webp"}
	if ext := ExpectedExt(srcPath); ext != "" {
		extensions = []string{ext}
	}

	for _, vi := range variantInfos {
		if variants&vi.v != 0 {
			found := false
			for _, ext := range extensions {
				outPath := filepath.Join(outputDir, fmt.Sprintf("%s.%s%s", baseName, vi.s, ext))
				outStat, err := os.Stat(outPath)
				if err == nil {
					found = true
					if srcMtime.After(outStat.ModTime()) {
						return false
					}
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}
