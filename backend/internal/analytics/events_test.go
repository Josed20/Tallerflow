package analytics

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEventValidationBoundariesAndTrust(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	valid := Event{ID: uuid.New(), InstallationID: uuid.New(), Type: "first_open", OccurredAt: now, Channel: "android", Version: "3"}
	require.NoError(t, valid.Validate(now, true))
	require.Error(t, valid.Validate(now, false))
	for _, mutate := range []func(*Event){
		func(e *Event) { e.ID = uuid.Nil }, func(e *Event) { e.InstallationID = uuid.Nil },
		func(e *Event) { e.Type = "download" }, func(e *Event) { e.Channel = "ios" },
		func(e *Event) { e.Version = "" }, func(e *Event) { e.Version = "sensitive@example.com" },
		func(e *Event) { e.OccurredAt = now.Add(-7*24*time.Hour - time.Second) },
		func(e *Event) { e.OccurredAt = now.Add(5*time.Minute + time.Second) },
	} {
		event := valid
		mutate(&event)
		require.Error(t, event.Validate(now, true))
	}
	valid.Type = "login"
	require.NoError(t, valid.Validate(now, false))
	require.Error(t, valid.Validate(now, true))
}
