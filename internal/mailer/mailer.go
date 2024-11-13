package mailer

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/mailgun/raymond/v2"
	"github.com/watzon/0x45/internal/config"
)

type Mailer struct {
	config *config.Config
	auth   smtp.Auth
}

func New(cfg *config.Config) (*Mailer, error) {
	// Create SMTP auth only if credentials are provided
	var auth smtp.Auth
	if cfg.SMTP.Username != "" && cfg.SMTP.Password != "" {
		auth = smtp.PlainAuth(
			"",
			cfg.SMTP.Username,
			cfg.SMTP.Password,
			cfg.SMTP.Host,
		)
	}

	return &Mailer{
		config: cfg,
		auth:   auth,
	}, nil
}

func (m *Mailer) SendVerification(to, token string) error {
	// Read the template file
	tpl, err := raymond.ParseFile("views/emails/verify_api_key.hbs")
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	// Render the template with data
	body, err := tpl.Exec(map[string]any{
		"baseUrl": m.config.Server.BaseURL,
		"token":   token,
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", m.config.SMTP.Host, m.config.SMTP.Port)

	// Configure TLS
	tlsConfig := &tls.Config{
		ServerName:         m.config.SMTP.Host,
		InsecureSkipVerify: !m.config.SMTP.StartTLS,
	}

	// Connect to the server
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer c.Close()

	// Start TLS if enabled
	if m.config.SMTP.StartTLS {
		if err = c.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate only if auth is configured
	if m.auth != nil {
		if err = c.Auth(m.auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	// Set the sender and recipient
	if err = c.Mail(m.config.SMTP.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	if err = c.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send the email body
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("failed to create data writer: %w", err)
	}
	defer w.Close()

	msg := fmt.Sprintf("To: %s\r\n"+
		"From: %s <%s>\r\n"+
		"Subject: Verify your Paste69 API Key\r\n"+
		"MIME-version: 1.0;\r\n"+
		"Content-Type: text/html; charset=\"UTF-8\";\r\n"+
		"\r\n"+
		"%s", to, m.config.SMTP.FromName, m.config.SMTP.From, body)

	if _, err = w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	return nil
}
