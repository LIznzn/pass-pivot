package system

import (
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, system *Handler) {
	mux.HandleFunc("POST /api/system/v1/health", system.Health)
}
