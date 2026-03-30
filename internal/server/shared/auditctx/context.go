package auditctx

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type contextKey struct{}

type RequestContext struct {
	Method    string
	Path      string
	IPAddress string
	UserAgent string
}

func WithRequestContext(ctx context.Context, meta RequestContext) context.Context {
	return context.WithValue(ctx, contextKey{}, meta)
}

func RequestContextFromContext(ctx context.Context) (RequestContext, bool) {
	meta, ok := ctx.Value(contextKey{}).(RequestContext)
	return meta, ok
}

func BuildRequestContext(r *http.Request) RequestContext {
	return RequestContext{
		Method:    strings.TrimSpace(r.Method),
		Path:      strings.TrimSpace(r.URL.Path),
		IPAddress: normalizeRemoteIP(r.RemoteAddr),
		UserAgent: strings.TrimSpace(r.UserAgent()),
	}
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
