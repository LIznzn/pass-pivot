package web

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"

	"gorm.io/gorm"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
)

type corsOriginResolver struct {
	db  *gorm.DB
	cfg config.Config
}

func NewCORS(db *gorm.DB, cfg config.Config) func(http.Handler) http.Handler {
	resolver := &corsOriginResolver{db: db, cfg: cfg}
	return resolver.withCORS
}

func (r *corsOriginResolver) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		origin := strings.TrimSpace(req.Header.Get("Origin"))
		if origin != "" && r.isAllowedOrigin(req.Context(), origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (r *corsOriginResolver) isAllowedOrigin(ctx context.Context, origin string) bool {
	normalized, ok := normalizeOrigin(origin)
	if !ok {
		return false
	}
	if isLoopbackOrigin(normalized) {
		return true
	}
	allowed := map[string]struct{}{}
	appendOrigin := func(raw string) {
		if value, valid := normalizeOrigin(raw); valid {
			allowed[value] = struct{}{}
		}
	}

	appendOrigin(r.cfg.AuthURL)
	appendOrigin(r.cfg.CoreURL)

	var organizations []model.Organization
	if err := r.db.WithContext(ctx).Select("domains").Find(&organizations).Error; err == nil {
		for _, organization := range organizations {
			for _, item := range organization.Domains {
				if !item.Verified {
					continue
				}
				appendOrigin("https://" + item.Host)
				appendOrigin("http://" + item.Host)
			}
		}
	}

	_, exists := allowed[normalized]
	return exists
}

func normalizeOrigin(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host), true
}

func isLoopbackOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
