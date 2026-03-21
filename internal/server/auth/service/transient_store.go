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

var transientStore = struct {
	mu                 sync.RWMutex
	authorizationCodes map[string]transientAuthorizationCodeRecord
	mfaChallenges      map[string]transientMFAChallengeRecord
	externalAuthStates map[string]transientExternalAuthStateRecord
}{
	authorizationCodes: map[string]transientAuthorizationCodeRecord{},
	mfaChallenges:      map[string]transientMFAChallengeRecord{},
	externalAuthStates: map[string]transientExternalAuthStateRecord{},
}

func storeAuthorizationCode(record model.AuthorizationCode) {
	cleanupExpiredTransientState()
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	transientStore.authorizationCodes[record.Code] = transientAuthorizationCodeRecord{AuthorizationCode: record}
}

func loadAuthorizationCode(code string) (model.AuthorizationCode, bool) {
	cleanupExpiredTransientState()
	transientStore.mu.RLock()
	defer transientStore.mu.RUnlock()
	record, ok := transientStore.authorizationCodes[code]
	return record.AuthorizationCode, ok
}

func consumeAuthorizationCode(code string, now time.Time) (model.AuthorizationCode, bool) {
	cleanupExpiredTransientState()
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
	cleanupExpiredTransientState()
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	transientStore.mfaChallenges[record.ID] = transientMFAChallengeRecord{MFAChallenge: record}
}

func latestActiveMFAChallenge(sessionID, method string) (model.MFAChallenge, bool) {
	cleanupExpiredTransientState()
	transientStore.mu.RLock()
	defer transientStore.mu.RUnlock()
	var latest model.MFAChallenge
	found := false
	for _, record := range transientStore.mfaChallenges {
		challenge := record.MFAChallenge
		if challenge.SessionID != sessionID || challenge.Method != method || challenge.ConsumedAt != nil {
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
	cleanupExpiredTransientState()
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	transientStore.externalAuthStates[record.State] = transientExternalAuthStateRecord{ExternalAuthState: record}
}

func loadExternalAuthState(state string) (model.ExternalAuthState, bool) {
	cleanupExpiredTransientState()
	transientStore.mu.RLock()
	defer transientStore.mu.RUnlock()
	record, ok := transientStore.externalAuthStates[state]
	return record.ExternalAuthState, ok
}

func deleteExternalAuthState(state string) {
	transientStore.mu.Lock()
	defer transientStore.mu.Unlock()
	delete(transientStore.externalAuthStates, state)
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
}
