package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
)

// DeriveCSRFToken recreates the browser's CSRF value from its opaque session
// token. The purpose label prevents a session-token digest from being reused as
// a CSRF token even when the caller supplies the same server secret.
func DeriveCSRFToken(secret []byte, rawSessionToken string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte("tallerflow/csrf/v1\x00"))
	_, _ = mac.Write([]byte(rawSessionToken))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func ValidCSRFToken(expected, submitted string) bool {
	if len(expected) == 0 || len(submitted) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(submitted)) == 1
}

// ValidRequestOrigin requires an exact scheme/host/port match. Referer is a
// fallback for clients that omit Origin; malformed or opaque values fail shut.
func ValidRequestOrigin(request *http.Request, allowedOrigin string) bool {
	allowed, err := url.Parse(allowedOrigin)
	if err != nil || !validOriginURL(allowed) {
		return false
	}
	value := strings.TrimSpace(request.Header.Get("Origin"))
	fromOrigin := value != ""
	if value == "" {
		value = strings.TrimSpace(request.Header.Get("Referer"))
	}
	if value == "" || value == "null" {
		return false
	}
	actual, err := url.Parse(value)
	if err != nil || !validOriginURL(actual) {
		return false
	}
	if fromOrigin && (actual.Path != "" || actual.RawQuery != "" || actual.Fragment != "") {
		return false
	}
	return strings.EqualFold(actual.Scheme, allowed.Scheme) && strings.EqualFold(actual.Host, allowed.Host)
}

func validOriginURL(value *url.URL) bool {
	return value != nil && (value.Scheme == "http" || value.Scheme == "https") && value.Host != "" && value.User == nil && value.Opaque == ""
}

// RequireCSRF belongs after authentication middleware, which supplies the
// expected token from the current session. Protected mutation routes can use
// it without duplicating origin and constant-time token validation.
func RequireCSRF(allowedOrigin string, expectedToken func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		valid := expectedToken != nil && ValidRequestOrigin(c.Request, allowedOrigin) &&
			ValidCSRFToken(expectedToken(c), c.GetHeader("X-CSRF-Token"))
		if !valid {
			httpx.RespondError(c, http.StatusForbidden, "CSRF_INVALID", "CSRF validation failed.")
			c.Abort()
			return
		}
		c.Next()
	}
}
