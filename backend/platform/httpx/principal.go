package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var ErrPrincipalMissing = errors.New("authenticated principal is missing")

type Principal struct {
	UserID                 uuid.UUID `json:"userId"`
	Email                  string    `json:"email"`
	DisplayName            string    `json:"displayName"`
	WorkshopID             uuid.UUID `json:"workshopId"`
	Role                   string    `json:"role"`
	PasswordChangeRequired bool      `json:"passwordChangeRequired"`
}

func (p Principal) Valid() bool {
	return p.UserID != uuid.Nil && p.WorkshopID != uuid.Nil && validRole(p.Role)
}

const principalKey = "authenticated_principal"

func SetPrincipal(c *gin.Context, principal Principal) {
	c.Set(principalKey, principal)
}

func PrincipalFromGin(c *gin.Context) (Principal, error) {
	value, ok := c.Get(principalKey)
	principal, typed := value.(Principal)
	if !ok || !typed || !principal.Valid() {
		return Principal{}, ErrPrincipalMissing
	}
	return principal, nil
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		if validRole(role) {
			allowed[role] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		principal, err := PrincipalFromGin(c)
		if err != nil {
			RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
			return
		}
		if _, ok := allowed[principal.Role]; !ok {
			RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot access this resource.")
			return
		}
		c.Next()
	}
}

func validRole(role string) bool {
	switch role {
	case "OWNER", "ADMIN", "SUPERVISOR", "OPERATOR":
		return true
	default:
		return false
	}
}
