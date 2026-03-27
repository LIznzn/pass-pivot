package mail

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const httpRequestTimeout = 15 * time.Second

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: httpRequestTimeout}
}

type Config struct {
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

type Message struct {
	To       string
	Subject  string
	TextBody string
	HTMLBody string
}

type Provider interface {
	SendMail(message Message) error
}

func GetMailProvider(mailType string, cfg Config) Provider {
	switch strings.ToLower(strings.TrimSpace(mailType)) {
	default:
		return nil
	case "smtp":
		return NewSMTPMailProvider(cfg)
	case "mailgun":
		return NewMailgunMailProvider(cfg)
	case "sendgrid":
		return NewSendGridMailProvider(cfg)
	}
}

func SendMailByMailType(mailType string, cfg Config, message Message) error {
	provider := GetMailProvider(mailType, cfg)
	if provider == nil {
		return fmt.Errorf("invalid mail provider: %s", mailType)
	}

	return provider.SendMail(message)
}
