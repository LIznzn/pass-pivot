package captcha

type GoogleCaptchaProvider struct{}

func NewGoogleCaptchaProvider() *DefaultCaptchaProvider {
	captcha := &DefaultCaptchaProvider{}
	return captcha
}

func (captcha *GoogleCaptchaProvider) VerifyCaptcha(token, clientId, clientSecret, clientId2 string) (bool, error) {
	return true, nil
}
