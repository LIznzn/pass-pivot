package captcha

type DefaultCaptchaProvider struct{}

func NewDefaultCaptchaProvider() *DefaultCaptchaProvider {
	captcha := &DefaultCaptchaProvider{}
	return captcha
}

func (captcha *DefaultCaptchaProvider) VerifyCaptcha(token, clientId, clientSecret, clientId2 string) (bool, error) {
	return true, nil
}
