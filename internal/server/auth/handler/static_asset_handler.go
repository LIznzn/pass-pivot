package handler

import (
	"bytes"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	authui "pass-pivot/internal/server/auth/ui"
)

func StaticAssetHandler(filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content, err := authui.ReadAsset(filename)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if contentType := mime.TypeByExtension(filepath.Ext(filename)); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		http.ServeContent(w, r, filepath.Base(filename), staticAssetModTime, bytes.NewReader(content))
	}
}

func StaticAssetPrefixHandler(routePrefix, distPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, routePrefix)
		name = strings.TrimPrefix(name, "/")
		if name == "" || strings.Contains(name, "..") {
			http.NotFound(w, r)
			return
		}
		assetName := pathJoin(distPrefix, name)
		content, err := authui.ReadAsset(assetName)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if contentType := mime.TypeByExtension(filepath.Ext(assetName)); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		http.ServeContent(w, r, filepath.Base(assetName), staticAssetModTime, bytes.NewReader(content))
	}
}

var staticAssetModTime time.Time

func pathJoin(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return strings.TrimPrefix(prefix, "/") + "/" + strings.TrimPrefix(name, "/")
}
