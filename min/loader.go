//go:build !wasm

package min

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tinywasm/image"
	"github.com/tinywasm/modfind"
)

func (h *Handler) moduleDirs() ([]string, error) {
	h.mu.Lock()
	if h.finder == nil {
		h.finder = modfind.New()
	}
	h.mu.Unlock()
	return h.finder.Dirs(h.config.RootDir)
}

// LoadImages discovers modules via go list and processes their images.
func (h *Handler) LoadImages() error {
	if h.config.RootDir == "" {
		return fmt.Errorf("config.RootDir is empty")
	}

	moduleDirs, err := h.moduleDirs()
	if err != nil {
		h.log("warning: failed to list modules:", err)
		return nil
	}
	var allAssets []ParsedAsset
	for _, dir := range moduleDirs {
		assets, err := ExtractImages(dir)
		if err != nil {
			h.log("warning: failed to extract images from module", dir, ":", err)
			continue
		}
		for _, asset := range assets {
			if err := h.processAsset(asset); err != nil {
				h.log("warning: failed to process asset", asset.AbsPath, ":", err)
			}
		}
		allAssets = append(allAssets, assets...)
	}
	h.cleanOrphans(allAssets)
	return nil
}

// ReloadModule re-extracts and re-processes images for a single module.
func (h *Handler) ReloadModule(moduleDir string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	assets, err := ExtractImages(moduleDir)
	if err != nil {
		return err
	}
	for _, asset := range assets {
		if err := h.processAsset(asset); err != nil {
			h.log("warning: failed to process asset", asset.AbsPath, ":", err)
		}
	}
	return nil
}

func (h *Handler) processAsset(asset ParsedAsset) error {
	if IsUpToDate(asset.AbsPath, asset.Variants, h.config.OutputDir) {
		return nil
	}
	if err := os.MkdirAll(h.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	_, err := ProcessImage(asset, h.config.OutputDir, h.config.Quality, h.log)
	return err
}

func (h *Handler) cleanOrphans(allAssets []ParsedAsset) {
	if h.config.OutputDir == "" {
		return
	}
	activeFiles := make(map[string]bool)
	for _, asset := range allAssets {
		variantInfos := []struct {
			v image.Variant
			s string
		}{
			{image.VariantS, "S"},
			{image.VariantM, "M"},
			{image.VariantL, "L"},
		}
		// Only the extension this source encodes to is live. Marking both
		// keeps a lossless .webp left over from an older release alive
		// forever, sitting next to the .jpg that superseded it — the page
		// then ships one of them and the other is pure dead weight.
		extensions := []string{".jpg", ".webp"}
		if ext := ExpectedExt(asset.AbsPath); ext != "" {
			extensions = []string{ext}
		}
		for _, vi := range variantInfos {
			if asset.Variants&vi.v != 0 {
				for _, ext := range extensions {
					activeFiles[fmt.Sprintf("%s.%s%s", asset.BaseName, vi.s, ext)] = true
				}
			}
		}
	}
	files, err := os.ReadDir(h.config.OutputDir)
	if err != nil {
		return
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if strings.HasSuffix(name, ".webp") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") {
			if !activeFiles[name] {
				os.Remove(filepath.Join(h.config.OutputDir, name))
			}
		}
	}
}
