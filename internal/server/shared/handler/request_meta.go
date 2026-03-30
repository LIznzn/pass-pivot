package handler

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type auditRequestContextKey struct{}

type AuditRequestContext struct {
	Method    string
	Path      string
	IPAddress string
	UserAgent string
}

func normalizeRemoteIP(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(remoteAddr, "[]")
}

func NormalizeRemoteIP(remoteAddr string) string {
	return normalizeRemoteIP(remoteAddr)
}

func WithAuditRequestContext(ctx context.Context, meta AuditRequestContext) context.Context {
	return context.WithValue(ctx, auditRequestContextKey{}, meta)
}

func AuditRequestContextFromContext(ctx context.Context) (AuditRequestContext, bool) {
	meta, ok := ctx.Value(auditRequestContextKey{}).(AuditRequestContext)
	return meta, ok
}

func BuildAuditRequestContext(r *http.Request) AuditRequestContext {
	return AuditRequestContext{
		Method:    strings.TrimSpace(r.Method),
		Path:      strings.TrimSpace(r.URL.Path),
		IPAddress: NormalizeRemoteIP(r.RemoteAddr),
		UserAgent: strings.TrimSpace(r.UserAgent()),
	}
}
