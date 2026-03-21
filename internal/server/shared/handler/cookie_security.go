package handler

import (
	"net/http"
	"strings"
)

func requestUsesSecureTransport(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	proto := strings.TrimSpace(strings.ToLower(r.Header.Get("X-Forwarded-Proto")))
	return proto == "https"
}
