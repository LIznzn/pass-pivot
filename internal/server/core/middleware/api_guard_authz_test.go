package middleware

import "testing"

func TestSkipsUserPolicyCheck(t *testing.T) {
	if !skipsUserPolicyCheck("/api/user/v1/profile/query") {
		t.Fatalf("skipsUserPolicyCheck() = false, want true for /api/user/v1/*")
	}
	if skipsUserPolicyCheck("/api/manage/v1/user/query") {
		t.Fatalf("skipsUserPolicyCheck() = true, want false for non-user API")
	}
}
