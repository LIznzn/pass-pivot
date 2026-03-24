package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func StaticAssetHandler(filename string) http.HandlerFunc {
	distDir := filepath.Join("web", "auth", "dist")
	assetPath := filepath.Join(distDir, filename)
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(assetPath); err != nil {
			http.Error(w, "auth ui assets are missing, run `cd web && npm run build:auth`", http.StatusServiceUnavailable)
			return
		}
		http.ServeFile(w, r, assetPath)
	}
}

func StaticAssetPrefixHandler(routePrefix, distPrefix string) http.HandlerFunc {
	distDir := filepath.Join("web", "auth", "dist")
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, routePrefix)
		name = strings.TrimPrefix(name, "/")
		if name == "" || strings.Contains(name, "..") {
			http.NotFound(w, r)
			return
		}
		assetPath := filepath.Join(distDir, distPrefix, name)
		if _, err := os.Stat(assetPath); err != nil {
			http.Error(w, "auth ui assets are missing, run `cd web && npm run build:auth`", http.StatusServiceUnavailable)
			return
		}
		http.ServeFile(w, r, assetPath)
	}
}
