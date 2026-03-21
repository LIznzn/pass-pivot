package handler

import "net/http"

const fingerprintCookieName = "ppvt_device_fingerprint"

func readFingerprintCookie(r *http.Request) string {
	cookie, err := r.Cookie(fingerprintCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func writeFingerprintCookie(w http.ResponseWriter, r *http.Request, fingerprint string) {
	if fingerprint == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     fingerprintCookieName,
		Value:    fingerprint,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesSecureTransport(r),
		MaxAge:   86400 * 365,
	})
}

func ReadFingerprintCookie(r *http.Request) string {
	return readFingerprintCookie(r)
}

func WriteFingerprintCookie(w http.ResponseWriter, r *http.Request, fingerprint string) {
	writeFingerprintCookie(w, r, fingerprint)
}
