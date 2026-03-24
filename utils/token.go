package utils

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	mathrand "math/rand/v2"
	"strings"
)

const humanTokenLetters = "ABCDEFGHJKLMNPQRSTUVWXYZ"
const humanTokenDigits = "0123456789"
const humanTokenAlphabet = humanTokenLetters + humanTokenDigits

func RandomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := cryptorand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func RandomHumanToken(size int) string {
	if size <= 0 {
		return ""
	}
	for {
		var builder strings.Builder
		builder.Grow(size)
		hasLetter := false
		hasDigit := false
		for i := 0; i < size; i++ {
			ch := humanTokenAlphabet[mathrand.IntN(len(humanTokenAlphabet))]
			if strings.ContainsRune(humanTokenLetters, rune(ch)) {
				hasLetter = true
			}
			if strings.ContainsRune(humanTokenDigits, rune(ch)) {
				hasDigit = true
			}
			builder.WriteByte(ch)
		}
		if hasLetter && hasDigit {
			return builder.String()
		}
	}
}
