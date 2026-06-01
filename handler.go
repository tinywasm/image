//go:build !wasm

package image

import "sync"

type ParsedAsset struct {
	AbsPath  string  // absolute path: moduleDir + "/" + asset.Path
	Variants Variant // resolved bitmask value
	Alt      string
	BaseName string // base name without extension: "logo", "hero"
}
type Handler struct {
	mu            sync.Mutex
	config        *Config
	log           func(messages ...any)
	listModulesFn func(rootDir string) ([]string, error)
}
type Config struct {
	RootDir   string
	OutputDir string
	Quality   int
}

func New(c *Config) *Handler {
	return &Handler{
		config: c,
		log:    func(messages ...any) {},
	}
}
func (h *Handler) SetLog(fn func(messages ...any)) {
	h.log = fn
}
func (h *Handler) SetListModulesFn(fn func(rootDir string) ([]string, error)) {
	h.listModulesFn = fn
}
func (h *Handler) Name() string           { return "IMAGEMIN" }
func (h *Handler) Logger(messages ...any) { h.log(messages...) }
func (h *Handler) UnobservedFiles() []string {
	return []string{h.config.OutputDir}
}
