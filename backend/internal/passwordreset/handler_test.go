package passwordreset

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupTestRouter(service *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(service)
	RegisterRoutes(router, handler)
	return router
}

func TestHandlerRequestReset(t *testing.T) {
	repo := newMockRepository()
	existingID := uuid.New()
	repo.users["owner@tallerflow.pe"] = existingID

	delivery := NewMemoryDelivery()
	service := NewService(repo, delivery, mockHasher{}, ServiceConfig{}, nil)
	router := setupTestRouter(service)

	t.Run("returns 200 for existing email and triggers delivery", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"email": "owner@tallerflow.pe"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-resets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)
		var response struct {
			Data struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
		require.Equal(t, "REQUESTED", response.Data.Status)
		require.NotEmpty(t, response.Data.Message)
		require.Len(t, delivery.Deliveries(), 1)
	})

	t.Run("returns identical 200 for non-existing email without delivery (anti-enumeration)", func(t *testing.T) {
		delivery.Reset()
		body, _ := json.Marshal(map[string]string{"email": "nonexistent@tallerflow.pe"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-resets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)
		var response struct {
			Data struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
		require.Equal(t, "REQUESTED", response.Data.Status)
		require.Empty(t, delivery.Deliveries())
	})

	t.Run("returns 400 for empty or invalid email", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"email": ""})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-resets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		require.Equal(t, http.StatusBadRequest, res.Code)
	})

	t.Run("returns 429 when rate limit is exceeded", func(t *testing.T) {
		repo.rateLimitAllowed = false
		body, _ := json.Marshal(map[string]string{"email": "owner@tallerflow.pe"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-resets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		require.Equal(t, http.StatusTooManyRequests, res.Code)
		require.Equal(t, "900", res.Header().Get("Retry-After"))
		repo.rateLimitAllowed = true
	})
}

func TestHandlerConsumeReset(t *testing.T) {
	repo := newMockRepository()
	existingID := uuid.New()
	repo.users["owner@tallerflow.pe"] = existingID

	delivery := NewMemoryDelivery()
	currentTime := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return currentTime }

	service := NewService(repo, delivery, mockHasher{}, ServiceConfig{}, clock)
	router := setupTestRouter(service)

	// Generate valid token
	require.NoError(t, service.RequestReset(t.Context(), "owner@tallerflow.pe", "127.0.0.1"))
	last, ok := delivery.LastDelivery()
	require.True(t, ok)
	rawToken := strings.Split(last.ResetURL, "token=")[1]

	t.Run("returns 400 if new password is too weak", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"token":        rawToken,
			"new_password": "short",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-resets/consume", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		require.Equal(t, http.StatusBadRequest, res.Code)
		var errResponse struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &errResponse))
		require.Equal(t, "PASSWORD_TOO_WEAK", errResponse.Error.Code)
	})

	t.Run("returns 200 on successful consumption and password update", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"token":        rawToken,
			"new_password": "ValidNewPassword123!",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-resets/consume", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)
		var response struct {
			Data struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
		require.Equal(t, "PASSWORD_RESET_COMPLETED", response.Data.Status)
	})

	t.Run("returns 400 with TOKEN_ALREADY_USED on second attempt", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"token":        rawToken,
			"new_password": "AnotherNewPassword123!",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-resets/consume", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		require.Equal(t, http.StatusBadRequest, res.Code)
		var errResponse struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &errResponse))
		require.Equal(t, "TOKEN_ALREADY_USED", errResponse.Error.Code)
	})

	t.Run("returns 400 with TOKEN_INVALID on corrupted token", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"token":        "corrupted-token-012345678901234567890123456789",
			"new_password": "ValidNewPassword123!",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-resets/consume", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		require.Equal(t, http.StatusBadRequest, res.Code)
		var errResponse struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &errResponse))
		require.Equal(t, "TOKEN_INVALID", errResponse.Error.Code)
	})
}
