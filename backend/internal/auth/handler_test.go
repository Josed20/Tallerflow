package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	handlerOrigin   = "https://app.tallerflow.test"
	handlerToken    = "opaque-session-token-for-handler-tests"
	handlerCSRF     = "csrf-token-for-handler-tests"
	handlerPassword = "Temporary secure passphrase 9!"
)

func TestLoginCookieAndSessionRestoration(t *testing.T) {
	fixture := newHandlerFixture(t, false)
	foreign := fixture.request(http.MethodPost, "/api/v1/auth/login", `{"email":"owner@example.com","password":"`+handlerPassword+`"}`, "https://attacker.example", "", "")
	if foreign.Code != http.StatusForbidden || responseErrorCode(t, foreign) != "ORIGIN_INVALID" || len(foreign.Result().Cookies()) != 0 {
		t.Fatalf("foreign-origin login response = %d %s, cookies = %v", foreign.Code, foreign.Body.String(), foreign.Result().Cookies())
	}
	login := fixture.request(http.MethodPost, "/api/v1/auth/login", `{"email":"owner@example.com","password":"`+handlerPassword+`"}`, handlerOrigin, "", "")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", login.Code, login.Body.String())
	}
	cookie := findSessionCookie(t, login)
	if cookie.Name != SessionCookieName || cookie.Value != handlerToken || cookie.Path != "/" || !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("login cookie does not satisfy __Host cookie policy: %+v", cookie)
	}
	if strings.Contains(login.Body.String(), handlerToken) {
		t.Fatal("login JSON exposed the opaque session token")
	}
	assertSessionData(t, login, handlerCSRF, false)

	restore := fixture.request(http.MethodGet, "/api/v1/auth/session", "", "", "", handlerToken)
	if restore.Code != http.StatusOK {
		t.Fatalf("restore status = %d, body = %s", restore.Code, restore.Body.String())
	}
	assertSessionData(t, restore, handlerCSRF, false)
	if strings.Contains(restore.Body.String(), handlerToken) {
		t.Fatal("restore JSON exposed the opaque session token")
	}

	fixture.useCases.invalid = true
	invalid := fixture.request(http.MethodGet, "/api/v1/auth/session", "", "", "", handlerToken)
	if invalid.Code != http.StatusUnauthorized || responseErrorCode(t, invalid) != "SESSION_INVALID" {
		t.Fatalf("invalid session response = %d %s", invalid.Code, invalid.Body.String())
	}
	cleared := findSessionCookie(t, invalid)
	if cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Fatalf("invalid session cookie was not expired: %+v", cleared)
	}
}

func TestNewHandlerRejectsInsecureProductionCookie(t *testing.T) {
	_, err := NewHandler(&handlerUseCases{}, HandlerConfig{AllowedOrigin: handlerOrigin, Environment: "production", Secure: false})
	if err == nil {
		t.Fatal("production handler accepted insecure session cookies")
	}
}

func TestRequestIPIgnoresUntrustedForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	var got string
	router.GET("/ip", func(c *gin.Context) {
		got = requestIP(c)
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/ip", nil)
	request.RemoteAddr = "192.0.2.44:54321"
	request.Header.Set("X-Forwarded-For", "203.0.113.99")

	router.ServeHTTP(httptest.NewRecorder(), request)

	if got != "192.0.2.44" {
		t.Fatalf("requestIP() = %q, want direct untrusted peer", got)
	}
}

func TestLoginMapsCredentialFailureAndThrottleWithoutSettingCookie(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		err        error
		status     int
		retryAfter bool
	}{
		{name: "invalid credentials", err: ErrInvalidCredentials, status: http.StatusUnauthorized, code: "AUTH_INVALID_CREDENTIALS"},
		{name: "rate limited", err: ErrTooManyAttempts, status: http.StatusTooManyRequests, code: "AUTH_RATE_LIMITED", retryAfter: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newHandlerFixture(t, false)
			fixture.useCases.loginErr = tc.err
			res := fixture.request(http.MethodPost, "/api/v1/auth/login", `{"email":"owner@example.com","password":"wrong"}`, handlerOrigin, "", "")
			if res.Code != tc.status || responseErrorCode(t, res) != tc.code || len(res.Result().Cookies()) != 0 {
				t.Fatalf("login response = %d %s, cookies = %v", res.Code, res.Body.String(), res.Result().Cookies())
			}
			if tc.retryAfter && res.Header().Get("Retry-After") == "" {
				t.Fatal("rate-limited login has no Retry-After header")
			}
		})
	}
}

func TestLogoutEnforcesCSRFAndOriginBeforeRevokingSession(t *testing.T) {
	fixture := newHandlerFixture(t, false)
	for _, tc := range []struct {
		name, origin, csrf string
	}{
		{name: "missing CSRF", origin: handlerOrigin},
		{name: "wrong CSRF", origin: handlerOrigin, csrf: "wrong-token"},
		{name: "foreign origin", origin: "https://attacker.example", csrf: handlerCSRF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := fixture.request(http.MethodPost, "/api/v1/auth/logout", "", tc.origin, tc.csrf, handlerToken)
			if res.Code != http.StatusForbidden || responseErrorCode(t, res) != "CSRF_INVALID" {
				t.Fatalf("logout response = %d %s, want 403 CSRF_INVALID", res.Code, res.Body.String())
			}
			restore := fixture.request(http.MethodGet, "/api/v1/auth/session", "", "", "", handlerToken)
			if restore.Code != http.StatusOK {
				t.Fatalf("rejected logout revoked the session: %d %s", restore.Code, restore.Body.String())
			}
		})
	}

	logout := fixture.request(http.MethodPost, "/api/v1/auth/logout", "", handlerOrigin, handlerCSRF, handlerToken)
	if logout.Code != http.StatusOK {
		t.Fatalf("valid logout response = %d %s", logout.Code, logout.Body.String())
	}
	if cookie := findSessionCookie(t, logout); cookie.Value != "" || cookie.MaxAge >= 0 {
		t.Fatalf("logout cookie was not expired: %+v", cookie)
	}
	restore := fixture.request(http.MethodGet, "/api/v1/auth/session", "", "", "", handlerToken)
	if restore.Code != http.StatusUnauthorized || responseErrorCode(t, restore) != "SESSION_INVALID" {
		t.Fatalf("logged-out session remained valid: %d %s", restore.Code, restore.Body.String())
	}
}

func TestMandatoryPasswordChangeGatesPrivateRouteAndRotatesSession(t *testing.T) {
	fixture := newHandlerFixture(t, true)
	private := fixture.request(http.MethodGet, "/api/v1/private", "", "", "", handlerToken)
	if private.Code != http.StatusForbidden || responseErrorCode(t, private) != "PASSWORD_CHANGE_REQUIRED" {
		t.Fatalf("private route response = %d %s", private.Code, private.Body.String())
	}
	restore := fixture.request(http.MethodGet, "/api/v1/auth/session", "", "", "", handlerToken)
	if restore.Code != http.StatusOK {
		t.Fatalf("mandatory-change session cannot be restored: %d %s", restore.Code, restore.Body.String())
	}
	assertSessionData(t, restore, handlerCSRF, true)

	change := fixture.request(http.MethodPost, "/api/v1/auth/change-password", `{"current_password":"`+handlerPassword+`","new_password":"New owner passphrase 10!"}`, handlerOrigin, handlerCSRF, handlerToken)
	if change.Code != http.StatusOK {
		t.Fatalf("change-password response = %d %s", change.Code, change.Body.String())
	}
	newCookie := findSessionCookie(t, change)
	if newCookie.Value == "" || newCookie.Value == handlerToken {
		t.Fatalf("password change did not rotate the session cookie: %+v", newCookie)
	}
	assertSessionData(t, change, "replacement-csrf-token", false)
	old := fixture.request(http.MethodGet, "/api/v1/auth/session", "", "", "", handlerToken)
	if old.Code != http.StatusUnauthorized {
		t.Fatalf("old session survived rotation: %d %s", old.Code, old.Body.String())
	}
	private = fixture.request(http.MethodGet, "/api/v1/private", "", "", "", newCookie.Value)
	if private.Code != http.StatusOK {
		t.Fatalf("private route stayed gated after password change: %d %s", private.Code, private.Body.String())
	}
}

type handlerUseCases struct {
	invalid    bool
	mustChange bool
	token      string
	csrf       string
	loginErr   error
}

func (u *handlerUseCases) session() AuthSession {
	return AuthSession{Token: u.token, CSRFToken: u.csrf, ExpiresAt: testNow.Add(8 * time.Hour), MustChangePassword: u.mustChange}
}

func (u *handlerUseCases) Login(_ context.Context, _, _ string, _ string) (AuthSession, error) {
	if u.loginErr != nil {
		return AuthSession{}, u.loginErr
	}
	return u.session(), nil
}

func (u *handlerUseCases) Restore(_ context.Context, token string) (AuthSession, error) {
	if u.invalid || token != u.token {
		return AuthSession{}, ErrSessionInvalid
	}
	return u.session(), nil
}

func (u *handlerUseCases) Logout(_ context.Context, token string) error {
	if u.invalid || token != u.token {
		return ErrSessionInvalid
	}
	u.invalid = true
	return nil
}

func (u *handlerUseCases) ChangePassword(_ context.Context, token, current, next string) (AuthSession, error) {
	if u.invalid || token != u.token {
		return AuthSession{}, ErrSessionInvalid
	}
	if current != handlerPassword || next != "New owner passphrase 10!" {
		return AuthSession{}, errors.New("invalid test credentials")
	}
	u.token = "rotated-opaque-session-token"
	u.csrf = "replacement-csrf-token"
	u.mustChange = false
	return u.session(), nil
}

type handlerFixture struct {
	router   *gin.Engine
	useCases *handlerUseCases
}

func newHandlerFixture(t *testing.T, mustChange bool) *handlerFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	u := &handlerUseCases{token: handlerToken, csrf: handlerCSRF, mustChange: mustChange}
	handler, err := NewHandler(u, HandlerConfig{AllowedOrigin: handlerOrigin, Environment: "production", Secure: true})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	RegisterRoutes(router, handler)
	router.GET("/api/v1/private", handler.RequireSession(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
	})
	return &handlerFixture{router: router, useCases: u}
}

func (f *handlerFixture) request(method, path, body, origin, csrf, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	}
	res := httptest.NewRecorder()
	f.router.ServeHTTP(res, req)
	return res
}

func findSessionCookie(t *testing.T, res *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == SessionCookieName {
			return cookie
		}
	}
	t.Fatalf("response has no %s cookie: %v", SessionCookieName, res.Header().Values("Set-Cookie"))
	return nil
}

func responseErrorCode(t *testing.T, res *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Error.Code
}

func assertSessionData(t *testing.T, res *httptest.ResponseRecorder, csrf string, mustChange bool) {
	t.Helper()
	var payload struct {
		Data struct {
			ExpiresAt          string `json:"expires_at"`
			CSRFToken          string `json:"csrf_token"`
			MustChangePassword bool   `json:"must_change_password"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.CSRFToken != csrf || payload.Data.MustChangePassword != mustChange || payload.Data.ExpiresAt != testNow.Add(8*time.Hour).Format(time.RFC3339) {
		t.Fatalf("session response = %+v, want CSRF %q, must-change %t and eight-hour UTC expiry", payload.Data, csrf, mustChange)
	}
}
