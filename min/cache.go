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
	variantInfos := []struct {
		v image.Variant
		s string
	}{
		{image.VariantS, "S"},
		{image.VariantM, "M"},
		{image.VariantL, "L"},
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
