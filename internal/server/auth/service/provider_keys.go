package service

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"sync"

	"github.com/go-jose/go-jose/v4"
)

type ProviderKeys struct {
	SigningKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	KeyID      string
}

type ClientAssertionKeys struct {
	SigningKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	KeyID      string
}

func (p *ClientAssertionKeys) PublicJWK() jose.JSONWebKey {
	return jose.JSONWebKey{
		Key:       p.PublicKey,
		KeyID:     p.KeyID,
		Algorithm: string(jose.EdDSA),
		Use:       "sig",
	}
}

type ProviderKeyStore struct {
	mu            sync.Mutex
	instance      *ProviderKeys
	instancePEM   string
	internalSeeds map[string]string
}

func NewProviderKeyStore(internalSeeds map[string]string) *ProviderKeyStore {
	return &ProviderKeyStore{internalSeeds: internalSeeds}
}

func NewProviderKeys() (*ProviderKeys, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return &ProviderKeys{
		SigningKey: key,
		PublicKey:  &key.PublicKey,
		KeyID:      keyIDForRSAPublicKey(&key.PublicKey),
	}, nil
}

func (s *ProviderKeyStore) Instance() (*ProviderKeys, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.instance != nil {
		return s.instance, nil
	}
	item, err := NewProviderKeys()
	if err != nil {
		return nil, err
	}
	s.instance = item
	publicPEM, err := EncodeRSAPublicKeyPEM(item.PublicKey)
	if err != nil {
		return nil, err
	}
	s.instancePEM = publicPEM
	return item, nil
}

func (s *ProviderKeyStore) InstancePublicPEM() (string, error) {
	if _, err := s.Instance(); err != nil {
		return "", err
	}
	return s.instancePEM, nil
}

func GenerateClientKeyMaterial() (publicKey, privateKey string, err error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return EncodeEd25519PublicKey(public), EncodeEd25519PrivateSeed(private.Seed()), nil
}

func (s *ProviderKeyStore) LoadClientVerificationKey(publicKey string) (*ClientAssertionKeys, error) {
	key, err := ParseEd25519PublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	return &ClientAssertionKeys{
		PublicKey: key,
		KeyID:     keyIDForEd25519PublicKey(key),
	}, nil
}

func (s *ProviderKeyStore) LoadInternalClientSigningKey(applicationID, publicKey string) (*ClientAssertionKeys, error) {
	if applicationID == "" {
		return nil, errors.New("application id is required")
	}
	seed, err := s.internalClientSeed(applicationID)
	if err != nil {
		return nil, err
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	derivedPublicKey := privateKey.Public().(ed25519.PublicKey)
	if publicKey != "" && publicKey != EncodeEd25519PublicKey(derivedPublicKey) {
		return nil, errors.New("internal client public key does not match derived key")
	}
	return &ClientAssertionKeys{
		SigningKey: privateKey,
		PublicKey:  derivedPublicKey,
		KeyID:      keyIDForEd25519PublicKey(derivedPublicKey),
	}, nil
}

func GenerateInternalClientPublicKey(seed string) (string, error) {
	rawSeed, err := DecodeEd25519PrivateSeed(seed)
	if err != nil {
		return "", err
	}
	privateKey := ed25519.NewKeyFromSeed(rawSeed)
	return EncodeEd25519PublicKey(privateKey.Public().(ed25519.PublicKey)), nil
}

func GenerateEd25519PrivateSeed() (string, error) {
	seed := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(seed); err != nil {
		return "", err
	}
	return EncodeEd25519PrivateSeed(seed), nil
}

func EncodeRSAPublicKeyPEM(publicKey *rsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", errors.New("public key is required")
	}
	raw, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: raw})), nil
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

func (p *ProviderKeys) JWKS() (map[string]any, error) {
	if p.PublicKey == nil && p.SigningKey != nil {
		p.PublicKey = &p.SigningKey.PublicKey
	}
	if p.PublicKey == nil {
		return nil, errors.New("public key is not configured")
	}
	public := p.PublicJWK()
	return map[string]any{
		"keys": []jose.JSONWebKey{public},
	}, nil
}

func (p *ProviderKeys) PublicJWK() jose.JSONWebKey {
	publicKey := p.PublicKey
	if publicKey == nil && p.SigningKey != nil {
		publicKey = &p.SigningKey.PublicKey
	}
	return jose.JSONWebKey{
		Key:       publicKey,
		KeyID:     p.KeyID,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
}

func keyIDForRSAPublicKey(publicKey *rsa.PublicKey) string {
	if publicKey == nil {
		return ""
	}
	raw, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:8])
}

func keyIDForEd25519PublicKey(publicKey ed25519.PublicKey) string {
	if len(publicKey) == 0 {
		return ""
	}
	sum := sha256.Sum256(publicKey)
	return hex.EncodeToString(sum[:8])
}

func (s *ProviderKeyStore) internalClientSeed(applicationID string) ([]byte, error) {
	value, ok := s.internalSeeds[applicationID]
	if !ok {
		return nil, errors.New("internal client private key is not configured in code")
	}
	return DecodeEd25519PrivateSeed(value)
}
