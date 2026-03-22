package manage

import (
	"testing"

	"pass-pivot/internal/model"
)

func TestValidateApplicationProtocolAllowsImplicitPublicClient(t *testing.T) {
	app := model.Application{
		ApplicationType:          "web",
		GrantType:                []string{"implicit"},
		ClientAuthenticationType: "none",
		TokenType:                []string{"access_token", "id_token"},
		AccessTokenTTLMinutes:    10,
		RefreshTokenTTLHours:     168,
	}

	if err := validateApplicationProtocol(app); err != nil {
		t.Fatalf("validateApplicationProtocol() error = %v", err)
	}
}
