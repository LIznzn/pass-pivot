package service

import "testing"

func TestBuildOIDCMetadataIncludesDeviceAuthorizationEndpoint(t *testing.T) {
	metadata := buildOIDCMetadata("http://localhost:8091")
	if metadata.DeviceAuthorizationEndpoint != "http://localhost:8091/auth/device_authorization" {
		t.Fatalf("DeviceAuthorizationEndpoint = %q", metadata.DeviceAuthorizationEndpoint)
	}
	var found bool
	for _, item := range metadata.GrantTypesSupported {
		if item == "urn:ietf:params:oauth:grant-type:device_code" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("GrantTypesSupported missing device_code grant: %#v", metadata.GrantTypesSupported)
	}
}

func TestNormalizeUserCode(t *testing.T) {
	if got := normalizeUserCode("abcd efgh"); got != "ABCD-EFGH" {
		t.Fatalf("normalizeUserCode() = %q", got)
	}
}
