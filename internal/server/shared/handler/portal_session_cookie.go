package handler

import (
	"net/http"
	"strings"
)

const authSessionCookiePrefix = "ppvt_session_"
const legacyAuthSessionCookieName = "ppvt_auth_session"
const pendingLoginChallengeCookieName = "ppvt_login_challenge"

func authSessionCookieName(organizationID string) string {
	normalized := strings.TrimSpace(organizationID)
	if normalized == "" {
		return legacyAuthSessionCookieName
	}
	return authSessionCookiePrefix + normalized
}

func readAuthSessionCookie(r *http.Request, organizationID string) string {
	cookie, err := r.Cookie(authSessionCookieName(organizationID))
	if err != nil {
		return ""
	}
	return cookie.Value
}

func writeAuthSessionCookie(w http.ResponseWriter, r *http.Request, organizationID, sessionID string) {
	if sessionID == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName(organizationID),
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesSecureTransport(r),
		MaxAge:   86400,
	})
	clearAuthSessionCookieByName(w, r, legacyAuthSessionCookieName)
}

func clearAuthSessionCookieByName(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesSecureTransport(r),
		MaxAge:   -1,
	})
}

func clearAuthSessionCookie(w http.ResponseWriter, r *http.Request, organizationID string) {
	clearAuthSessionCookieByName(w, r, authSessionCookieName(organizationID))
	clearAuthSessionCookieByName(w, r, legacyAuthSessionCookieName)
}

func readAnyAuthSessionCookie(r *http.Request) string {
	for _, cookie := range r.Cookies() {
		if strings.HasPrefix(cookie.Name, authSessionCookiePrefix) && strings.TrimSpace(cookie.Value) != "" {
			return cookie.Value
		}
	}
	cookie, err := r.Cookie(legacyAuthSessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func clearAllAuthSessionCookies(w http.ResponseWriter, r *http.Request) {
	clearAuthSessionCookieByName(w, r, legacyAuthSessionCookieName)
	for _, cookie := range r.Cookies() {
		if strings.HasPrefix(cookie.Name, authSessionCookiePrefix) {
			clearAuthSessionCookieByName(w, r, cookie.Name)
		}
	}
}

func readPendingLoginChallengeCookie(r *http.Request) string {
	cookie, err := r.Cookie(pendingLoginChallengeCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func writePendingLoginChallengeCookie(w http.ResponseWriter, r *http.Request, challenge string) {
	if challenge == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     pendingLoginChallengeCookieName,
		Value:    challenge,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesSecureTransport(r),
		MaxAge:   600,
	})
}

func clearPendingLoginChallengeCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     pendingLoginChallengeCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesSecureTransport(r),
		MaxAge:   -1,
	})
}

func ReadAuthSessionCookie(r *http.Request, organizationID string) string {
	return readAuthSessionCookie(r, organizationID)
}

func ReadAnyAuthSessionCookie(r *http.Request) string {
	return readAnyAuthSessionCookie(r)
}

func WriteAuthSessionCookie(w http.ResponseWriter, r *http.Request, organizationID, sessionID string) {
	writeAuthSessionCookie(w, r, organizationID, sessionID)
}

func ClearAuthSessionCookie(w http.ResponseWriter, r *http.Request, organizationID string) {
	clearAuthSessionCookie(w, r, organizationID)
}

func ClearAllAuthSessionCookies(w http.ResponseWriter, r *http.Request) {
	clearAllAuthSessionCookies(w, r)
}

func ReadPendingLoginChallengeCookie(r *http.Request) string {
	return readPendingLoginChallengeCookie(r)
}

func WritePendingLoginChallengeCookie(w http.ResponseWriter, r *http.Request, challenge string) {
	writePendingLoginChallengeCookie(w, r, challenge)
}

func ClearPendingLoginChallengeCookie(w http.ResponseWriter, r *http.Request) {
	clearPendingLoginChallengeCookie(w, r)
}
