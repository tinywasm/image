package imagemin_test

import (
	"testing"

	"github.com/tinywasm/imagemin"
)

func TestHandlerUnobservedFiles(t *testing.T) {
	config := &imagemin.Config{OutputDir: "web/public/img"}
	handler := imagemin.New(config)
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
