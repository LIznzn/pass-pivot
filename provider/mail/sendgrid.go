package mail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type SendGridMailProvider struct {
	from   string
	apiKey string
	client *http.Client
}

func NewSendGridMailProvider(cfg Config) *SendGridMailProvider {
	return &SendGridMailProvider{
		from:   strings.TrimSpace(cfg.From),
		apiKey: strings.TrimSpace(cfg.SendGridAPIKey),
		client: newHTTPClient(),
	}
}

func (provider *SendGridMailProvider) SendMail(message Message) error {
	content := make([]map[string]string, 0, 2)
	if strings.TrimSpace(message.TextBody) != "" {
		content = append(content, map[string]string{
			"type":  "text/plain",
			"value": message.TextBody,
		})
	}
	if strings.TrimSpace(message.HTMLBody) != "" {
		content = append(content, map[string]string{
			"type":  "text/html",
			"value": message.HTMLBody,
		})
	}

	payload := map[string]any{
		"personalizations": []map[string]any{
			{
				"to": []map[string]string{
					{"email": message.To},
				},
			},
		},
		"from": map[string]string{
			"email": provider.from,
		},
		"subject": message.Subject,
		"content": content,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+provider.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := provider.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("sendgrid send failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
}
