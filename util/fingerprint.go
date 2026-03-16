package util

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

func GenerateFingerprint() (string, error) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return "", err
	}
	sum := sha1.Sum(seed)
	return strings.ToUpper(hex.EncodeToString(sum[:])), nil
}
