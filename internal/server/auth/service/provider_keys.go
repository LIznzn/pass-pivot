package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"

	"github.com/go-jose/go-jose/v4"
	"gorm.io/gorm"

	"pass-pivot/internal/model"
)

type ProviderKeys struct {
	SigningKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	KeyID      string
}

type ClientAssertionKeys struct {
	SigningKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	KeyID      string
}

func (p *ClientAssertionKeys) PublicJWK() jose.JSONWebKey {
	return jose.JSONWebKey{
		Key:       p.PublicKey,
		KeyID:     p.KeyID,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
}

type ProviderKeyStore struct {
	db *gorm.DB
}

func NewProviderKeyStore(db *gorm.DB) *ProviderKeyStore {
	return &ProviderKeyStore{db: db}
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

func NewOrganizationSigningKey(organizationID string) (*model.OrganizationSigningKey, error) {
	if organizationID == "" {
		return nil, errors.New("organization id is required")
	}
	keys, err := NewProviderKeys()
	if err != nil {
		return nil, err
	}
	privateKeyBase64, err := EncodeRSAPrivateKeyBase64(keys.SigningKey)
	if err != nil {
		return nil, err
	}
	publicKeyBase64, err := EncodeRSAPublicKeyBase64(keys.PublicKey)
	if err != nil {
		return nil, err
	}
	return &model.OrganizationSigningKey{
		OrganizationID:   organizationID,
		PrivateKeyBase64: privateKeyBase64,
		PublicKeyBase64:  publicKeyBase64,
		KeyID:            keys.KeyID,
		Status:           "active",
	}, nil
}

func NewApplicationClientKey(applicationID string) (*model.ApplicationKey, string, error) {
	if applicationID == "" {
		return nil, "", errors.New("application id is required")
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
	}
	privateKeyBase64, err := EncodeRSAPrivateKeyBase64(privateKey)
	if err != nil {
		return nil, "", err
	}
	publicKeyBase64, err := EncodeRSAPublicKeyBase64(&privateKey.PublicKey)
	if err != nil {
		return nil, "", err
	}
	return &model.ApplicationKey{
		ApplicationID:    applicationID,
		PublicKeyBase64:  publicKeyBase64,
		PrivateKeyBase64: privateKeyBase64,
		KeyID:            keyIDForRSAPublicKey(&privateKey.PublicKey),
		Status:           "active",
	}, privateKeyBase64, nil
}

func (s *ProviderKeyStore) ProviderKeysForOrganization(ctx context.Context, organizationID string) (*ProviderKeys, error) {
	if organizationID == "" {
		return nil, errors.New("organization id is required")
	}
	record, err := s.loadOrganizationSigningKey(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	return providerKeysFromRecord(record)
}

func (s *ProviderKeyStore) ProviderKeysForApplication(ctx context.Context, applicationID string) (*ProviderKeys, error) {
	if applicationID == "" {
		return nil, errors.New("application id is required")
	}
	organizationID, err := s.applicationOrganizationID(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	return s.ProviderKeysForOrganization(ctx, organizationID)
}

func (s *ProviderKeyStore) ProviderJWKs(ctx context.Context) ([]jose.JSONWebKey, error) {
	var records []model.OrganizationSigningKey
	if err := s.db.WithContext(ctx).
		Where("status = ?", "active").
		Order("created_at ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	keys := make([]jose.JSONWebKey, 0, len(records))
	for _, record := range records {
		item, err := providerKeysFromRecord(&record)
		if err != nil {
			return nil, err
		}
		keys = append(keys, item.PublicJWK())
	}
	return keys, nil
}

func (s *ProviderKeyStore) LoadClientSigningKey(ctx context.Context, applicationID string) (*ClientAssertionKeys, error) {
	if applicationID == "" {
		return nil, errors.New("application id is required")
	}
	record, err := s.loadApplicationKey(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	return clientAssertionSigningKeysFromRecord(record)
}

func (s *ProviderKeyStore) LoadClientVerificationKey(ctx context.Context, applicationID string, fallbackPublicKey string) (*ClientAssertionKeys, error) {
	if applicationID == "" {
		return nil, errors.New("application id is required")
	}
	record, err := s.loadApplicationKey(ctx, applicationID)
	if err != nil {
		if fallbackPublicKey == "" {
			return nil, err
		}
		return loadClientVerificationKeyFromPublicKey(fallbackPublicKey)
	}
	return clientAssertionVerificationKeysFromRecord(record)
}

func EncodeRSAPrivateKeyBase64(privateKey *rsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", errors.New("private key is required")
	}
	raw, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func ParseRSAPrivateKeyBase64(value string) (*rsa.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid rsa private key base64")
	}
	key, err := x509.ParsePKCS8PrivateKey(raw)
	if err != nil {
		return nil, errors.New("invalid rsa private key")
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("invalid rsa private key type")
	}
	return rsaKey, nil
}

func EncodeRSAPublicKeyBase64(publicKey *rsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", errors.New("public key is required")
	}
	raw, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func ParseRSAPublicKeyBase64(value string) (*rsa.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid rsa public key base64")
	}
	key, err := x509.ParsePKIXPublicKey(raw)
	if err != nil {
		return nil, errors.New("invalid rsa public key")
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid rsa public key type")
	}
	return rsaKey, nil
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

func providerKeysFromRecord(record *model.OrganizationSigningKey) (*ProviderKeys, error) {
	if record == nil {
		return nil, errors.New("organization signing key is required")
	}
	privateKey, err := ParseRSAPrivateKeyBase64(record.PrivateKeyBase64)
	if err != nil {
		return nil, err
	}
	publicKey, err := ParseRSAPublicKeyBase64(record.PublicKeyBase64)
	if err != nil {
		return nil, err
	}
	return &ProviderKeys{
		SigningKey: privateKey,
		PublicKey:  publicKey,
		KeyID:      record.KeyID,
	}, nil
}

func clientAssertionSigningKeysFromRecord(record *model.ApplicationKey) (*ClientAssertionKeys, error) {
	if record == nil {
		return nil, errors.New("application key is required")
	}
	privateKey, err := ParseRSAPrivateKeyBase64(record.PrivateKeyBase64)
	if err != nil {
		return nil, err
	}
	publicKey := &privateKey.PublicKey
	if record.PublicKeyBase64 != "" {
		encodedPublicKey, err := EncodeRSAPublicKeyBase64(publicKey)
		if err != nil {
			return nil, err
		}
		if record.PublicKeyBase64 != encodedPublicKey {
			return nil, errors.New("application client public key does not match derived key")
		}
	}
	return &ClientAssertionKeys{
		SigningKey: privateKey,
		PublicKey:  publicKey,
		KeyID:      record.KeyID,
	}, nil
}

func clientAssertionVerificationKeysFromRecord(record *model.ApplicationKey) (*ClientAssertionKeys, error) {
	if record == nil {
		return nil, errors.New("application key is required")
	}
	return loadClientVerificationKeyFromPublicKey(record.PublicKeyBase64)
}

func loadClientVerificationKeyFromPublicKey(publicKey string) (*ClientAssertionKeys, error) {
	key, err := ParseRSAPublicKeyBase64(publicKey)
	if err != nil {
		return nil, err
	}
	return &ClientAssertionKeys{
		PublicKey: key,
		KeyID:     keyIDForRSAPublicKey(key),
	}, nil
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

func (s *ProviderKeyStore) loadOrganizationSigningKey(ctx context.Context, organizationID string) (*model.OrganizationSigningKey, error) {
	var record model.OrganizationSigningKey
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND status = ?", organizationID, "active").
		Order("created_at DESC").
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("organization signing key is not configured")
		}
		return nil, err
	}
	return &record, nil
}

func (s *ProviderKeyStore) loadApplicationKey(ctx context.Context, applicationID string) (*model.ApplicationKey, error) {
	var record model.ApplicationKey
	if err := s.db.WithContext(ctx).
		Where("application_id = ? AND status = ?", applicationID, "active").
		Order("created_at DESC").
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("application client key is not configured")
		}
		return nil, err
	}
	return &record, nil
}

func (s *ProviderKeyStore) applicationOrganizationID(ctx context.Context, applicationID string) (string, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Select("id", "project_id").First(&app, "id = ?", applicationID).Error; err != nil {
		return "", err
	}
	var project model.Project
	if err := s.db.WithContext(ctx).Select("id", "organization_id").First(&project, "id = ?", app.ProjectID).Error; err != nil {
		return "", err
	}
	return project.OrganizationID, nil
}
