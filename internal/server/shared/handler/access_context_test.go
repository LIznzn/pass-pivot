package handler

import (
	"testing"

	"pass-pivot/internal/model"
	coreservice "pass-pivot/internal/server/core/service"
)

func TestCurrentUserIDOrTargetAllowsSelfWithoutRole(t *testing.T) {
	identity := &coreservice.AccessTokenIdentity{
		User: &model.User{BaseModel: model.BaseModel{ID: "user-1"}},
	}

	targetUserID, allowed := CurrentUserIDOrTarget(identity, "")
	if !allowed {
		t.Fatalf("CurrentUserIDOrTarget() allowed = false, want true")
	}
	if targetUserID != "user-1" {
		t.Fatalf("CurrentUserIDOrTarget() targetUserID = %q, want %q", targetUserID, "user-1")
	}

	targetUserID, allowed = CurrentUserIDOrTarget(identity, "user-1")
	if !allowed {
		t.Fatalf("CurrentUserIDOrTarget() allowed = false, want true for self target")
	}
	if targetUserID != "user-1" {
		t.Fatalf("CurrentUserIDOrTarget() targetUserID = %q, want %q for self target", targetUserID, "user-1")
	}
}
