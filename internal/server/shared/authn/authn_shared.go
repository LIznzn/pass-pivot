package authn

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"pass-pivot/internal/model"
)

func CompactTokens(pair *TokenPair) []model.Token {
	items := make([]model.Token, 0, 2)
	if pair == nil {
		return items
	}
	if pair.AccessToken != nil {
		items = append(items, *pair.AccessToken)
	}
	if pair.RefreshToken != nil {
		items = append(items, *pair.RefreshToken)
	}
	return items
}

func RecoveryCodes() []string {
	codes := make([]string, 0, 10)
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	const digits = "0123456789"
	const alphabet = letters + digits
	for i := 0; i < 10; i++ {
		for {
			var builder strings.Builder
			builder.Grow(5)
			hasLetter := false
			hasDigit := false
			for j := 0; j < 5; j++ {
				ch := alphabet[rand.IntN(len(alphabet))]
				if strings.ContainsRune(letters, rune(ch)) {
					hasLetter = true
				}
				if strings.ContainsRune(digits, rune(ch)) {
					hasDigit = true
				}
				builder.WriteByte(ch)
			}
			if hasLetter && hasDigit {
				codes = append(codes, fmt.Sprintf("%s", builder.String()))
				break
			}
		}
	}
	return codes
}
