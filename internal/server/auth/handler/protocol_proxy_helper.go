package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type internalAPIError struct {
	Error string `json:"error"`
}

func (h *OIDCHandler) callSystemAPI(r *http.Request, path string, payload any) ([]byte, error) {
	return h.callInternalJSONAPI(r, "authn-api", "/api/authn", http.MethodPost, path, payload, nil)
}

func (h *OIDCHandler) callInternalJSONAPI(r *http.Request, internalAppName, audiencePath, method, path string, payload any, extraHeaders map[string]string) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}
	audience := strings.TrimRight(h.cfg.CoreURL, "/") + audiencePath
	clientID, assertion, err := h.BuildNamedClientAssertion(r.Context(), internalAppName, audience)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(r.Context(), method, strings.TrimRight(h.cfg.CoreURL, "/")+path, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-PPVT-Client-ID", clientID)
	req.Header.Set("X-PPVT-Client-Assertion-Type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	req.Header.Set("X-PPVT-Client-Assertion", assertion)
	req.Header.Set("X-PPVT-Original-Remote-Addr", r.RemoteAddr)
	req.Header.Set("X-PPVT-Original-User-Agent", r.UserAgent())
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	for key, value := range extraHeaders {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, serviceErrorFromBody(bodyBytes)
	}
	return bodyBytes, nil
}
