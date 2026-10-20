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

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	handlerOrigin       = "https://app.tallerflow.test"
	mobileHandlerOrigin = "https://localhost"
	handlerToken        = "opaque-session-token-for-handler-tests"
	handlerCSRF         = "csrf-token-for-handler-tests"
	handlerPassword     = "Temporary secure passphrase 9!"
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

func TestRequireSessionStoresPrincipalInGin(t *testing.T) {
	fixture := newHandlerFixture(t, false)
	response := fixture.request(http.MethodGet, "/api/v1/private", "", "", "", handlerToken)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), fixture.useCases.principal.UserID.String()) {
		t.Fatalf("private response = %d %s", response.Code, response.Body.String())
	}
}

func TestRequireSessionRejectsMutationsWithoutValidCSRF(t *testing.T) {
	fixture := newHandlerFixture(t, false)
	fixture.router.POST("/api/v1/private-mutation", fixture.handler.RequireSession(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	missing := fixture.request(http.MethodPost, "/api/v1/private-mutation", "", handlerOrigin, "", handlerToken)
	if missing.Code != http.StatusForbidden || responseErrorCode(t, missing) != "CSRF_INVALID" {
		t.Fatalf("mutation without csrf = %d %s, want 403 CSRF_INVALID", missing.Code, missing.Body.String())
	}

	valid := fixture.request(http.MethodPost, "/api/v1/private-mutation", "", handlerOrigin, handlerCSRF, handlerToken)
	if valid.Code != http.StatusNoContent {
		t.Fatalf("mutation with csrf = %d %s, want 204", valid.Code, valid.Body.String())
	}
}

func TestMobileLoginReturnsBearerTokenWithoutCookie(t *testing.T) {
	for name, origin := range map[string]string{"WebView origin": mobileHandlerOrigin, "native client without Origin": ""} {
		t.Run(name, func(t *testing.T) {
			fixture := newHandlerFixture(t, false)
			login := fixture.request(http.MethodPost, "/api/v1/auth/login", `{"email":"owner@example.com","password":"`+handlerPassword+`","client":"mobile"}`, origin, "", "")

			if login.Code != http.StatusOK || len(login.Result().Cookies()) != 0 {
				t.Fatalf("mobile login = %d %s, cookies = %v", login.Code, login.Body.String(), login.Result().Cookies())
			}
			data := mobileSessionPayload(t, login)
			if data.SessionToken != handlerToken || data.CSRFToken != "" || data.ExpiresAt == "" {
				t.Fatalf("mobile login data = %+v, want bearer token and no CSRF token", data)
			}
			if fixture.useCases.loginKind != SessionKindMobile {
				t.Fatalf("login opened a %q session, want MOBILE", fixture.useCases.loginKind)
			}
		})
	}
}

func TestMobileLoginRejectsOriginsOutsideTheMobileList(t *testing.T) {
	for _, origin := range []string{"https://attacker.example", handlerOrigin} {
		fixture := newHandlerFixture(t, false)
		res := fixture.request(http.MethodPost, "/api/v1/auth/login", `{"email":"owner@example.com","password":"`+handlerPassword+`","client":"mobile"}`, origin, "", "")
		if res.Code != http.StatusForbidden || responseErrorCode(t, res) != "ORIGIN_INVALID" {
			t.Fatalf("mobile login from %s = %d %s, want 403 ORIGIN_INVALID", origin, res.Code, res.Body.String())
		}
	}
}

func TestLoginRejectsUnknownClient(t *testing.T) {
	fixture := newHandlerFixture(t, false)
	res := fixture.request(http.MethodPost, "/api/v1/auth/login", `{"email":"owner@example.com","password":"`+handlerPassword+`","client":"desktop"}`, handlerOrigin, "", "")
	if res.Code != http.StatusBadRequest || responseErrorCode(t, res) != "INVALID_REQUEST" {
		t.Fatalf("unknown client login = %d %s, want 400", res.Code, res.Body.String())
	}
}

func TestBearerSessionReachesPrivateRoutesWithoutCSRF(t *testing.T) {
	fixture := newHandlerFixture(t, false)
	fixture.useCases.kind = SessionKindMobile
	fixture.router.POST("/api/v1/private-mutation", fixture.handler.RequireSession(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	if res := fixture.bearerRequest(http.MethodGet, "/api/v1/private", "", mobileHandlerOrigin, handlerToken); res.Code != http.StatusOK {
		t.Fatalf("bearer private GET = %d %s", res.Code, res.Body.String())
	}
	if res := fixture.bearerRequest(http.MethodPost, "/api/v1/private-mutation", "", mobileHandlerOrigin, handlerToken); res.Code != http.StatusNoContent {
		t.Fatalf("bearer mutation without CSRF = %d %s, want 204", res.Code, res.Body.String())
	}
	restore := fixture.bearerRequest(http.MethodGet, "/api/v1/auth/session", "", "", handlerToken)
	if restore.Code != http.StatusOK {
		t.Fatalf("bearer session restore = %d %s", restore.Code, restore.Body.String())
	}
	if data := mobileSessionPayload(t, restore); data.SessionToken != "" || data.CSRFToken != "" {
		t.Fatalf("bearer restore echoed secrets: %+v", data)
	}
}

func TestSessionTokenOnlyWorksThroughItsOwnTransport(t *testing.T) {
	web := newHandlerFixture(t, false)
	if res := web.bearerRequest(http.MethodGet, "/api/v1/private", "", "", handlerToken); res.Code != http.StatusUnauthorized || responseErrorCode(t, res) != "SESSION_INVALID" {
		t.Fatalf("web session used as bearer = %d %s, want 401", res.Code, res.Body.String())
	}

	mobile := newHandlerFixture(t, false)
	mobile.useCases.kind = SessionKindMobile
	if res := mobile.request(http.MethodGet, "/api/v1/private", "", "", "", handlerToken); res.Code != http.StatusUnauthorized || responseErrorCode(t, res) != "SESSION_INVALID" {
		t.Fatalf("mobile session used as cookie = %d %s, want 401", res.Code, res.Body.String())
	}
}

func TestCookieAndBearerTogetherAreRejected(t *testing.T) {
	fixture := newHandlerFixture(t, false)
	for _, path := range []string{"/api/v1/auth/session", "/api/v1/private"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: handlerToken})
		req.Header.Set("Authorization", "Bearer "+handlerToken)
		res := httptest.NewRecorder()
		fixture.router.ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest || responseErrorCode(t, res) != "AUTH_AMBIGUOUS_CREDENTIALS" {
			t.Fatalf("%s with cookie and bearer = %d %s, want 400", path, res.Code, res.Body.String())
		}
	}
}

func TestBearerFromUnlistedOriginIsRejected(t *testing.T) {
	fixture := newHandlerFixture(t, false)
	fixture.useCases.kind = SessionKindMobile
	res := fixture.bearerRequest(http.MethodGet, "/api/v1/private", "", "https://attacker.example", handlerToken)
	if res.Code != http.StatusForbidden || responseErrorCode(t, res) != "ORIGIN_INVALID" {
		t.Fatalf("bearer from foreign origin = %d %s, want 403 ORIGIN_INVALID", res.Code, res.Body.String())
	}
}

func TestBearerChangePasswordRotatesTokenAndLogoutRevokesIt(t *testing.T) {
	fixture := newHandlerFixture(t, true)
	fixture.useCases.kind = SessionKindMobile

	change := fixture.bearerRequest(http.MethodPost, "/api/v1/auth/change-password", `{"current_password":"`+handlerPassword+`","new_password":"New owner passphrase 10!"}`, mobileHandlerOrigin, handlerToken)
	if change.Code != http.StatusOK || len(change.Result().Cookies()) != 0 {
		t.Fatalf("bearer change-password = %d %s, cookies = %v", change.Code, change.Body.String(), change.Result().Cookies())
	}
	replacement := mobileSessionPayload(t, change).SessionToken
	if replacement == "" || replacement == handlerToken {
		t.Fatalf("change-password returned session_token %q, want a rotated token", replacement)
	}
	if old := fixture.bearerRequest(http.MethodGet, "/api/v1/auth/session", "", "", handlerToken); old.Code != http.StatusUnauthorized {
		t.Fatalf("old bearer survived rotation: %d", old.Code)
	}

	logout := fixture.bearerRequest(http.MethodPost, "/api/v1/auth/logout", "", mobileHandlerOrigin, replacement)
	if logout.Code != http.StatusOK {
		t.Fatalf("bearer logout = %d %s", logout.Code, logout.Body.String())
	}
	if revoked := fixture.bearerRequest(http.MethodGet, "/api/v1/auth/session", "", "", replacement); revoked.Code != http.StatusUnauthorized || responseErrorCode(t, revoked) != "SESSION_INVALID" {
		t.Fatalf("revoked bearer = %d %s, want 401 SESSION_INVALID", revoked.Code, revoked.Body.String())
	}
}

type handlerUseCases struct {
	invalid    bool
	mustChange bool
	token      string
	csrf       string
	kind       SessionKind
	loginKind  SessionKind
	loginErr   error
	principal  httpx.Principal
}

func (u *handlerUseCases) session() AuthSession {
	u.principal.PasswordChangeRequired = u.mustChange
	csrf := u.csrf
	if u.kind == SessionKindMobile {
		csrf = ""
	}
	return AuthSession{Token: u.token, CSRFToken: csrf, Kind: u.kind.normalized(), ExpiresAt: testNow.Add(8 * time.Hour), MustChangePassword: u.mustChange, Principal: u.principal}
}

func (u *handlerUseCases) Login(_ context.Context, _, _ string, _ string, kind SessionKind) (AuthSession, error) {
	if u.loginErr != nil {
		return AuthSession{}, u.loginErr
	}
	u.loginKind = kind
	u.kind = kind
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
	handler  *Handler
	useCases *handlerUseCases
}

func newHandlerFixture(t *testing.T, mustChange bool) *handlerFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	u := &handlerUseCases{token: handlerToken, csrf: handlerCSRF, mustChange: mustChange, principal: httpx.Principal{
		UserID: uuid.New(), WorkshopID: uuid.New(), Email: "owner@example.com", DisplayName: "Owner", Role: "OWNER",
	}}
	handler, err := NewHandler(u, HandlerConfig{AllowedOrigin: handlerOrigin, MobileOrigins: []string{mobileHandlerOrigin}, Environment: "production", Secure: true})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	RegisterRoutes(router, handler)
	router.GET("/api/v1/private", handler.RequireSession(), func(c *gin.Context) {
		principal, err := httpx.PrincipalFromGin(c)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"user_id": principal.UserID}})
	})
	return &handlerFixture{router: router, handler: handler, useCases: u}
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

func (f *handlerFixture) bearerRequest(method, path, body, origin, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	f.router.ServeHTTP(res, req)
	return res
}

type mobileSessionData struct {
	ExpiresAt          string `json:"expires_at"`
	SessionToken       string `json:"session_token"`
	CSRFToken          string `json:"csrf_token"`
	MustChangePassword bool   `json:"must_change_password"`
}

func mobileSessionPayload(t *testing.T, res *httptest.ResponseRecorder) mobileSessionData {
	t.Helper()
	var payload struct {
		Data mobileSessionData `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Data
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
