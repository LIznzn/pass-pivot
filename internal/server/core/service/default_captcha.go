package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"pass-pivot/util"
)

const (
	DefaultCaptchaTTL          = 5 * time.Minute
	defaultCaptchaTokenPart    = 2
	defaultCaptchaLength       = 5
	defaultCaptchaWidth        = 180
	defaultCaptchaHeight       = 64
	defaultCaptchaCharset      = "234567ACDEFGHJKLMNPQRTUVWXYZ"
	defaultCaptchaNoiseLines   = 10
	defaultCaptchaNoiseCircles = 18
)

type DefaultCaptchaChallenge struct {
	ImageDataURL   string    `json:"imageDataUrl"`
	ChallengeToken string    `json:"challengeToken"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

type defaultCaptchaPayload struct {
	Answer    string `json:"answer"`
	Nonce     string `json:"nonce"`
	ExpiresAt int64  `json:"expiresAt"`
}

var defaultCaptchaNonceStore = struct {
	mu   sync.Mutex
	used map[string]int64
}{
	used: map[string]int64{},
}

func CreateDefaultCaptcha(secret string, now time.Time) (*DefaultCaptchaChallenge, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("default captcha secret is empty")
	}
	answer, err := randomDefaultCaptchaAnswer(defaultCaptchaLength)
	if err != nil {
		return nil, err
	}
	imageDataURL, err := buildDefaultCaptchaImageDataURL(answer)
	if err != nil {
		return nil, err
	}
	nonce, err := util.RandomToken(12)
	if err != nil {
		return nil, err
	}
	expiresAt := now.Add(DefaultCaptchaTTL)
	challengeToken, err := signDefaultCaptchaPayload(secret, defaultCaptchaPayload{
		Answer:    answer,
		Nonce:     nonce,
		ExpiresAt: expiresAt.Unix(),
	})
	if err != nil {
		return nil, err
	}
	return &DefaultCaptchaChallenge{
		ImageDataURL:   imageDataURL,
		ChallengeToken: challengeToken,
		ExpiresAt:      expiresAt,
	}, nil
}

func BuildDefaultCaptchaResponseToken(challengeToken, answer string) string {
	return strings.TrimSpace(challengeToken) + "." + base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(answer)))
}

func VerifyDefaultCaptcha(secret, token string, now time.Time) (bool, error) {
	token = strings.TrimSpace(token)
	secret = strings.TrimSpace(secret)
	if token == "" {
		return false, fmt.Errorf("default captcha token is empty")
	}
	if secret == "" {
		return false, fmt.Errorf("default captcha secret is empty")
	}
	challengeToken, answer, err := parseDefaultCaptchaResponseToken(token)
	if err != nil {
		return false, err
	}
	payload, err := verifyDefaultCaptchaPayload(secret, challengeToken)
	if err != nil {
		return false, err
	}
	if payload.ExpiresAt < now.Unix() {
		return false, fmt.Errorf("default captcha expired")
	}
	if !hmac.Equal([]byte(strings.ToUpper(strings.TrimSpace(payload.Answer))), []byte(normalizeDefaultCaptchaAnswer(answer))) {
		return false, nil
	}
	if !consumeDefaultCaptchaNonce(payload.Nonce, payload.ExpiresAt, now.Unix()) {
		return false, fmt.Errorf("default captcha has already been used")
	}
	return true, nil
}

func consumeDefaultCaptchaNonce(nonce string, expiresAtUnix, nowUnix int64) bool {
	defaultCaptchaNonceStore.mu.Lock()
	defer defaultCaptchaNonceStore.mu.Unlock()
	for key, expiry := range defaultCaptchaNonceStore.used {
		if expiry <= nowUnix {
			delete(defaultCaptchaNonceStore.used, key)
		}
	}
	nonce = strings.TrimSpace(nonce)
	if nonce == "" {
		return false
	}
	if existingExpiry, exists := defaultCaptchaNonceStore.used[nonce]; exists && existingExpiry > nowUnix {
		return false
	}
	defaultCaptchaNonceStore.used[nonce] = expiresAtUnix
	return true
}

func parseDefaultCaptchaResponseToken(token string) (string, string, error) {
	idx := strings.LastIndex(token, ".")
	if idx <= 0 || idx >= len(token)-1 {
		return "", "", fmt.Errorf("invalid default captcha token")
	}
	challengeToken := strings.TrimSpace(token[:idx])
	answerBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token[idx+1:]))
	if err != nil {
		return "", "", fmt.Errorf("invalid default captcha answer encoding")
	}
	answer := strings.TrimSpace(string(answerBytes))
	if answer == "" {
		return "", "", fmt.Errorf("default captcha answer is empty")
	}
	return challengeToken, answer, nil
}

func signDefaultCaptchaPayload(secret string, payload defaultCaptchaPayload) (string, error) {
	encodedPayload, err := encodeDefaultCaptchaPayload(payload)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(encodedPayload)); err != nil {
		return "", err
	}
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encodedPayload + "." + signature, nil
}

func verifyDefaultCaptchaPayload(secret, token string) (*defaultCaptchaPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != defaultCaptchaTokenPart {
		return nil, fmt.Errorf("invalid default captcha challenge token")
	}
	encodedPayload := strings.TrimSpace(parts[0])
	signature := strings.TrimSpace(parts[1])
	if encodedPayload == "" || signature == "" {
		return nil, fmt.Errorf("invalid default captcha challenge token")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(encodedPayload)); err != nil {
		return nil, err
	}
	expectedSignature := mac.Sum(nil)
	actualSignature, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid default captcha challenge signature")
	}
	if !hmac.Equal(actualSignature, expectedSignature) {
		return nil, fmt.Errorf("invalid default captcha challenge signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("invalid default captcha payload")
	}
	var payload defaultCaptchaPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("invalid default captcha payload")
	}
	if strings.TrimSpace(payload.Answer) == "" || payload.ExpiresAt <= 0 || strings.TrimSpace(payload.Nonce) == "" {
		return nil, fmt.Errorf("invalid default captcha payload")
	}
	return &payload, nil
}

func encodeDefaultCaptchaPayload(payload defaultCaptchaPayload) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

func normalizeDefaultCaptchaAnswer(answer string) string {
	return strings.ToUpper(strings.TrimSpace(answer))
}

func randomDefaultCaptchaAnswer(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("default captcha length must be positive")
	}
	token, err := util.RandomToken(length + 8)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(token))
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		builder.WriteByte(defaultCaptchaCharset[int(sum[i])%len(defaultCaptchaCharset)])
	}
	return builder.String(), nil
}

func buildDefaultCaptchaImageDataURL(answer string) (string, error) {
	if strings.TrimSpace(answer) == "" {
		return "", fmt.Errorf("default captcha answer is empty")
	}
	seed := sha256.Sum256([]byte(answer))
	svg := buildDefaultCaptchaSVG(answer, seed)
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg)), nil
}

func buildDefaultCaptchaSVG(answer string, seed [32]byte) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="captcha">`, defaultCaptchaWidth, defaultCaptchaHeight, defaultCaptchaWidth, defaultCaptchaHeight))
	builder.WriteString(`<defs><filter id="warp" x="-20%" y="-20%" width="140%" height="140%"><feTurbulence type="fractalNoise" baseFrequency="0.024 0.08" numOctaves="2" seed="7" result="noise"/><feDisplacementMap in="SourceGraphic" in2="noise" scale="8" xChannelSelector="R" yChannelSelector="B"/></filter><filter id="blur"><feGaussianBlur stdDeviation="0.35"/></filter></defs>`)
	builder.WriteString(`<rect width="100%" height="100%" rx="10" ry="10" fill="#f7f4ea"/>`)
	for i := 0; i < defaultCaptchaNoiseLines; i++ {
		x1 := int(seed[(i*4)%len(seed)]) % defaultCaptchaWidth
		y1 := int(seed[(i*4+1)%len(seed)]) % defaultCaptchaHeight
		x2 := int(seed[(i*4+2)%len(seed)]) % defaultCaptchaWidth
		y2 := int(seed[(i*4+3)%len(seed)]) % defaultCaptchaHeight
		color := defaultCaptchaPalette(i + 1)
		builder.WriteString(fmt.Sprintf(`<path d="M%d %d Q %d %d %d %d" stroke="%s" stroke-width="1.4" fill="none" opacity="0.55"/>`, x1, y1, (x1+x2)/2, (y1+y2)/2+int(seed[(i+11)%len(seed)])%18-9, x2, y2, color))
	}
	for i := 0; i < defaultCaptchaNoiseCircles; i++ {
		x := int(seed[(i+7)%len(seed)]) % defaultCaptchaWidth
		y := int(seed[(i+19)%len(seed)]) % defaultCaptchaHeight
		r := int(seed[(i+3)%len(seed)]%4) + 1
		builder.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" fill="%s" opacity="0.35"/>`, x, y, r, defaultCaptchaPalette(i+3)))
	}
	for i := 0; i < 8; i++ {
		x := int(seed[(i+8)%len(seed)]) % defaultCaptchaWidth
		y := int(seed[(i+16)%len(seed)]) % defaultCaptchaHeight
		w := 10 + int(seed[(i+20)%len(seed)]%20)
		h := 2 + int(seed[(i+24)%len(seed)]%5)
		builder.WriteString(fmt.Sprintf(`<ellipse cx="%d" cy="%d" rx="%d" ry="%d" fill="%s" opacity="0.16" filter="url(#blur)"/>`, x, y, w, h, defaultCaptchaPalette(i+1)))
	}
	builder.WriteString(`<g filter="url(#warp)">`)
	for i, ch := range answer {
		x := 18 + i*29 + int(seed[(i+5)%len(seed)]%8) - 2
		y := 41 + int(seed[(i+9)%len(seed)]%10) - 5
		rotate := int(seed[(i+13)%len(seed)]%51) - 25
		skew := int(seed[(i+17)%len(seed)]%21) - 10
		fontSize := 27 + int(seed[(i+21)%len(seed)]%10)
		color := defaultCaptchaPalette(i)
		if seed[(i+25)%len(seed)]%3 == 0 {
			builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Georgia, Times New Roman, serif" font-size="%d" font-weight="700" letter-spacing="-1" fill="none" stroke="%s" stroke-width="1.8" stroke-linejoin="round" transform="rotate(%d %d %d) skewX(%d)">%c</text>`, x, y, fontSize, color, rotate, x, y, skew, ch))
			continue
		}
		builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Georgia, Times New Roman, serif" font-size="%d" font-weight="700" letter-spacing="-1" fill="%s" transform="rotate(%d %d %d) skewX(%d)">%c</text>`, x, y, fontSize, color, rotate, x, y, skew, ch))
	}
	builder.WriteString(`</g>`)
	for i := 0; i < 6; i++ {
		x := 10 + int(seed[(i+23)%len(seed)])%(defaultCaptchaWidth-20)
		y := 8 + int(seed[(i+27)%len(seed)])%(defaultCaptchaHeight-16)
		w := 18 + int(seed[(i+29)%len(seed)]%18)
		h := 4 + int(seed[(i+31)%len(seed)]%8)
		builder.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="2" ry="2" fill="%s" opacity="0.28" transform="rotate(%d %d %d)"/>`, x, y, w, h, defaultCaptchaPalette(i+2), int(seed[(i+1)%len(seed)]%50)-25, x, y))
	}
	for i := 0; i < 5; i++ {
		x1 := int(seed[(i+2)%len(seed)]) % defaultCaptchaWidth
		y1 := int(seed[(i+6)%len(seed)]) % defaultCaptchaHeight
		x2 := int(seed[(i+10)%len(seed)]) % defaultCaptchaWidth
		y2 := int(seed[(i+14)%len(seed)]) % defaultCaptchaHeight
		builder.WriteString(fmt.Sprintf(`<path d="M%d %d C %d %d %d %d %d %d" stroke="%s" stroke-width="2.8" fill="none" opacity="0.48"/>`, x1, y1, x1+12, y1-14, x2-12, y2+14, x2, y2, defaultCaptchaPalette(i+4)))
	}
	for i := 0; i < 3; i++ {
		x := 12 + int(seed[(i+4)%len(seed)])%(defaultCaptchaWidth-24)
		y := 14 + int(seed[(i+12)%len(seed)])%(defaultCaptchaHeight-28)
		builder.WriteString(fmt.Sprintf(`<path d="M%d %d l10 -8 l9 10 l-11 7 z" fill="%s" opacity="0.18"/>`, x, y, defaultCaptchaPalette(i+1)))
	}
	builder.WriteString(`</svg>`)
	return builder.String()
}

func defaultCaptchaPalette(index int) string {
	colors := []string{"#264653", "#1d3557", "#6d597a", "#7f5539", "#8a5a44", "#3a5a40"}
	return colors[index%len(colors)]
}
