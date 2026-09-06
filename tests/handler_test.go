//go:build !wasm

package image_test

import (
	"testing"

	"webtyp.com/image/min"
)

func TestHandlerUnobservedFiles(t *testing.T) {
	config := &min.Config{OutputDir: "web/public/img"}
	handler := min.New(config)
	unobserved := handler.UnobservedFiles()

	foundOutputDir := false
	for _, f := range unobserved {
		if f == "web/public/img" {
			foundOutputDir = true
		}
	}

	if !foundOutputDir {
		t.Errorf("expected OutputDir in UnobservedFiles")
	}
}
