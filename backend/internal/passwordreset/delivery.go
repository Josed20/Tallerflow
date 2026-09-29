package passwordreset

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PasswordResetDelivery interface {
	Deliver(ctx context.Context, recipientEmail, resetURL string) error
}

type SMTPDeliveryConfig struct {
	Host        string
	Port        string
	Username    string
	Password    string
	FromAddress string
	RequireTLS  bool
	Timeout     time.Duration
}

type SMTPDelivery struct {
	config SMTPDeliveryConfig
}

func NewSMTPDelivery(cfg SMTPDeliveryConfig) (*SMTPDelivery, error) {
	cfg.Host = strings.TrimSpace(cfg.Host)
	if cfg.FromAddress == "" {
		cfg.FromAddress = "soporte@tallerflow.pe"
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
	if !cfg.RequireTLS && !isLoopbackSMTPHost(cfg.Host) {
		return nil, fmt.Errorf("SMTP TLS is required for non-loopback hosts")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &SMTPDelivery{config: cfg}, nil
}

func isLoopbackSMTPHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (d *SMTPDelivery) Deliver(ctx context.Context, recipientEmail, resetURL string) error {
	trimmedEmail := strings.TrimSpace(recipientEmail)
	if trimmedEmail == "" {
		return fmt.Errorf("recipient email is required")
	}
	recipient, err := mail.ParseAddress(trimmedEmail)
	if err != nil || recipient.Address != trimmedEmail {
		return fmt.Errorf("recipient email is invalid")
	}
	if strings.ContainsAny(resetURL, "\r\n") {
		return fmt.Errorf("reset URL is invalid")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, d.config.Timeout)
	defer cancel()

	addr := net.JoinHostPort(d.config.Host, d.config.Port)
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

	client, err := smtp.NewClient(connection, d.config.Host)
	if err != nil {
		return fmt.Errorf("smtp client failed: %w", err)
	}
	defer client.Quit()
	if d.config.RequireTLS {
		supportsTLS, _ := client.Extension("STARTTLS")
		if !supportsTLS {
			return fmt.Errorf("smtp server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: d.config.Host}); err != nil {
			return fmt.Errorf("smtp STARTTLS failed: %w", err)
		}
	}

	subject := "Restablecimiento de contraseña — TallerFlow"
	body := fmt.Sprintf("Hola,\r\n\r\n"+
		"Recibimos una solicitud para restablecer la contraseña de tu cuenta en TallerFlow.\r\n\r\n"+
		"Puedes crear una nueva contraseña ingresando en el siguiente enlace:\r\n"+
		"%s\r\n\r\n"+
		"Este enlace es válido por 1 hora y solo puede ser utilizado una vez.\r\n"+
		"Si no solicitaste este cambio, puedes ignorar este correo de forma segura.\r\n\r\n"+
		"— Equipo TallerFlow", resetURL)

	message := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n"+
		"%s", d.config.FromAddress, trimmedEmail, subject, body)

	var auth smtp.Auth
	if d.config.Username != "" || d.config.Password != "" {
		auth = smtp.PlainAuth("", d.config.Username, d.config.Password, d.config.Host)
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp authentication failed: %w", err)
		}
	}
	if err := client.Mail(d.config.FromAddress); err != nil {
		return fmt.Errorf("smtp sender rejected: %w", err)
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("smtp recipient rejected: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp message rejected: %w", err)
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp message write failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp delivery failed: %w", err)
	}
	return nil
}

// MemoryDelivery is an in-memory delivery implementation ideal for tests and dry runs.
type MemoryDelivery struct {
	mu         sync.Mutex
	deliveries []DeliveryRecord
}

type DeliveryRecord struct {
	RecipientEmail string
	ResetURL       string
}

func NewMemoryDelivery() *MemoryDelivery {
	return &MemoryDelivery{deliveries: make([]DeliveryRecord, 0)}
}

func (m *MemoryDelivery) Deliver(_ context.Context, recipientEmail, resetURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveries = append(m.deliveries, DeliveryRecord{
		RecipientEmail: strings.TrimSpace(recipientEmail),
		ResetURL:       resetURL,
	})
	return nil
}

func (m *MemoryDelivery) Deliveries() []DeliveryRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]DeliveryRecord, len(m.deliveries))
	copy(copied, m.deliveries)
	return copied
}

func (m *MemoryDelivery) LastDelivery() (DeliveryRecord, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.deliveries) == 0 {
		return DeliveryRecord{}, false
	}
	return m.deliveries[len(m.deliveries)-1], true
}

func (m *MemoryDelivery) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveries = m.deliveries[:0]
}
