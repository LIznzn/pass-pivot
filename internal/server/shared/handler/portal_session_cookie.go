package handler

import "net/http"

const portalSessionCookieName = "ppvt_portal_session"

func readPortalSessionCookie(r *http.Request) string {
	cookie, err := r.Cookie(portalSessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func writePortalSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string) {
	if sessionID == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     portalSessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesSecureTransport(r),
		MaxAge:   86400,
	})
}

func clearPortalSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     portalSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestUsesSecureTransport(r),
		MaxAge:   -1,
	})
}

func ReadPortalSessionCookie(r *http.Request) string {
	return readPortalSessionCookie(r)
}

func WritePortalSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string) {
	writePortalSessionCookie(w, r, sessionID)
}

func ClearPortalSessionCookie(w http.ResponseWriter, r *http.Request) {
	clearPortalSessionCookie(w, r)
}
