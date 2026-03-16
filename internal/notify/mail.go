package notify

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

type Mailer interface {
	Send(ctx context.Context, to string, subject string, body string) error
}

type NoopMailer struct{}

func (NoopMailer) Send(ctx context.Context, to string, subject string, body string) error {
	return nil
}

type SMTPMailer struct {
	from string
	addr string
	auth smtp.Auth
}

type SMTPConfig struct {
	From     string
	Host     string
	Port     int
	Username string
	Password string
}

func NewMailer(cfg SMTPConfig) Mailer {
	if cfg.From == "" || cfg.Host == "" || cfg.Port <= 0 {
		return NoopMailer{}
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	return &SMTPMailer{
		from: cfg.From,
		addr: addr,
		auth: auth,
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
