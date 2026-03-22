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

type CloudflareCaptchaProvider struct{}

type cloudflareVerifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

func NewCloudflareCaptchaProvider() *CloudflareCaptchaProvider {
	captcha := &CloudflareCaptchaProvider{}
	return captcha
}

func (captcha *CloudflareCaptchaProvider) VerifyCaptcha(token, clientId, clientSecret string) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, fmt.Errorf("cloudflare captcha token is empty")
	}
	if strings.TrimSpace(clientSecret) == "" {
		return false, fmt.Errorf("cloudflare captcha secret is empty")
	}

	form := url.Values{}
	form.Set("secret", clientSecret)
	form.Set("response", token)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", form)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("cloudflare captcha verify failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result cloudflareVerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	return result.Success, nil
}
