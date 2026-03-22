package service

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"pass-pivot/internal/model"
)

func TestBuildImplicitRedirectUsesFragment(t *testing.T) {
	redirectTarget, err := buildImplicitRedirect(
		"http://localhost:18093/callback",
		"state-1",
		[]model.Token{{
			Type:      "access_token",
			Token:     "access-token-value",
			Scope:     "openid profile",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}},
		"id-token-value",
	)
	if err != nil {
		t.Fatalf("buildImplicitRedirect() error = %v", err)
	}
	if !strings.Contains(redirectTarget, "#") {
		t.Fatalf("redirect target missing fragment: %q", redirectTarget)
	}

	parsed, err := url.Parse(redirectTarget)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	values, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		t.Fatalf("url.ParseQuery() error = %v", err)
	}
	if values.Get("state") != "state-1" {
		t.Fatalf("state = %q, want %q", values.Get("state"), "state-1")
	}
	if values.Get("access_token") != "access-token-value" {
		t.Fatalf("access_token = %q, want %q", values.Get("access_token"), "access-token-value")
	}
	if values.Get("id_token") != "id-token-value" {
		t.Fatalf("id_token = %q, want %q", values.Get("id_token"), "id-token-value")
	}
	if values.Get("token_type") != "Bearer" {
		t.Fatalf("token_type = %q, want %q", values.Get("token_type"), "Bearer")
	}
}

func TestRedirectWithOAuthErrorForImplicitUsesFragment(t *testing.T) {
	service := &OIDCService{}

	redirectTarget := service.redirectWithOAuthErrorForResponseType(
		"http://localhost:18093/callback",
		"id_token token",
		"invalid_request",
		"state-2",
		"nonce is required",
	)
	if !strings.Contains(redirectTarget, "#") {
		t.Fatalf("redirect target missing fragment: %q", redirectTarget)
	}

	parsed, err := url.Parse(redirectTarget)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	values, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		t.Fatalf("url.ParseQuery() error = %v", err)
	}
	if values.Get("error") != "invalid_request" {
		t.Fatalf("error = %q, want %q", values.Get("error"), "invalid_request")
	}
	if values.Get("state") != "state-2" {
		t.Fatalf("state = %q, want %q", values.Get("state"), "state-2")
	}
}
