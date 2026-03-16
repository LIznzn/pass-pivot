package captcha

type GeetestCaptchaProvider struct{}

func NewGeetestCaptchaProvider() *DefaultCaptchaProvider {
	captcha := &DefaultCaptchaProvider{}
	return captcha
}

func (captcha *GeetestCaptchaProvider) VerifyCaptcha(token, clientId, clientSecret, clientId2 string) (bool, error) {
	return true, nil
}
