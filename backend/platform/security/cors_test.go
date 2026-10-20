package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

const mobileOrigin = "https://localhost"

func newCORSRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MobileCORS([]string{mobileOrigin, "capacitor://localhost"}))
	router.POST("/api/v1/orders", func(c *gin.Context) { c.Status(http.StatusCreated) })
	return router
}

func corsRequest(router *gin.Engine, method, origin, requestMethod string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/api/v1/orders", nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if requestMethod != "" {
		request.Header.Set("Access-Control-Request-Method", requestMethod)
		request.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestMobileCORSAnswersPreflightForListedOriginWithoutCredentials(t *testing.T) {
	response := corsRequest(newCORSRouter(), http.MethodOptions, mobileOrigin, http.MethodPost)

	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != mobileOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, mobileOrigin)
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("preflight does not allow the Authorization header")
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Access-Control-Allow-Credentials = %q; mobile CORS must never allow cookies", got)
	}
	if got := response.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want Origin", got)
	}
}

func TestMobileCORSDeniesUnlistedOrigin(t *testing.T) {
	router := newCORSRouter()

	preflight := corsRequest(router, http.MethodOptions, "https://attacker.example", http.MethodPost)
	if preflight.Code != http.StatusForbidden || preflight.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unlisted preflight = %d with ACAO %q, want 403 without CORS headers", preflight.Code, preflight.Header().Get("Access-Control-Allow-Origin"))
	}
	simple := corsRequest(router, http.MethodPost, "https://attacker.example", "")
	if simple.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unlisted origin received Access-Control-Allow-Origin")
	}
}

func TestMobileCORSLeavesSameOriginWebRequestsUntouched(t *testing.T) {
	response := corsRequest(newCORSRouter(), http.MethodPost, "https://tallerflowpe.com", "")

	if response.Code != http.StatusCreated {
		t.Fatalf("same-origin web request status = %d, want handler's 201", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("web request received CORS headers")
	}
}

func TestMobileCORSExposesHeadersOnListedOriginRequest(t *testing.T) {
	response := corsRequest(newCORSRouter(), http.MethodPost, mobileOrigin, "")

	if response.Code != http.StatusCreated || response.Header().Get("Access-Control-Allow-Origin") != mobileOrigin {
		t.Fatalf("mobile request = %d with ACAO %q", response.Code, response.Header().Get("Access-Control-Allow-Origin"))
	}
	if response.Header().Get("Access-Control-Expose-Headers") == "" {
		t.Fatal("mobile request cannot read X-Request-ID or Retry-After")
	}
}

func TestMobileOriginAllowedMatchesExactOriginsOnly(t *testing.T) {
	allowed := []string{mobileOrigin}
	for origin, want := range map[string]bool{
		"https://localhost":      true,
		"HTTPS://LOCALHOST":      true,
		"https://localhost:8443": false,
		"https://localhost.evil": false,
		"http://localhost":       false,
		"":                       false,
		"null":                   false,
		"https://localhost/path": false,
	} {
		if got := MobileOriginAllowed(origin, allowed); got != want {
			t.Errorf("MobileOriginAllowed(%q) = %t, want %t", origin, got, want)
		}
	}
}
