package utils

import (
	"golang.org/x/crypto/bcrypt"
)

func HashSecret(secret string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CheckSecret(hash string, secret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}
