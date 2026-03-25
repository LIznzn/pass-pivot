package captcha

import "fmt"

type Provider interface {
	VerifyCaptcha(token, clientId, clientSecret string) (bool, error)
}

func GetCaptchaProvider(captchaType string) Provider {
	switch captchaType {
	default:
		return nil
	case "Default":
		return NewDefaultCaptchaProvider()
	case "Google reCAPTCHA":
		return NewGoogleCaptchaProvider()
	case "Cloudflare Turnstile":
		return NewCloudflareCaptchaProvider()
	}
}

func VerifyCaptchaByCaptchaType(captchaType, token, clientId, clientSecret string) (bool, error) {
	provider := GetCaptchaProvider(captchaType)
	if provider == nil {
		return false, fmt.Errorf("invalid captcha provider: %s", captchaType)
	}

	return provider.VerifyCaptcha(token, clientId, clientSecret)
}
