package signup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const testOrigin = "https://tallerflowpe.com"

type stubUseCases struct {
	registerErr error
	resendErr   error
	verifyErr   error
	input       Input
	verified    string
}

func (s *stubUseCases) Register(_ context.Context, input Input) error {
	s.input = input
	return s.registerErr
}

func (s *stubUseCases) Resend(context.Context, string) error { return s.resendErr }

func (s *stubUseCases) Verify(_ context.Context, token string, _ auth.SessionMetadata) (auth.RawSession, error) {
	s.verified = token
	if s.verifyErr != nil {
		return auth.RawSession{}, s.verifyErr
	}
	return auth.RawSession{Token: "issued-session", CSRFToken: "issued-csrf", ExpiresAt: time.Date(2026, 10, 22, 20, 0, 0, 0, time.UTC)}, nil
}

type countingLimiter struct {
	limit int
	seen  int
}

func (l *countingLimiter) Allow(string) bool {
	l.seen++
	return l.seen <= l.limit
}

func newTestRouter(t *testing.T, service UseCases, limiter RequestLimiter) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	if limiter == nil {
		limiter = &countingLimiter{limit: 100}
	}
	handler, err := NewHandler(service, HandlerConfig{
		AllowedOrigin: testOrigin, Environment: "production", Secure: true,
		RegisterLimiter: limiter, ResendLimiter: limiter, VerifyLimiter: limiter,
	})
	require.NoError(t, err)
	router := gin.New()
	RegisterRoutes(router, handler)
	return router
}

func post(router http.Handler, path, origin, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	return payload.Error.Code
}

const signupBody = `{"workshop_name":"Confecciones Lucho","owner_name":"Lucho","email":"lucho@taller.pe","password":"Secure signup password 2026!","accept_terms":true}`

func TestSignupAcceptsAValidRequestWith202AndNoCookie(t *testing.T) {
	service := &stubUseCases{}
	recorder := post(newTestRouter(t, service, nil), "/api/v1/signup", testOrigin, signupBody)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.JSONEq(t, `{"data":{}}`, recorder.Body.String())
	require.Empty(t, recorder.Result().Cookies(), "no session exists until the email is verified")
	require.True(t, service.input.AcceptTerms)
	require.Equal(t, "lucho@taller.pe", service.input.Email)
}

func TestSignupMapsEveryRefusal(t *testing.T) {
	cases := []struct {
		name    string
		origin  string
		body    string
		err     error
		status  int
		code    string
		limiter RequestLimiter
	}{
		{name: "foreign origin", origin: "https://evil.example", body: signupBody, status: http.StatusForbidden, code: "ORIGIN_INVALID"},
		{name: "missing origin", origin: "", body: signupBody, status: http.StatusForbidden, code: "ORIGIN_INVALID"},
		{name: "closed", origin: testOrigin, body: signupBody, err: ErrClosed, status: http.StatusForbidden, code: "SIGNUP_CLOSED"},
		{name: "invalid fields", origin: testOrigin, body: signupBody, err: &ValidationError{Fields: []string{"accept_terms"}}, status: http.StatusUnprocessableEntity, code: "SIGNUP_VALIDATION_FAILED"},
		{name: "unknown field", origin: testOrigin, body: `{"email":"a@b.pe","is_admin":true}`, status: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "two JSON values", origin: testOrigin, body: signupBody + signupBody, status: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "rate limited", origin: testOrigin, body: signupBody, status: http.StatusTooManyRequests, code: "SIGNUP_RATE_LIMITED", limiter: &countingLimiter{limit: 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := post(newTestRouter(t, &stubUseCases{registerErr: tc.err}, tc.limiter), "/api/v1/signup", tc.origin, tc.body)
			require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
			require.Equal(t, tc.code, errorCode(t, recorder))
		})
	}
}

func TestSignupValidationErrorListsTheFields(t *testing.T) {
	recorder := post(newTestRouter(t, &stubUseCases{registerErr: &ValidationError{Fields: []string{"password", "accept_terms"}}}, nil), "/api/v1/signup", testOrigin, signupBody)

	require.JSONEq(t, `["password","accept_terms"]`, mustField(t, recorder.Body.Bytes()))
}

func TestVerifySetsTheSessionCookieAndReturnsTheCSRFToken(t *testing.T) {
	service := &stubUseCases{}
	recorder := post(newTestRouter(t, service, nil), "/api/v1/auth/email-verifications/consume", testOrigin, `{"token":"`+strings.Repeat("a", 43)+`"}`)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"data":{"expires_at":"2026-10-22T20:00:00Z","csrf_token":"issued-csrf","must_change_password":false}}`, recorder.Body.String())
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, auth.SessionCookieName, cookies[0].Name)
	require.Equal(t, "issued-session", cookies[0].Value)
	require.True(t, cookies[0].HttpOnly)
	require.True(t, cookies[0].Secure)
}

func TestVerifyRefusals(t *testing.T) {
	body := `{"token":"` + strings.Repeat("a", 43) + `"}`
	recorder := post(newTestRouter(t, &stubUseCases{verifyErr: ErrTokenInvalid}, nil), "/api/v1/auth/email-verifications/consume", testOrigin, body)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, "VERIFICATION_TOKEN_INVALID", errorCode(t, recorder))
	require.Empty(t, recorder.Result().Cookies())

	recorder = post(newTestRouter(t, &stubUseCases{}, nil), "/api/v1/auth/email-verifications/consume", "https://evil.example", body)
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Equal(t, "CSRF_INVALID", errorCode(t, recorder))
}

func TestResendAlwaysAnswers202ForValidRequests(t *testing.T) {
	recorder := post(newTestRouter(t, &stubUseCases{}, nil), "/api/v1/auth/email-verifications/resend", testOrigin, `{"email":"anyone@example.com"}`)
	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.JSONEq(t, `{"data":{}}`, recorder.Body.String())

	recorder = post(newTestRouter(t, &stubUseCases{}, &countingLimiter{limit: 0}), "/api/v1/auth/email-verifications/resend", testOrigin, `{"email":"anyone@example.com"}`)
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "900", recorder.Header().Get("Retry-After"))
}

func TestNewHandlerRefusesInsecureProductionCookies(t *testing.T) {
	limiter := &countingLimiter{limit: 1}
	_, err := NewHandler(&stubUseCases{}, HandlerConfig{AllowedOrigin: testOrigin, Environment: "production", Secure: false,
		RegisterLimiter: limiter, ResendLimiter: limiter, VerifyLimiter: limiter})
	require.Error(t, err)
}

func mustField(t *testing.T, body []byte) string {
	t.Helper()
	var payload struct {
		Error struct {
			Details struct {
				Fields json.RawMessage `json:"fields"`
			} `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	return string(payload.Error.Details.Fields)
}
