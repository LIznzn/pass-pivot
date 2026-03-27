package mail

import (
	"fmt"
	"net/smtp"
	"strings"
)

type SMTPMailProvider struct {
	from string
	addr string
	auth smtp.Auth
}

func NewSMTPMailProvider(cfg Config) *SMTPMailProvider {
	addr := fmt.Sprintf("%s:%d", strings.TrimSpace(cfg.SMTPHost), cfg.SMTPPort)
	var auth smtp.Auth
	if strings.TrimSpace(cfg.SMTPUser) != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, strings.TrimSpace(cfg.SMTPHost))
	}

	return &SMTPMailProvider{
		from: strings.TrimSpace(cfg.From),
		addr: addr,
		auth: auth,
	}
}

func (provider *SMTPMailProvider) SendMail(message Message) error {
	var builder strings.Builder
	builder.WriteString("From: " + provider.from + "\r\n")
	builder.WriteString("To: " + message.To + "\r\n")
	builder.WriteString("Subject: " + message.Subject + "\r\n")
	builder.WriteString("MIME-Version: 1.0\r\n")
	if strings.TrimSpace(message.HTMLBody) != "" && strings.TrimSpace(message.TextBody) != "" {
		boundary := "ppvt-mail-boundary"
		builder.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n\r\n")
		builder.WriteString("--" + boundary + "\r\n")
		builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		builder.WriteString(message.TextBody + "\r\n")
		builder.WriteString("--" + boundary + "\r\n")
		builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		builder.WriteString(message.HTMLBody + "\r\n")
		builder.WriteString("--" + boundary + "--")
	} else if strings.TrimSpace(message.HTMLBody) != "" {
		builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		builder.WriteString(message.HTMLBody)
	} else {
		builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		builder.WriteString(message.TextBody)
	}
	return smtp.SendMail(provider.addr, provider.auth, provider.from, []string{message.To}, []byte(builder.String()))
}
