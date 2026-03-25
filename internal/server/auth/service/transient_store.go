package service

import (
	"sync"
	"time"

	"pass-pivot/internal/model"
)

type transientAuthorizationCodeRecord struct {
	model.AuthorizationCode
}

type transientMFAChallengeRecord struct {
	model.MFAChallenge
}

type transientExternalAuthStateRecord struct {
	model.ExternalAuthState
}

type transientMFAVerificationAttemptRecord struct {
	Count     int
	ExpiresAt time.Time
}

var transientStore = struct {
	mu                 sync.RWMutex
	authorizationCodes map[string]transientAuthorizationCodeRecord
	mfaChallenges      map[string]transientMFAChallengeRecord
	externalAuthStates map[string]transientExternalAuthStateRecord
	mfaVerifyAttempts  map[string]transientMFAVerificationAttemptRecord
}{
	authorizationCodes: map[string]transientAuthorizationCodeRecord{},
	mfaChallenges:      map[string]transientMFAChallengeRecord{},
	externalAuthStates: map[string]transientExternalAuthStateRecord{},
	mfaVerifyAttempts:  map[string]transientMFAVerificationAttemptRecord{},
}

func storeAuthorizationCode(record model.AuthorizationCode) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	transientStore.authorizationCodes[record.Code] = transientAuthorizationCodeRecord{AuthorizationCode: record}
}

func loadAuthorizationCode(code string) (model.AuthorizationCode, bool) {
	transientStore.mu.RLock()
	record, ok := transientStore.authorizationCodes[code]
	transientStore.mu.RUnlock()
	if !ok {
		return model.AuthorizationCode{}, false
	}
	now := time.Now()
	if record.ExpiresAt.Before(now) || record.ConsumedAt != nil {
		deleteAuthorizationCode(code)
		return model.AuthorizationCode{}, false
	}
	return record.AuthorizationCode, true
}

func consumeAuthorizationCode(code string, now time.Time) (model.AuthorizationCode, bool) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	record, ok := transientStore.authorizationCodes[code]
	if !ok {
		return model.AuthorizationCode{}, false
	}
	if record.ExpiresAt.Before(now) || record.ConsumedAt != nil {
		delete(transientStore.authorizationCodes, code)
		return model.AuthorizationCode{}, false
	}
	record.ConsumedAt = &now
	transientStore.authorizationCodes[code] = record
	return record.AuthorizationCode, true
}

func deleteAuthorizationCode(code string) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	delete(transientStore.authorizationCodes, code)
}

func deleteAuthorizationCodesByUser(userID string) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	for code, record := range transientStore.authorizationCodes {
		if record.UserID == userID {
			delete(transientStore.authorizationCodes, code)
		}
	}
}

func DeleteAuthorizationCodesByUser(userID string) {
	deleteAuthorizationCodesByUser(userID)
}

func storeMFAChallenge(record model.MFAChallenge) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	transientStore.mfaChallenges[record.ID] = transientMFAChallengeRecord{MFAChallenge: record}
}

func latestActiveMFAChallenge(sessionID, method string) (model.MFAChallenge, bool) {
	transientStore.mu.RLock()
	defer transientStore.mu.RUnlock()
	var latest model.MFAChallenge
	now := time.Now()
	found := false
	for _, record := range transientStore.mfaChallenges {
		challenge := record.MFAChallenge
		if challenge.SessionID != sessionID || challenge.Method != method || challenge.ConsumedAt != nil || challenge.ExpiresAt.Before(now) {
			continue
		}
		if !found || challenge.CreatedAt.After(latest.CreatedAt) {
			latest = challenge
			found = true
		}
	}
	return latest, found
}

func updateMFAChallenge(record model.MFAChallenge) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	transientStore.mfaChallenges[record.ID] = transientMFAChallengeRecord{MFAChallenge: record}
}

func deleteMFAChallengesByUser(userID string) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	for id, record := range transientStore.mfaChallenges {
		if record.UserID == userID {
			delete(transientStore.mfaChallenges, id)
		}
	}
}

func DeleteMFAChallengesByUser(userID string) {
	deleteMFAChallengesByUser(userID)
}

func storeExternalAuthState(record model.ExternalAuthState) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	transientStore.externalAuthStates[record.State] = transientExternalAuthStateRecord{ExternalAuthState: record}
}

func loadExternalAuthState(state string) (model.ExternalAuthState, bool) {
	transientStore.mu.RLock()
	record, ok := transientStore.externalAuthStates[state]
	transientStore.mu.RUnlock()
	if !ok {
		return model.ExternalAuthState{}, false
	}
	if record.ExpiresAt.Before(time.Now()) {
		deleteExternalAuthState(state)
		return model.ExternalAuthState{}, false
	}
	return record.ExternalAuthState, true
}

func deleteExternalAuthState(state string) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	delete(transientStore.externalAuthStates, state)
}

func incrementMFAVerificationAttempt(sessionID, method string, expiresAt time.Time) int {
	key := mfaVerificationAttemptKey(sessionID, method)
	now := time.Now()
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	record := transientStore.mfaVerifyAttempts[key]
	if record.ExpiresAt.Before(now) {
		record = transientMFAVerificationAttemptRecord{}
	}
	record.Count++
	record.ExpiresAt = expiresAt
	transientStore.mfaVerifyAttempts[key] = record
	return record.Count
}

func loadMFAVerificationAttemptCount(sessionID, method string) int {
	key := mfaVerificationAttemptKey(sessionID, method)
	transientStore.mu.RLock()
	record, ok := transientStore.mfaVerifyAttempts[key]
	transientStore.mu.RUnlock()
	if !ok {
		return 0
	}
	if record.ExpiresAt.Before(time.Now()) {
		clearMFAVerificationAttempts(sessionID, method)
		return 0
	}
	return record.Count
}

func clearMFAVerificationAttempts(sessionID, method string) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	delete(transientStore.mfaVerifyAttempts, mfaVerificationAttemptKey(sessionID, method))
}

func mfaVerificationAttemptKey(sessionID, method string) string {
	return sessionID + ":" + method
}

func init() {
	go runTransientStoreCleanupLoop()
}

func runTransientStoreCleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		cleanupExpiredTransientState()
	}
}

func cleanupExpiredTransientState() {
	now := time.Now()
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	for code, record := range transientStore.authorizationCodes {
		if record.ExpiresAt.Before(now) || record.ConsumedAt != nil {
			delete(transientStore.authorizationCodes, code)
		}
	}
	for id, record := range transientStore.mfaChallenges {
		if record.ExpiresAt.Before(now) || record.ConsumedAt != nil {
			delete(transientStore.mfaChallenges, id)
		}
	}
	for state, record := range transientStore.externalAuthStates {
		if record.ExpiresAt.Before(now) {
			delete(transientStore.externalAuthStates, state)
		}
	}
	for key, record := range transientStore.mfaVerifyAttempts {
		if record.ExpiresAt.Before(now) {
			delete(transientStore.mfaVerifyAttempts, key)
		}
	}
}
