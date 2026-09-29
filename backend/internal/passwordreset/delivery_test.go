package passwordreset

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMemoryDelivery(t *testing.T) {
	delivery := NewMemoryDelivery()
	ctx := context.Background()

	require.NoError(t, delivery.Deliver(ctx, "user1@example.com", "http://localhost:8080/reset?token=123"))
	require.NoError(t, delivery.Deliver(ctx, "user2@example.com", "http://localhost:8080/reset?token=456"))

	deliveries := delivery.Deliveries()
	require.Len(t, deliveries, 2)
	require.Equal(t, "user1@example.com", deliveries[0].RecipientEmail)
	require.Equal(t, "http://localhost:8080/reset?token=123", deliveries[0].ResetURL)

	last, ok := delivery.LastDelivery()
	require.True(t, ok)
	require.Equal(t, "user2@example.com", last.RecipientEmail)

	delivery.Reset()
	require.Empty(t, delivery.Deliveries())
}

func TestNewSMTPDeliveryRejectsUnsafeAuthenticationConfiguration(t *testing.T) {
	_, err := NewSMTPDelivery(SMTPDeliveryConfig{
		Host:     "smtp.example.com",
		Port:     "587",
		Username: "smtp-user",
		Password: "smtp-password",
	})

	require.Error(t, err)
}

func TestNewSMTPDeliveryRejectsPlaintextRemoteRelay(t *testing.T) {
	_, err := NewSMTPDelivery(SMTPDeliveryConfig{
		Host: "smtp.example.com",
		Port: "25",
	})

	require.Error(t, err, "remote SMTP relays must require TLS even without credentials")
}

func TestNewSMTPDeliveryAcceptsMailpitWithoutCredentials(t *testing.T) {
	delivery, err := NewSMTPDelivery(SMTPDeliveryConfig{Host: "127.0.0.1", Port: "1025", Timeout: time.Second})

	require.NoError(t, err)
	require.False(t, delivery.config.RequireTLS)
	require.Equal(t, time.Second, delivery.config.Timeout)
}
