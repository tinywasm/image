//go:build !wasm

package min

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Artifact es una imagen ya procesada, lista para servirse o para escribirse.
// Path es la ruta relativa a la raíz del sitio y ES la URL: "img/foto.M.jpg".
type Artifact struct {
	Path      string
	Mediatype string
	Content   []byte
}

// cacheNamespace es el subdirectorio del caché de usuario donde viven las
// imágenes convertidas de todos los proyectos.
const cacheNamespace = "tinywasm/images"

// imgDir es el prefijo bajo el que se sirve toda imagen procesada.
const imgDir = "img"

// Mediatypes de las tres codificaciones que este paquete produce.
const (
	MediatypeJPEG = "image/jpeg"
	MediatypeWebP = "image/webp"
	MediatypeSVG  = "image/svg+xml"
)

// mediatypeFor devuelve el tipo MIME a partir de la extensión del archivo.
func mediatypeFor(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg":
		return MediatypeJPEG
	case ".webp":
		return MediatypeWebP
	case ".svg":
		return MediatypeSVG
	default:
		return ""
	}
}

// Artifacts devuelve el contenido de cada imagen producida por el último
// LoadImages/ReloadModule.
//
// Existe porque el consumidor de desarrollo sirve el sitio desde memoria. Sin
// esto, la única forma de servir una imagen era que este paquete la dejara en
// un directorio dentro del proyecto del usuario, y ese directorio pasaba a ser
// una segunda salida del sitio que nadie pidió.
//
// Un origen cuya salida no se puede leer se omite en silencio: no es un error
// del consumidor, y el log ya reportó el fallo de conversión.
func (h *Handler) Artifacts() []Artifact {
	h.mu.Lock()
	assets := append([]ParsedAsset(nil), h.assets...)
	outputDir := h.config.OutputDir
	h.mu.Unlock()

	var result []Artifact
	for _, asset := range assets {
		names := outputNames(asset)
		for _, name := range names {
			mt := mediatypeFor(name)
			if mt == "" {
				continue
			}
			fullPath := filepath.Join(outputDir, name)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			result = append(result, Artifact{
				Path:      path.Join(imgDir, name),
				Mediatype: mt,
				Content:   content,
			})
		}
	}
	return result
}

// DefaultCacheDir es dónde guardar las conversiones de un proyecto cuando la
// salida NO es un entregable: un directorio por proyecto bajo el caché del
// usuario, nunca dentro del proyecto.
//
// La conversión es cara y merece caché entre arranques; lo que no merece es
// que esa caché viva dentro del repositorio del usuario, donde se confunde con
// la salida publicable del sitio.
func DefaultCacheDir(rootDir string) (string, error) {
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("min: no hay directorio de caché de usuario: %w", err)
	}
	hash := sha256.Sum256([]byte(rootDir))
	key := hex.EncodeToString(hash[:])[:16]
	return filepath.Join(userCache, cacheNamespace, key, imgDir), nil
}
