package min

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tinywasm/image"
)

func (h *Handler) listModulesReal(rootDir string) ([]string, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = rootDir
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	var dirs []string
	dec := json.NewDecoder(&out)
	for dec.More() {
		var m struct {
			Dir string
		}
		if err := dec.Decode(&m); err != nil {
			return nil, err
		}
		if m.Dir != "" {
			dirs = append(dirs, m.Dir)
		}
	}
	return dirs, nil
}

func (h *Handler) InitDefaultLoader() {
	h.listModulesFn = h.listModulesReal
}

// LoadImages discovers modules via go list and processes their images.
func (h *Handler) LoadImages() error {
	if h.config.RootDir == "" {
		return fmt.Errorf("config.RootDir is empty")
	}
	if h.listModulesFn == nil {
		return fmt.Errorf("listModulesFn not set")
	}
	moduleDirs, err := h.listModulesFn(h.config.RootDir)
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
		for _, vi := range variantInfos {
			if asset.Variants&vi.v != 0 {
				activeFiles[fmt.Sprintf("%s.%s.webp", asset.BaseName, vi.s)] = true
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
		if strings.HasSuffix(name, ".webp") {
			if !activeFiles[name] {
				os.Remove(filepath.Join(h.config.OutputDir, name))
			}
		}
	}
}
