package authz

import (
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, authz *Handler) {
	mux.HandleFunc("POST /api/authz/v1/policy/check", authz.CheckPolicy)
	mux.HandleFunc("POST /api/authz/v1/policy/query", authz.ListSubjectPolicies)
}
