package captcha

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GeetestCaptchaProvider struct{}

type geetestVerifyToken struct {
	LotNumber     string `json:"lot_number"`
	CaptchaOutput string `json:"captcha_output"`
	PassToken     string `json:"pass_token"`
	GenTime       string `json:"gen_time"`
}

type geetestVerifyResponse struct {
	Result string `json:"result"`
	Reason string `json:"reason"`
}

func NewGeetestCaptchaProvider() *GeetestCaptchaProvider {
	captcha := &GeetestCaptchaProvider{}
	return captcha
}

func (captcha *GeetestCaptchaProvider) VerifyCaptcha(token, clientId, clientSecret, clientId2 string) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, fmt.Errorf("geetest captcha token is empty")
	}
	if strings.TrimSpace(clientId) == "" {
		return false, fmt.Errorf("geetest captcha id is empty")
	}
	if strings.TrimSpace(clientSecret) == "" {
		return false, fmt.Errorf("geetest captcha key is empty")
	}

	var verifyToken geetestVerifyToken
	if err := json.Unmarshal([]byte(token), &verifyToken); err != nil {
		values, parseErr := url.ParseQuery(token)
		if parseErr != nil {
			return false, fmt.Errorf("invalid geetest captcha token: %w", err)
		}
		verifyToken = geetestVerifyToken{
			LotNumber:     values.Get("lot_number"),
			CaptchaOutput: values.Get("captcha_output"),
			PassToken:     values.Get("pass_token"),
			GenTime:       values.Get("gen_time"),
		}
	}

	if strings.TrimSpace(verifyToken.LotNumber) == "" ||
		strings.TrimSpace(verifyToken.CaptchaOutput) == "" ||
		strings.TrimSpace(verifyToken.PassToken) == "" ||
		strings.TrimSpace(verifyToken.GenTime) == "" {
		return false, fmt.Errorf("invalid geetest captcha token payload")
	}

	mac := hmac.New(sha256.New, []byte(clientSecret))
	if _, err := mac.Write([]byte(verifyToken.LotNumber)); err != nil {
		return false, err
	}
	signToken := hex.EncodeToString(mac.Sum(nil))

	form := url.Values{}
	form.Set("lot_number", verifyToken.LotNumber)
	form.Set("captcha_output", verifyToken.CaptchaOutput)
	form.Set("pass_token", verifyToken.PassToken)
	form.Set("gen_time", verifyToken.GenTime)
	form.Set("sign_token", signToken)

	verifyURL := fmt.Sprintf("https://gcaptcha4.geetest.com/validate?captcha_id=%s", url.QueryEscape(clientId))
	resp, err := (&http.Client{Timeout: 10 * time.Second}).PostForm(verifyURL, form)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("geetest captcha verify failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result geetestVerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	return result.Result == "success", nil
}
