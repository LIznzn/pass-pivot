package handler

import "net/http"

func OriginalRemoteAddr(r *http.Request) string {
	if forwarded := r.Header.Get("X-PPVT-Original-Remote-Addr"); forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func OriginalUserAgent(r *http.Request) string {
	if forwarded := r.Header.Get("X-PPVT-Original-User-Agent"); forwarded != "" {
		return forwarded
	}
	return r.UserAgent()
}

func originalRemoteAddr(r *http.Request) string {
	return OriginalRemoteAddr(r)
}

func originalUserAgent(r *http.Request) string {
	return OriginalUserAgent(r)
}
