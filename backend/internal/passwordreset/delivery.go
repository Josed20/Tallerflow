package passwordreset

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// PasswordResetDelivery delivers the only copy of the raw reset token.
// Implementations must never log the URL or token.
type PasswordResetDelivery interface {
	DeliverPasswordReset(context.Context, string, string) error
}

type SMTPConfig struct {
	Address, Username, Password, From string
	UseTLS                            bool
}

type SMTPDelivery struct{ config SMTPConfig }

func NewSMTPDelivery(config SMTPConfig) (*SMTPDelivery, error) {
	if strings.TrimSpace(config.Address) == "" || strings.TrimSpace(config.From) == "" {
		return nil, fmt.Errorf("smtp address and sender are required")
	}
	return &SMTPDelivery{config: config}, nil
}

func (d *SMTPDelivery) DeliverPasswordReset(ctx context.Context, recipient, resetURL string) error {
	host, _, err := net.SplitHostPort(d.config.Address)
	if err != nil {
		return fmt.Errorf("invalid smtp address: %w", err)
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", d.config.Address)
	if err != nil {
		return fmt.Errorf("connect smtp: %w", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("start smtp: %w", err)
	}
	defer client.Close()
	if d.config.UseTLS {
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("secure smtp: %w", err)
		}
	}
	if d.config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", d.config.Username, d.config.Password, host)); err != nil {
			return fmt.Errorf("authenticate smtp: %w", err)
		}
	}
	if err := client.Mail(d.config.From); err != nil {
		return fmt.Errorf("set smtp sender: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("set smtp recipient: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("open smtp message: %w", err)
	}
	body := "From: " + d.config.From + "\r\nTo: " + recipient + "\r\nSubject: Recupera tu acceso a TallerFlow\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nAbre este enlace para crear una nueva contraseña:\r\n" + resetURL + "\r\n\r\nEl enlace vence en una hora. Si no lo solicitaste, ignora este mensaje.\r\n"
	if _, err = w.Write([]byte(body)); err != nil {
		_ = w.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close smtp message: %w", err)
	}
	return client.Quit()
}
