package passwordreset

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/mail"
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
	sender mail.Sender
}

func NewSMTPDelivery(cfg SMTPDeliveryConfig) (*SMTPDelivery, error) {
	sender, err := mail.NewSMTPSender(mail.SMTPConfig{
		Host:        cfg.Host,
		Port:        cfg.Port,
		Username:    cfg.Username,
		Password:    cfg.Password,
		FromAddress: cfg.FromAddress,
		RequireTLS:  cfg.RequireTLS,
		Timeout:     cfg.Timeout,
	})
	if err != nil {
		return nil, err
	}
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
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &SMTPDelivery{config: cfg, sender: sender}, nil
}

func (d *SMTPDelivery) Deliver(ctx context.Context, recipientEmail, resetURL string) error {
	msg, err := mail.Render("password-reset", map[string]string{
		"URL":       resetURL,
		"ExpiresIn": "1 hora",
	})
	if err != nil {
		return fmt.Errorf("render password reset template: %w", err)
	}
	msg.To = recipientEmail
	return d.sender.Send(ctx, msg)
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
