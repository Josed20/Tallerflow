package onboarding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

func TestHandlerPublishesAvailabilityWithoutIdentityData(t *testing.T) {
	handler := newTestHandler(t, &useCasesFake{status: Status{Available: true}}, limiterFake(true))
	router := gin.New()
	RegisterRoutes(router, handler)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/status", nil))

	if response.Code != http.StatusOK || response.Body.String() != `{"data":{"available":true}}` {
		t.Fatalf("status response = %d %s", response.Code, response.Body.String())
	}
}

func TestHandlerCreatesOwnerSessionCookieWithoutReturningOpaqueToken(t *testing.T) {
	result := Result{Token: "raw-session-must-stay-in-cookie", CSRFToken: strings.Repeat("c", 43), ExpiresAt: time.Date(2026, 9, 27, 23, 0, 0, 0, time.UTC)}
	useCases := &useCasesFake{result: result}
	handler := newTestHandler(t, useCases, limiterFake(true))
	router := gin.New()
	RegisterRoutes(router, handler)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/workshop", bytes.NewBufferString(validJSON()))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:8080")
	request.Header.Set("User-Agent", "test-browser")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("create response = %d %s", response.Code, response.Body.String())
	}
	cookie := response.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "tallerflow_session=raw-session-must-stay-in-cookie") || !strings.Contains(cookie, "HttpOnly") || !strings.Contains(cookie, "SameSite=Lax") {
		t.Fatalf("session cookie is incomplete: %q", cookie)
	}
	if strings.Contains(response.Body.String(), result.Token) || !strings.Contains(response.Body.String(), `"must_change_password":false`) {
		t.Fatalf("response exposed the token or wrong password state: %s", response.Body.String())
	}
	if useCases.metadata.UserAgent != "test-browser" {
		t.Fatal("handler lost session metadata")
	}
}

func TestHandlerRejectsForeignOriginBeforeCreatingOwner(t *testing.T) {
	useCases := &useCasesFake{}
	handler := newTestHandler(t, useCases, limiterFake(true))
	router := gin.New()
	RegisterRoutes(router, handler)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/workshop", bytes.NewBufferString(validJSON()))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://evil.example")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden || useCases.createCalls != 0 {
		t.Fatalf("foreign origin response = %d, calls = %d", response.Code, useCases.createCalls)
	}
}

func TestHandlerMapsValidationConflictThrottleAndInternalErrorsSafely(t *testing.T) {
	tests := []struct {
		name    string
		useCase *useCasesFake
		limiter limiterFake
		want    int
		code    string
	}{
		{name: "validation", useCase: &useCasesFake{err: &ValidationError{Fields: map[string]string{"email": "invalid"}}}, limiter: true, want: 422, code: "ONBOARDING_VALIDATION_FAILED"},
		{name: "claimed", useCase: &useCasesFake{err: ErrUnavailable}, limiter: true, want: 409, code: "ONBOARDING_UNAVAILABLE"},
		{name: "throttled", useCase: &useCasesFake{}, limiter: false, want: 429, code: "ONBOARDING_RATE_LIMITED"},
		{name: "internal", useCase: &useCasesFake{err: errors.New("database password secret-value")}, limiter: true, want: 500, code: "INTERNAL_ERROR"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := newTestHandler(t, tc.useCase, tc.limiter)
			router := gin.New()
			RegisterRoutes(router, handler)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/workshop", bytes.NewBufferString(validJSON()))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", "http://localhost:8080")

			router.ServeHTTP(response, request)

			if response.Code != tc.want || !strings.Contains(response.Body.String(), tc.code) || strings.Contains(response.Body.String(), "secret-value") {
				t.Fatalf("response = %d %s, want %d %s", response.Code, response.Body.String(), tc.want, tc.code)
			}
		})
	}
}

func newTestHandler(t *testing.T, useCases UseCases, limiter RequestLimiter) *Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler, err := NewHandler(useCases, HandlerConfig{AllowedOrigin: "http://localhost:8080", Environment: "development", Secure: false, Limiter: limiter})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func validJSON() string {
	payload := map[string]string{"workshop_name": "Taller Demo", "owner_name": "Owner Demo", "email": "owner@example.com", "password": "Secure password 123!", "password_confirmation": "Secure password 123!"}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}

type limiterFake bool

func (l limiterFake) Allow(string) bool { return bool(l) }

type useCasesFake struct {
	status      Status
	result      Result
	err         error
	metadata    auth.SessionMetadata
	createCalls int
}

func (f *useCasesFake) Status(context.Context) (Status, error) { return f.status, f.err }
func (f *useCasesFake) Create(_ context.Context, _ Input, metadata auth.SessionMetadata) (Result, error) {
	f.createCalls++
	f.metadata = metadata
	return f.result, f.err
}
