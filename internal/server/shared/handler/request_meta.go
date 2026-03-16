package handler

import (
	"net"
	"strings"
)

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
