package handler

import (
	"context"
	"net/http"
)

type trustedForwardHeadersContextKey struct{}

func WithTrustedForwardHeaders(ctx context.Context) context.Context {
	return context.WithValue(ctx, trustedForwardHeadersContextKey{}, true)
}

func TrustedForwardHeadersFromContext(ctx context.Context) bool {
	allowed, _ := ctx.Value(trustedForwardHeadersContextKey{}).(bool)
	return allowed
}

func OriginalRemoteAddr(r *http.Request) string {
	if TrustedForwardHeadersFromContext(r.Context()) {
		if forwarded := r.Header.Get("X-PPVT-Original-Remote-Addr"); forwarded != "" {
			return forwarded
		}
	}
	return r.RemoteAddr
}

func OriginalUserAgent(r *http.Request) string {
	if TrustedForwardHeadersFromContext(r.Context()) {
		if forwarded := r.Header.Get("X-PPVT-Original-User-Agent"); forwarded != "" {
			return forwarded
		}
	}
	return r.UserAgent()
}

func SanitizeInternalForwardHeaders(r *http.Request) {
	if TrustedForwardHeadersFromContext(r.Context()) {
		return
	}
	r.Header.Del("X-PPVT-Original-Remote-Addr")
	r.Header.Del("X-PPVT-Original-User-Agent")
}

func originalRemoteAddr(r *http.Request) string {
	return OriginalRemoteAddr(r)
}

func originalUserAgent(r *http.Request) string {
	return OriginalUserAgent(r)
}
