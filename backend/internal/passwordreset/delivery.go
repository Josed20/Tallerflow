package passwordreset

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"sync"
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
}

type SMTPDelivery struct {
	config SMTPDeliveryConfig
}

func NewSMTPDelivery(cfg SMTPDeliveryConfig) *SMTPDelivery {
	if cfg.FromAddress == "" {
		cfg.FromAddress = "soporte@tallerflow.pe"
	}
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == "" {
		cfg.Port = "1025"
	}
	return &SMTPDelivery{config: cfg}
}

func (d *SMTPDelivery) Deliver(_ context.Context, recipientEmail, resetURL string) error {
	trimmedEmail := strings.TrimSpace(recipientEmail)
	if trimmedEmail == "" {
		return fmt.Errorf("recipient email is required")
	}

	addr := fmt.Sprintf("%s:%s", d.config.Host, d.config.Port)
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

	err := smtp.SendMail(addr, auth, d.config.FromAddress, []string{trimmedEmail}, []byte(message))
	if err != nil {
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
