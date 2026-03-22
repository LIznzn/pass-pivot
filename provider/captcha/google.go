package captcha

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GoogleCaptchaProvider struct{}

type googleVerifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

func NewGoogleCaptchaProvider() *GoogleCaptchaProvider {
	captcha := &GoogleCaptchaProvider{}
	return captcha
}

func (captcha *GoogleCaptchaProvider) VerifyCaptcha(token, clientId, clientSecret string) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, fmt.Errorf("google captcha token is empty")
	}
	if strings.TrimSpace(clientSecret) == "" {
		return false, fmt.Errorf("google captcha secret is empty")
	}

	form := url.Values{}
	form.Set("secret", clientSecret)
	form.Set("response", token)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).PostForm("https://www.google.com/recaptcha/api/siteverify", form)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("google captcha verify failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result googleVerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	return result.Success, nil
}
