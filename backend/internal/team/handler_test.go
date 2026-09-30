package team

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConsumeRateLimitsRepeatedPublicInvitationAttempts(t *testing.T) {
	service, err := NewService(unavailableConsumeStore{}, auth.NewPasswordHasher(auth.DefaultPasswordParams()), []byte("test-pepper"), "http://localhost:8080", nil, nil)
	require.NoError(t, err)
	handler, err := NewHandler(service, "http://localhost:8080")
	require.NoError(t, err)

	for attempt := 1; attempt <= 5; attempt++ {
		response := consumeInvitationRequest(handler)
		require.Equalf(t, http.StatusUnprocessableEntity, response.Code, "attempt %d", attempt)
	}

	response := consumeInvitationRequest(handler)
	require.Equal(t, http.StatusTooManyRequests, response.Code)
	require.Equal(t, "900", response.Header().Get("Retry-After"))
}

func consumeInvitationRequest(handler *Handler) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/team/invitations/consume", bytes.NewBufferString(`{"token":"VHVKYWpXTllUSWVSc09KU2hwU3dPUUdPbG5IY3JvRGFGb0h6SWtfQ0hXdw","name":"Demo User","password":"secure-password"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:8080")
	context.Request = request
	handler.Consume(context)
	return response
}

type unavailableConsumeStore struct{ Store }

func (unavailableConsumeStore) ConsumeInvitation(context.Context, []byte, string, string, time.Time) (ConsumeResult, error) {
	return ConsumeResult{}, ErrInvitationUnavailable
}
