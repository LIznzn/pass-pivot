package authn

import "pass-pivot/internal/model"

type LoginInput struct {
	OrganizationID      string
	ApplicationID       string
	Identifier          string
	Secret              string
	CaptchaProvider     string
	CaptchaToken        string
	IPAddress           string
	UserAgent           string
	DeviceKey           string
	TrustCurrentDevice  bool
	RequireAnnouncement bool
}

type LoginResult struct {
	Session             model.Session `json:"session"`
	NextStep            string        `json:"nextStep"`
	RecoveryCodesIssued []string      `json:"recoveryCodesIssued,omitempty"`
	Tokens              []model.Token `json:"tokens,omitempty"`
	Fingerprint         string        `json:"-"`
}
