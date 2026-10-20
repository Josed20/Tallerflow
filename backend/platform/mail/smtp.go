package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SMTPConfig holds configuration for the SMTP mail sender.
type SMTPConfig struct {
	Host        string
	Port        string
	Username    string
	Password    string
	FromAddress string
	RequireTLS  bool
	Timeout     time.Duration
}

// SMTPSender implements Sender using standard SMTP.
type SMTPSender struct {
	config SMTPConfig
}

// NewSMTPSender validates configuration and returns a new SMTPSender.
func NewSMTPSender(cfg SMTPConfig) (*SMTPSender, error) {
	cfg.Host = strings.TrimSpace(cfg.Host)
	if cfg.FromAddress == "" {
		cfg.FromAddress = "soporte@tallerflowpe.com"
	}
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == "" {
		cfg.Port = "1025"
	}
	if _, err := strconv.ParseUint(cfg.Port, 10, 16); err != nil || cfg.Port == "0" {
		return nil, fmt.Errorf("smtp port is invalid")
	}
	from, err := mail.ParseAddress(cfg.FromAddress)
	if err != nil || from.Address != cfg.FromAddress {
		return nil, fmt.Errorf("smtp from address is invalid")
	}
	if (cfg.Username == "") != (cfg.Password == "") {
		return nil, fmt.Errorf("smtp username and password must be configured together")
	}
	if (cfg.Username != "" || cfg.Password != "") && !cfg.RequireTLS {
		return nil, fmt.Errorf("smtp authentication requires TLS")
	}
	if !cfg.RequireTLS && !isLocalDevelopmentSMTPHost(cfg.Host) {
		return nil, fmt.Errorf("SMTP TLS is required for non-local hosts")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &SMTPSender{config: cfg}, nil
}

func isLocalDevelopmentSMTPHost(host string) bool {
	if strings.EqualFold(host, "localhost") || strings.EqualFold(host, "mailpit") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Send dispatches an email message via SMTP.
func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	trimmedTo := strings.TrimSpace(msg.To)
	if trimmedTo == "" {
		return fmt.Errorf("recipient email is required")
	}
	recipient, err := mail.ParseAddress(trimmedTo)
	if err != nil || recipient.Address != trimmedTo {
		return fmt.Errorf("recipient email is invalid")
	}
	trimmedSubject := strings.TrimSpace(msg.Subject)
	if trimmedSubject == "" {
		return fmt.Errorf("email subject is required")
	}
	if strings.ContainsAny(trimmedSubject, "\r\n") {
		return fmt.Errorf("email subject contains invalid characters")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	addr := net.JoinHostPort(s.config.Host, s.config.Port)
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp connection failed: %w", err)
	}
	defer connection.Close()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = connection.Close()
		case <-done:
		}
	}()
	defer close(done)

	client, err := smtp.NewClient(connection, s.config.Host)
	if err != nil {
		return fmt.Errorf("smtp client failed: %w", err)
	}
	defer client.Quit()

	if s.config.RequireTLS {
		supportsTLS, _ := client.Extension("STARTTLS")
		if !supportsTLS {
			return fmt.Errorf("smtp server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.config.Host}); err != nil {
			return fmt.Errorf("smtp STARTTLS failed: %w", err)
		}
	}

	var rawMessage string
	if msg.HTML != "" {
		boundary := fmt.Sprintf("----=_Part_%d", time.Now().UnixNano())
		rawMessage = fmt.Sprintf("From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n"+
			"--%s\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n\r\n"+
			"%s\r\n\r\n"+
			"--%s\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n\r\n"+
			"%s\r\n\r\n"+
			"--%s--",
			s.config.FromAddress, trimmedTo, trimmedSubject, boundary,
			boundary, msg.Text,
			boundary, msg.HTML,
			boundary)
	} else {
		rawMessage = fmt.Sprintf("From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n\r\n"+
			"%s", s.config.FromAddress, trimmedTo, trimmedSubject, msg.Text)
	}

	var auth smtp.Auth
	if s.config.Username != "" || s.config.Password != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp authentication failed: %w", err)
		}
	}
	if err := client.Mail(s.config.FromAddress); err != nil {
		return fmt.Errorf("smtp sender rejected: %w", err)
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("smtp recipient rejected: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp message rejected: %w", err)
	}
	if _, err := writer.Write([]byte(rawMessage)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp message write failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp delivery failed: %w", err)
	}
	return nil
}
