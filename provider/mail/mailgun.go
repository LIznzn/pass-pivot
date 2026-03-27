package mail

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type MailgunMailProvider struct {
	from    string
	domain  string
	apiKey  string
	apiBase string
	client  *http.Client
}

func NewMailgunMailProvider(cfg Config) *MailgunMailProvider {
	apiBase := strings.TrimRight(strings.TrimSpace(cfg.MailgunAPIBase), "/")
	if apiBase == "" {
		apiBase = "https://api.mailgun.net"
	}

	return &MailgunMailProvider{
		from:    strings.TrimSpace(cfg.From),
		domain:  strings.TrimSpace(cfg.MailgunDomain),
		apiKey:  strings.TrimSpace(cfg.MailgunAPIKey),
		apiBase: apiBase,
		client:  newHTTPClient(),
	}
}

func (provider *MailgunMailProvider) SendMail(message Message) error {
	form := url.Values{}
	form.Set("from", provider.from)
	form.Set("to", message.To)
	form.Set("subject", message.Subject)
	if strings.TrimSpace(message.TextBody) != "" {
		form.Set("text", message.TextBody)
	}
	if strings.TrimSpace(message.HTMLBody) != "" {
		form.Set("html", message.HTMLBody)
	}

	endpoint := fmt.Sprintf("%s/v3/%s/messages", provider.apiBase, url.PathEscape(strings.TrimSpace(provider.domain)))
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("api:"+provider.apiKey)))

	resp, err := provider.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("mailgun send failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
}
