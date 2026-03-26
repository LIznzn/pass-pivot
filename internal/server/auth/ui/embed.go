package ui

import (
	"embed"
	"io/fs"
	"path"
	"strings"
)

//go:embed dist dist/* dist/assets/* dist/assets/*.css dist/assets/*.js
var embeddedFiles embed.FS

func ReadIndex() ([]byte, error) {
	return fs.ReadFile(embeddedFiles, "dist/index.html")
}

func ReadAsset(name string) ([]byte, error) {
	normalized, ok := normalizeAssetName(name)
	if !ok {
		return nil, fs.ErrNotExist
	}
	return fs.ReadFile(embeddedFiles, path.Join("dist", normalized))
}

func normalizeAssetName(name string) (string, bool) {
	trimmed := strings.TrimSpace(name)
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return "", false
	}
	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == "" || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", false
	}
	return cleaned, true
}
