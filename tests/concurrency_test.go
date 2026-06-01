package imagemin_test

import (
	"sync"
	"testing"

	"github.com/tinywasm/imagemin"
)

func TestReloadConcurrency(t *testing.T) {
	env := newTestEnv(t)
	env.copyTestImage("img/logo.png", "gopher.S.png")
	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: "img/logo.png", Variants: imagemin.VariantS},
	})

	const goroutines = 5
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			err := env.Handler.ReloadModule(env.ModuleDir)
			if err != nil {
				t.Errorf("ReloadModule failed: %v", err)
			}
		}()
	}

	wg.Wait()
	env.assertWebPExists("logo", imagemin.VariantS)
}

func TestLoadAndReloadConcurrent(t *testing.T) {
	env := newTestEnv(t)
	env.copyTestImage("img/logo.png", "gopher.S.png")
	env.writeSSRGoWithImages([]imagemin.Asset{
		{Path: "img/logo.png", Variants: imagemin.VariantS},
	})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		env.Handler.LoadImages()
	}()

	go func() {
		defer wg.Done()
		env.Handler.ReloadModule(env.ModuleDir)
	}()

	wg.Wait()
}
