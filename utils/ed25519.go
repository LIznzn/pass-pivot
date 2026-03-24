package utils

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
)

func GenerateEd25519KeyMaterial() (publicKey, privateSeed string, err error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return EncodeEd25519PublicKey(public), EncodeEd25519PrivateSeed(private.Seed()), nil
}

func GenerateEd25519PrivateSeed() (string, error) {
	seed := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(seed); err != nil {
		return "", err
	}
	return EncodeEd25519PrivateSeed(seed), nil
}

func DeriveEd25519PublicKey(seed string) (string, error) {
	privateKey, err := NewEd25519PrivateKeyFromSeed(seed)
	if err != nil {
		return "", err
	}
	return EncodeEd25519PublicKey(privateKey.Public().(ed25519.PublicKey)), nil
}

func NewEd25519PrivateKeyFromSeed(seed string) (ed25519.PrivateKey, error) {
	rawSeed, err := DecodeEd25519PrivateSeed(seed)
	if err != nil {
		return nil, err
	}
	return ed25519.NewKeyFromSeed(rawSeed), nil
}

func EncodeEd25519PublicKey(publicKey ed25519.PublicKey) string {
	return hex.EncodeToString(publicKey)
}

func EncodeEd25519PrivateSeed(seed []byte) string {
	return hex.EncodeToString(seed)
}

func ParseEd25519PublicKey(value string) (ed25519.PublicKey, error) {
	raw, err := hex.DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid ed25519 public key")
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, errors.New("invalid ed25519 public key length")
	}
	return ed25519.PublicKey(raw), nil
}

func DecodeEd25519PrivateSeed(value string) ([]byte, error) {
	raw, err := hex.DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid ed25519 private seed")
	}
	if len(raw) != ed25519.SeedSize {
		return nil, errors.New("invalid ed25519 private seed length")
	}
	return raw, nil
}
