package httpx

import (
	"context"
	"errors"
	"net/http"

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

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, error) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	if !ok || !principal.Valid() {
		return Principal{}, ErrPrincipalMissing
	}
	return principal, nil
}

func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		if validRole(role) {
			allowed[role] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := PrincipalFromContext(r.Context())
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			if _, ok := allowed[principal.Role]; !ok {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
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
