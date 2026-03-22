package captcha

import (
	"time"

	coreservice "pass-pivot/internal/server/core/service"
)

type DefaultCaptchaProvider struct {
	now func() time.Time
}

func NewDefaultCaptchaProvider() *DefaultCaptchaProvider {
	return &DefaultCaptchaProvider{now: time.Now}
}

func (captcha *DefaultCaptchaProvider) VerifyCaptcha(token, clientId, clientSecret string) (bool, error) {
	return coreservice.VerifyDefaultCaptcha(clientSecret, token, captcha.now())
}
