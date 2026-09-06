//go:build !wasm

package image_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"webtyp.com/image"
	"webtyp.com/image/min"
	"webtyp.com/modfind"
)

func TestLoadImagesFromModule(t *testing.T) {
	env := newTestEnv(t)

	imgPath := "img/logo.png"
	env.createTinyImage(imgPath)

	env.writeImageGoWithImages([]image.Asset{
		{Path: imgPath, Variants: image.VariantS, Alt: "Gopher"},
	})

	err := env.Handler.LoadImages()
	if err != nil {
		t.Fatalf("LoadImages failed: %v", err)
	}

	env.assertJPGExists("logo", image.VariantS)
}

func TestReloadModuleNewImage(t *testing.T) {
	env := newTestEnv(t)

	env.createTinyImage("img/one.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
	})
	env.Handler.ReloadModule(env.ModuleDir)
	env.assertJPGExists("one", image.VariantS)

	env.createTinyImage("img/two.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
		{Path: "img/two.png", Variants: image.VariantS},
	})

	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule failed: %v", err)
	}

	env.assertJPGExists("one", image.VariantS)
	env.assertJPGExists("two", image.VariantS)
}

func TestReloadModuleRemovedImageDoesNotCleanup(t *testing.T) {
	env := newTestEnv(t)

	env.createTinyImage("img/one.png")
	env.createTinyImage("img/two.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
		{Path: "img/two.png", Variants: image.VariantS},
	})
	env.Handler.ReloadModule(env.ModuleDir)
	env.assertJPGExists("one", image.VariantS)
	env.assertJPGExists("two", image.VariantS)

	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
	})

	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule failed: %v", err)
	}

	env.assertJPGExists("one", image.VariantS)
	// ReloadModule no longer cleans orphans (global cleanup only in LoadImages)
	env.assertJPGExists("two", image.VariantS)
}

func TestGlobalOrphanCleanup(t *testing.T) {
	env := newTestEnv(t)

	env.createTinyImage("img/one.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/one.png", Variants: image.VariantS},
	})

	// Initial load
	env.Handler.LoadImages()
	env.assertJPGExists("one", image.VariantS)

	// Add orphans manually (.jpg and .webp)
	orphanJpgPath := filepath.Join(env.OutputDir, "orphan.S.jpg")
	os.WriteFile(orphanJpgPath, []byte("garbage"), 0644)
	orphanWebpPath := filepath.Join(env.OutputDir, "orphan.S.webp")
	os.WriteFile(orphanWebpPath, []byte("garbage"), 0644)

	// Load again, should cleanup orphans
	env.Handler.LoadImages()
	if _, err := os.Stat(orphanJpgPath); err == nil {
		t.Error("expected JPG orphan to be removed")
	}
	if _, err := os.Stat(orphanWebpPath); err == nil {
		t.Error("expected WebP orphan to be removed")
	}
	env.assertJPGExists("one", image.VariantS)

	// Remove image from SSR and load again
	env.writeImageGoWithImages([]image.Asset{})
	env.Handler.LoadImages()
	env.assertJPGNotExists("one", image.VariantS)
}

func TestLoadImagesGoListFails(t *testing.T) {
	env := newTestEnv(t)
	f := modfind.New()

	// We want to simulate a go list failure.
	// We can just use an invalid directory to trigger an error in modfind.Dirs
	handler := min.New(&min.Config{
		RootDir:   "/non-existent-dir-at-all-12345",
		OutputDir: env.OutputDir,
	})
	handler.SetFinder(f)
	handler.SetLog(func(messages ...any) {
		// Just to capture that it's called
	})

	err := handler.LoadImages()
	if err != nil {
		t.Fatalf("LoadImages should not return error on go list failure, only log warning. Got: %v", err)
	}
}

func TestLoadImagesRootDirEmpty(t *testing.T) {
	env := newTestEnv(t)
	env.Handler = min.New(&min.Config{
		RootDir:   "",
		OutputDir: env.OutputDir,
	})
	err := env.Handler.LoadImages()
	if err == nil {
		t.Error("expected error for empty RootDir")
	}
}

func TestReloadModuleNoImageFile(t *testing.T) {
	env := newTestEnv(t)
	// No image.go file in ModuleDir
	err := env.Handler.ReloadModule(env.ModuleDir)
	if err != nil {
		t.Fatalf("ReloadModule should not fail if no image.go exists: %v", err)
	}
}

// TestStaleLosslessOutputIsReplacedAndRemoved reproduces what shipped in
// mjosefa-website: a site built with the old lossless-WebP encoder, then
// rebuilt after the switch to JPEG for opaque photographs. Every variant ended
// up on disk TWICE — a fresh 52 KB .jpg next to a 1.4 MB .webp from the
// previous release — because both the freshness check and the orphan sweep
// accepted either extension. The .webp was never regenerated (some file
// matched, so the asset looked current) and never deleted (some asset could
// have claimed it, so it looked live).
func TestStaleLosslessOutputIsReplacedAndRemoved(t *testing.T) {
	env := newTestEnv(t)

	env.createTinyImage("img/photo.jpg")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/photo.jpg", Variants: image.VariantS},
	})

	// The previous release's output: same base name and variant, WebP.
	stale := filepath.Join(env.OutputDir, "photo.S.webp")
	if err := os.MkdirAll(env.OutputDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("lossless webp from the old encoder"), 0644); err != nil {
		t.Fatal(err)
	}

	env.Handler.LoadImages()

	// The opaque JPEG source must be encoded to .jpg despite the .webp sitting
	// there looking like a finished output.
	env.assertJPGExists("photo", image.VariantS)

	if _, err := os.Stat(stale); err == nil {
		t.Error("stale .webp from the previous encoder survived the rebuild; the page now ships two copies of every photograph")
	}
}

// TestDeclaredVectorReachesOutput cierra el hueco que hacía que un mismo
// proyecto sirviera el logo en release y devolviera 404 en desarrollo.
//
// Un logo vectorial no cabía en image.Asset —el pipeline solo sabía convertir
// mapas de bits—, así que cada sitio se escribía su propio copiador de
// archivos en su herramienta de build. Esa copia era invisible para el
// demonio, que sirve exactamente lo que este handler produce.
func TestDeclaredVectorReachesOutput(t *testing.T) {
	env := newTestEnv(t)

	const logo = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><circle cx="5" cy="5" r="4"/></svg>`
	srcPath := filepath.Join(env.ModuleDir, "img", "logo-completo.svg")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, []byte(logo), 0644); err != nil {
		t.Fatal(err)
	}

	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/logo-completo.svg", Alt: "Logo"},
	})

	if err := env.Handler.LoadImages(); err != nil {
		t.Fatalf("LoadImages failed: %v", err)
	}

	out := filepath.Join(env.OutputDir, "logo-completo.svg")
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("el vector declarado no llegó a la salida: %v", err)
	}
	if string(got) != logo {
		t.Errorf("el vector se entregó alterado.\nesperado: %q\nobtenido: %q", logo, string(got))
	}
}

// TestUndeclaredVectorIsRemoved es la otra mitad de la regla: el sitio se sirve
// desde lo declarado, no desde lo que alguien dejó a mano en la salida. Un
// archivo que sobrevive sin declaración es la vía por la que release y
// desarrollo se desincronizan sin que nada falle.
func TestUndeclaredVectorIsRemoved(t *testing.T) {
	env := newTestEnv(t)

	stale := filepath.Join(env.OutputDir, "logo-viejo.svg")
	if err := os.WriteFile(stale, []byte(`<svg/>`), 0644); err != nil {
		t.Fatal(err)
	}

	env.createTinyImage("img/foto.png")
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/foto.png", Variants: image.VariantS},
	})

	if err := env.Handler.LoadImages(); err != nil {
		t.Fatalf("LoadImages failed: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("un vector sin declarar sobrevivió en la salida: %s", stale)
	}
}

// TestArtifactsSirvenElSitioSinTocarElProyecto tiene la forma exacta del
// consumidor: un servidor de desarrollo que responde peticiones desde memoria.
// Si Artifacts() no devuelve el contenido, la única alternativa del consumidor
// es escribir un directorio de salida dentro del proyecto del usuario.
func TestArtifactsSirvenElSitioSinTocarElProyecto(t *testing.T) {
	env := newTestEnv(t)

	// 1. Crear origen ráster y vectorial
	env.createTinyImage("img/foto.png")

	const svgContent = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><circle cx="5" cy="5" r="4"/></svg>`
	svgSrcPath := filepath.Join(env.ModuleDir, "img", "logo.svg")
	if err := os.MkdirAll(filepath.Dir(svgSrcPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(svgSrcPath, []byte(svgContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Declararlos
	env.writeImageGoWithImages([]image.Asset{
		{Path: "img/foto.png", Variants: image.VariantM, Alt: "Foto"},
		{Path: "img/logo.svg", Alt: "Logo"},
	})

	// 3. LoadImages()
	if err := env.Handler.LoadImages(); err != nil {
		t.Fatalf("LoadImages failed: %v", err)
	}

	// 4. Montar http.ServeMux con los Artifacts()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		for _, art := range env.Handler.Artifacts() {
			if art.Path == reqPath {
				w.Header().Set("Content-Type", art.Mediatype)
				w.Write(art.Content)
				return
			}
		}
		http.NotFound(w, r)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// 5. Pedir /img/<nombre ráster> y /img/<nombre vector>
	testCases := []struct {
		urlPath          string
		expectedFileName string
		expectedMime     string
	}{
		{"/img/foto.M.jpg", "foto.M.jpg", min.MediatypeJPEG},
		{"/img/logo.svg", "logo.svg", min.MediatypeSVG},
	}

	for _, tc := range testCases {
		resp, err := http.Get(server.URL + tc.urlPath)
		if err != nil {
			t.Fatalf("failed to GET %s: %v", tc.urlPath, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s status code = %d; expected 200", tc.urlPath, resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != tc.expectedMime {
			t.Errorf("GET %s Content-Type = %q; expected %q", tc.urlPath, contentType, tc.expectedMime)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read body for %s: %v", tc.urlPath, err)
		}

		expectedOnDiskPath := filepath.Join(env.OutputDir, tc.expectedFileName)
		diskContent, err := os.ReadFile(expectedOnDiskPath)
		if err != nil {
			t.Fatalf("failed to read output file on disk %s: %v", expectedOnDiskPath, err)
		}

		if !bytes.Equal(body, diskContent) {
			t.Errorf("GET %s body does not match disk content byte for byte", tc.urlPath)
		}
	}
}

// TestDefaultCacheDirQuedaFueraDelProyecto afirma que min.DefaultCacheDir(root)
// devuelve una ruta que no tiene a root como prefijo, y que dos raíces
// distintas dan rutas distintas.
func TestDefaultCacheDirQuedaFueraDelProyecto(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()

	cache1, err := min.DefaultCacheDir(root1)
	if err != nil {
		t.Fatalf("DefaultCacheDir(root1) failed: %v", err)
	}

	cache2, err := min.DefaultCacheDir(root2)
	if err != nil {
		t.Fatalf("DefaultCacheDir(root2) failed: %v", err)
	}

	cleanRoot1 := filepath.Clean(root1)
	cleanCache1 := filepath.Clean(cache1)
	if strings.HasPrefix(cleanCache1, cleanRoot1) {
		t.Errorf("cache dir %q is inside root dir %q", cache1, root1)
	}

	if cache1 == cache2 {
		t.Errorf("different roots %q and %q generated identical cache dir %q", root1, root2, cache1)
	}
}
