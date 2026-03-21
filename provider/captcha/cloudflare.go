package captcha

type CloudflareCaptchaProvider struct{}

func NewCloudflareCaptchaProvider() *DefaultCaptchaProvider {
	captcha := &DefaultCaptchaProvider{}
	return captcha
}

func (captcha *CloudflareCaptchaProvider) VerifyCaptcha(token, clientId, clientSecret, clientId2 string) (bool, error) {
	return true, nil
}
