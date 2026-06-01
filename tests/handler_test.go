package image_test

import (
	"testing"

	"github.com/tinywasm/image"
)

func TestHandlerUnobservedFiles(t *testing.T) {
	config := &image.Config{OutputDir: "web/public/img"}
	handler := image.New(config)
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
