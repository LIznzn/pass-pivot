package captcha

import "fmt"

type CaptchaProvider interface {
	VerifyCaptcha(token, clientId, clientSecret, clientId2 string) (bool, error)
}

func GetCaptchaProvider(captchaType string) CaptchaProvider {
	switch captchaType {
	default:
		return nil
	case "Default":
		return NewDefaultCaptchaProvider()
	case "Google reCAPTCHA":
		return NewGoogleCaptchaProvider()
	case "Cloudflare Turnstile":
		return NewCloudflareCaptchaProvider()
	case "GeeTest CAPTCHA":
		return NewGeetestCaptchaProvider()
	}
}

func VerifyCaptchaByCaptchaType(captchaType, token, clientId, clientSecret, clientId2 string) (bool, error) {
	provider := GetCaptchaProvider(captchaType)
	if provider == nil {
		return false, fmt.Errorf("invalid captcha provider: %s", captchaType)
	}

	return provider.VerifyCaptcha(token, clientId, clientSecret, clientId2)
}
