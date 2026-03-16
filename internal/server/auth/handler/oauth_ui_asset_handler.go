package handler

import (
	"net/http"
	"os"
	"path/filepath"
)

func OAuthUIAssetHandler(filename string) http.HandlerFunc {
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
