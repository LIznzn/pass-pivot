package notify

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
)

type Mailer interface {
	Send(ctx context.Context, to string, subject string, body string) error
}

type NoopMailer struct{}

func (NoopMailer) Send(ctx context.Context, to string, subject string, body string) error {
	return nil
}

type MailConfig struct {
	Provider       string
	From           string
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPass       string
	MailgunDomain  string
	MailgunAPIKey  string
	MailgunAPIBase string
	SendGridAPIKey string
}

type SMTPMailer struct {
	from string
	addr string
	auth smtp.Auth
}

type MailgunMailer struct {
	from    string
	domain  string
	apiKey  string
	apiBase string
	client  *http.Client
}

type SendGridMailer struct {
	from   string
	apiKey string
	client *http.Client
}

func NewMailer(cfg MailConfig) Mailer {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "smtp":
		if cfg.From == "" || cfg.SMTPHost == "" || cfg.SMTPPort <= 0 {
			return NoopMailer{}
		}
		addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
		var auth smtp.Auth
		if cfg.SMTPUser != "" {
			auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
		}
		return &SMTPMailer{
			from: cfg.From,
			addr: addr,
			auth: auth,
		}
	case "mailgun":
		if cfg.From == "" || cfg.MailgunDomain == "" || cfg.MailgunAPIKey == "" {
			return NoopMailer{}
		}
		apiBase := strings.TrimRight(strings.TrimSpace(cfg.MailgunAPIBase), "/")
		if apiBase == "" {
			apiBase = "https://api.mailgun.net"
		}
		return &MailgunMailer{
			from:    cfg.From,
			domain:  cfg.MailgunDomain,
			apiKey:  cfg.MailgunAPIKey,
			apiBase: apiBase,
			client:  http.DefaultClient,
		}
	case "sendgrid":
		if cfg.From == "" || cfg.SendGridAPIKey == "" {
			return NoopMailer{}
		}
		return &SendGridMailer{
			from:   cfg.From,
			apiKey: cfg.SendGridAPIKey,
			client: http.DefaultClient,
		}
	default:
		return NoopMailer{}
	}
}

func (m *SMTPMailer) Send(ctx context.Context, to string, subject string, body string) error {
	var builder strings.Builder
	builder.WriteString("From: " + m.from + "\r\n")
	builder.WriteString("To: " + to + "\r\n")
	builder.WriteString("Subject: " + subject + "\r\n")
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	builder.WriteString(body)
	return smtp.SendMail(m.addr, m.auth, m.from, []string{to}, []byte(builder.String()))
}

func (m *MailgunMailer) Send(ctx context.Context, to string, subject string, body string) error {
	form := url.Values{}
	form.Set("from", m.from)
	form.Set("to", to)
	form.Set("subject", subject)
	form.Set("text", body)
	endpoint := fmt.Sprintf("%s/v3/%s/messages", m.apiBase, url.PathEscape(strings.TrimSpace(m.domain)))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("api:"+m.apiKey)))
	resp, err := m.client.Do(req)
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

func (m *SendGridMailer) Send(ctx context.Context, to string, subject string, body string) error {
	payload := map[string]any{
		"personalizations": []map[string]any{
			{
				"to": []map[string]string{
					{"email": to},
				},
			},
		},
		"from": map[string]string{
			"email": m.from,
		},
		"subject": subject,
		"content": []map[string]string{
			{
				"type":  "text/plain",
				"value": body,
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.client.Do(req)
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
