package team

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
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
		response := consumeInvitationRequest(handler, validInvitationToken(attempt))
		require.Equalf(t, http.StatusUnprocessableEntity, response.Code, "attempt %d", attempt)
	}

	response := consumeInvitationRequest(handler, validInvitationToken(1))
	require.Equal(t, http.StatusTooManyRequests, response.Code)
	require.Equal(t, "900", response.Header().Get("Retry-After"))
}

func TestConsumeRateLimitsAttemptsThatVaryThePublicToken(t *testing.T) {
	service, err := NewService(unavailableConsumeStore{}, auth.NewPasswordHasher(auth.DefaultPasswordParams()), []byte("test-pepper"), "http://localhost:8080", nil, nil)
	require.NoError(t, err)
	handler, err := NewHandler(service, "http://localhost:8080")
	require.NoError(t, err)

	for attempt := 1; attempt <= 5; attempt++ {
		response := consumeInvitationRequest(handler, validInvitationToken(attempt))
		require.Equalf(t, http.StatusUnprocessableEntity, response.Code, "attempt %d", attempt)
	}

	response := consumeInvitationRequest(handler, validInvitationToken(6))
	require.Equal(t, http.StatusTooManyRequests, response.Code)
}

func consumeInvitationRequest(handler *Handler, token string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/team/invitations/consume", bytes.NewBufferString(fmt.Sprintf(`{"token":%q,"name":"Demo User","password":"secure-password"}`, token)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:8080")
	context.Request = request
	handler.Consume(context)
	return response
}

func validInvitationToken(attempt int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("invitation-attempt-%032d", attempt)))
}

type unavailableConsumeStore struct{ Store }

func (unavailableConsumeStore) ConsumeInvitation(context.Context, []byte, string, string, time.Time) (ConsumeResult, error) {
	return ConsumeResult{}, ErrInvitationUnavailable
}
