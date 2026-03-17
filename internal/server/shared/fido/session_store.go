package fido

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"pass-pivot/internal/model"
	"pass-pivot/util"
)

type webauthnChallengeRecord struct {
	UserID         string
	OrganizationID string
	SessionID      string
	FlowType       string
	ChallengeID    string
	Challenge      string
	ExpiresAt      time.Time
}

func (s *Service) storeWebAuthnSession(_ context.Context, organizationID, userID, sessionID, flow string, sessionData *webauthn.SessionData, options any) (string, any, error) {
	raw, err := json.Marshal(sessionData)
	if err != nil {
		return "", nil, err
	}
	challengeID, err := util.RandomToken(20)
	if err != nil {
		return "", nil, err
	}
	record := webauthnChallengeRecord{
		UserID:         userID,
		OrganizationID: organizationID,
		SessionID:      sessionID,
		FlowType:       flow,
		ChallengeID:    challengeID,
		Challenge:      string(raw),
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}
	s.storeInMemoryWebAuthnSession(record)
	return challengeID, options, nil
}

func (s *Service) loadWebAuthnSession(ctx context.Context, challengeID, flow string) (webauthnChallengeRecord, *webauthn.SessionData, model.User, webauthnUser, error) {
	record, ok := s.getWebAuthnSession(challengeID)
	if !ok || record.FlowType != flow {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, errors.New("webauthn challenge not found")
	}
	if record.ExpiresAt.Before(time.Now()) {
		s.deleteWebAuthnSession(challengeID)
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, errors.New("webauthn challenge expired")
	}
	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(record.Challenge), &sessionData); err != nil {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, err
	}
	user, webUser, err := s.loadWebAuthnUser(ctx, record.UserID, webauthnUsageForFlow(record.FlowType))
	if err != nil {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, err
	}
	return record, &sessionData, user, webUser, nil
}

func (s *Service) loadWebAuthnSessionByPrefix(ctx context.Context, challengeID, flowPrefix string) (webauthnChallengeRecord, *webauthn.SessionData, model.User, webauthnUser, error) {
	record, ok := s.getWebAuthnSession(challengeID)
	if !ok || !strings.HasPrefix(record.FlowType, flowPrefix) {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, errors.New("webauthn challenge not found")
	}
	if record.ExpiresAt.Before(time.Now()) {
		s.deleteWebAuthnSession(challengeID)
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, errors.New("webauthn challenge expired")
	}
	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(record.Challenge), &sessionData); err != nil {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, err
	}
	user, webUser, err := s.loadWebAuthnUser(ctx, record.UserID, "all")
	if err != nil {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, err
	}
	return record, &sessionData, user, webUser, nil
}

func webauthnUsageForFlow(flow string) string {
	switch {
	case strings.HasPrefix(flow, "registration:"):
		return "all"
	case strings.HasPrefix(flow, "assertion:"):
		return strings.TrimPrefix(flow, "assertion:")
	default:
		return "all"
	}
}

func (s *Service) storeInMemoryWebAuthnSession(record webauthnChallengeRecord) {
	s.cleanupExpiredWebAuthnSessions()
	s.webauthnMu.Lock()
	defer s.webauthnMu.Unlock()
	s.webauthnSessions[record.ChallengeID] = record
}

func (s *Service) getWebAuthnSession(challengeID string) (webauthnChallengeRecord, bool) {
	s.cleanupExpiredWebAuthnSessions()
	s.webauthnMu.RLock()
	defer s.webauthnMu.RUnlock()
	record, ok := s.webauthnSessions[challengeID]
	return record, ok
}

func (s *Service) deleteWebAuthnSession(challengeID string) {
	s.webauthnMu.Lock()
	defer s.webauthnMu.Unlock()
	delete(s.webauthnSessions, challengeID)
}

func (s *Service) cleanupExpiredWebAuthnSessions() {
	now := time.Now()
	s.webauthnMu.Lock()
	defer s.webauthnMu.Unlock()
	for challengeID, record := range s.webauthnSessions {
		if record.ExpiresAt.Before(now) {
			delete(s.webauthnSessions, challengeID)
		}
	}
}
