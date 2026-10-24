package mail

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFakeSender(t *testing.T) {
	fake := NewFake()
	ctx := context.Background()

	msg1 := Message{
		To:      "test1@example.com",
		Subject: "Subject 1",
		Text:    "Text body 1",
	}
	msg2 := Message{
		To:      "test2@example.com",
		Subject: "Subject 2",
		Text:    "Text body 2",
		HTML:    "<p>HTML body 2</p>",
	}

	require.NoError(t, fake.Send(ctx, msg1))
	require.NoError(t, fake.Send(ctx, msg2))

	messages := fake.Messages()
	require.Len(t, messages, 2)
	require.Equal(t, msg1, messages[0])
	require.Equal(t, msg2, messages[1])

	last, ok := fake.LastMessage()
	require.True(t, ok)
	require.Equal(t, msg2, last)

	fake.Reset()
	require.Empty(t, fake.Messages())
	_, ok = fake.LastMessage()
	require.False(t, ok)

	// Error simulation
	expectedErr := errors.New("simulated mail error")
	fake.SetError(expectedErr)
	err := fake.Send(ctx, msg1)
	require.Equal(t, expectedErr, err)
	require.Empty(t, fake.Messages())

	fake.Reset()
	require.NoError(t, fake.Send(ctx, msg1))
	require.Len(t, fake.Messages(), 1)
}

func TestFakeSenderConcurrency(t *testing.T) {
	fake := NewFake()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = fake.Send(ctx, Message{
				To:      "user@example.com",
				Subject: "Concurrent",
				Text:    "Body",
			})
		}(i)
	}
	wg.Wait()

	require.Len(t, fake.Messages(), 50)
}

func TestRenderTemplates(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		data         any
		expectSubj   string
		containsText []string
	}{
		{
			name:         "verify-email",
			templateName: "verify-email",
			data: map[string]string{
				"Name":         "Carlos",
				"WorkshopName": "Taller Gamarra",
				"URL":          "https://tallerflowpe.com/verificar?token=xyz",
				"ExpiresIn":    "24 horas",
			},
			expectSubj: "Verifica tu correo — TallerFlow",
			containsText: []string{
				"Carlos",
				"«Taller Gamarra»",
				"https://tallerflowpe.com/verificar?token=xyz",
				"24 horas",
			},
		},
		{
			name:         "team-invitation",
			templateName: "team-invitation",
			data: map[string]string{
				"WorkshopName": "Taller Gamarra",
				"InviterName":  "Lucho",
				"Role":         "OPERATOR",
				"URL":          "https://tallerflowpe.com/join?token=abc",
				"ExpiresIn":    "7 días",
			},
			expectSubj: "Invitación para unirte a Taller Gamarra en TallerFlow",
			containsText: []string{
				"Lucho",
				"«Taller Gamarra»",
				"OPERATOR",
				"https://tallerflowpe.com/join?token=abc",
				"7 días",
			},
		},
		{
			name:         "password-reset",
			templateName: "password-reset",
			data: map[string]string{
				"URL":       "https://tallerflowpe.com/reset-password?token=123",
				"ExpiresIn": "1 hora",
			},
			expectSubj: "Restablecimiento de contraseña — TallerFlow",
			containsText: []string{
				"https://tallerflowpe.com/reset-password?token=123",
				"1 hora",
			},
		},
		{
			name:         "workshop-suspended",
			templateName: "workshop-suspended",
			data: map[string]string{
				"WorkshopName": "Taller Gamarra",
				"Reason":       "Falta de pago",
				"SupportEmail": "soporte@tallerflowpe.com",
			},
			expectSubj: "Tu taller ha sido suspendido — TallerFlow",
			containsText: []string{
				"«Taller Gamarra»",
				"Falta de pago",
				"soporte@tallerflowpe.com",
			},
		},
		{
			name:         "workshop-deletion-scheduled",
			templateName: "workshop-deletion-scheduled",
			data: map[string]string{
				"WorkshopName": "Taller Gamarra",
				"DeletionDate": "2026-11-20",
				"SupportEmail": "soporte@tallerflowpe.com",
			},
			expectSubj: "Eliminación programada de taller — TallerFlow",
			containsText: []string{
				"«Taller Gamarra»",
				"2026-11-20",
				"soporte@tallerflowpe.com",
				"30 días",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := Render(tt.templateName, tt.data)
			require.NoError(t, err)
			require.Equal(t, tt.expectSubj, msg.Subject)
			require.NotEmpty(t, msg.Text)
			require.NotEmpty(t, msg.HTML)
			require.Contains(t, msg.HTML, "#1F3A68")
			require.Contains(t, msg.HTML, "TallerFlow")
			for _, expectedStr := range tt.containsText {
				require.Contains(t, msg.Text, expectedStr)
			}
		})
	}
}

func TestRenderUnknownTemplate(t *testing.T) {
	_, err := Render("unknown-template", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown email template")
}

func TestNewSMTPSenderRejectsUnsafeAuthentication(t *testing.T) {
	_, err := NewSMTPSender(SMTPConfig{
		Host:     "smtp.example.com",
		Port:     "587",
		Username: "smtp-user",
		Password: "smtp-password",
	})
	require.Error(t, err)
}

func TestNewSMTPSenderRejectsPlaintextRemoteRelay(t *testing.T) {
	_, err := NewSMTPSender(SMTPConfig{
		Host: "smtp.example.com",
		Port: "25",
	})
	require.Error(t, err, "remote SMTP relays must require TLS even without credentials")
}

func TestNewSMTPSenderAcceptsMailpitWithoutCredentials(t *testing.T) {
	sender, err := NewSMTPSender(SMTPConfig{
		Host:    "mailpit",
		Port:    "1025",
		Timeout: time.Second,
	})
	require.NoError(t, err)
	require.False(t, sender.config.RequireTLS)
	require.Equal(t, "mailpit", sender.config.Host)
	require.Equal(t, time.Second, sender.config.Timeout)
}

func TestNewSMTPSenderValidatesFromAndPort(t *testing.T) {
	_, err := NewSMTPSender(SMTPConfig{
		Host:        "localhost",
		Port:        "invalid",
		FromAddress: "soporte@tallerflowpe.com",
	})
	require.Error(t, err)

	_, err = NewSMTPSender(SMTPConfig{
		Host:        "localhost",
		Port:        "1025",
		FromAddress: "not-an-email",
	})
	require.Error(t, err)
}
