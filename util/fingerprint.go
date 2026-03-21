package util

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

func GenerateFingerprint() (string, error) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return "", err
	}
	sum := sha256.Sum256(seed)
	return strings.ToUpper(hex.EncodeToString(sum[:])), nil
}

func SignFingerprint(rawFingerprint, secret string) (string, error) {
	if rawFingerprint == "" {
		return "", errors.New("raw fingerprint is required")
	}
	if secret == "" {
		return "", errors.New("fingerprint secret is required")
	}
	payload := base64.RawURLEncoding.EncodeToString([]byte(rawFingerprint))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + signature, nil
}

func VerifyFingerprint(signedFingerprint, secret string) (string, bool) {
	if signedFingerprint == "" || secret == "" {
		return "", false
	}
	parts := strings.Split(signedFingerprint, ".")
	if len(parts) != 2 {
		return "", false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, expected) {
		return "", false
	}
	rawFingerprint, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	return string(rawFingerprint), true
}
