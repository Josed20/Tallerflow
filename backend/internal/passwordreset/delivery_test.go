package passwordreset

import (
	"context"
	"testing"

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
