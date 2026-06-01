//go:build !wasm

package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func IsUpToDate(srcPath string, variants Variant, outputDir string) bool {
	srcStat, err := os.Stat(srcPath)
	if err != nil {
		return false
	}
	srcMtime := srcStat.ModTime()
	variantInfos := []struct {
		v Variant
		s string
	}{
		{VariantS, "S"},
		{VariantM, "M"},
		{VariantL, "L"},
	}
	baseName := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
	for _, vi := range variantInfos {
		if variants&vi.v != 0 {
			outPath := filepath.Join(outputDir, fmt.Sprintf("%s.%s.webp", baseName, vi.s))
			outStat, err := os.Stat(outPath)
			if err != nil {
				return false
			}
			if srcMtime.After(outStat.ModTime()) {
				return false
			}
		}
	}
	return true
}
