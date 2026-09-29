package team

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/google/uuid"
)

const invitationLifetime = 7 * 24 * time.Hour

type Service struct {
	store  Store
	hasher auth.PasswordHasher
	pepper []byte
	origin string
	clock  func() time.Time
	random io.Reader
}

func NewService(store Store, hasher auth.PasswordHasher, pepper []byte, origin string, clock func() time.Time, random io.Reader) (*Service, error) {
	if store == nil {
		return nil, errors.New("team store is required")
	}
	if len(pepper) == 0 {
		return nil, errors.New("team token pepper is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("team origin is invalid")
	}
	if clock == nil {
		clock = time.Now
	}
	if random == nil {
		random = rand.Reader
	}
	return &Service{store: store, hasher: hasher, pepper: append([]byte(nil), pepper...), origin: strings.TrimRight(origin, "/"), clock: clock, random: random}, nil
}

func (s *Service) List(ctx context.Context, principal httpx.Principal) ([]Member, []Invitation, error) {
	if !canManageTeam(principal.Role) {
		return nil, nil, ErrForbidden
	}
	members, err := s.store.ListMembers(ctx, principal.WorkshopID)
	if err != nil {
		return nil, nil, err
	}
	invitations, err := s.store.ListInvitations(ctx, principal.WorkshopID, s.clock().UTC())
	if err != nil {
		return nil, nil, err
	}
	return members, invitations, nil
}

func (s *Service) Invite(ctx context.Context, principal httpx.Principal, input InviteInput) (InvitationCreated, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	role := strings.ToUpper(strings.TrimSpace(input.Role))
	if !validEmail(email) || !canInvite(principal.Role, role) {
		return InvitationCreated{}, ErrInvalidInput
	}
	token, err := newInvitationToken(s.random)
	if err != nil {
		return InvitationCreated{}, err
	}
	hash := invitationDigest(s.pepper, token)
	invitation, err := s.store.CreateInvitation(ctx, principal.WorkshopID, principal.UserID, email, role, encodeDigest(hash), s.clock().UTC().Add(invitationLifetime))
	if err != nil {
		return InvitationCreated{}, err
	}
	return InvitationCreated{
		Invitation: invitation,
		Token:      token,
		JoinURL:    s.origin + "/join?token=" + url.QueryEscape(publicToken(token)),
	}, nil
}

func (s *Service) Consume(ctx context.Context, input ConsumeInput) (ConsumeResult, error) {
	tokenBytes, err := decodePublicToken(strings.TrimSpace(input.Token))
	if err != nil || len(tokenBytes) == 0 {
		return ConsumeResult{}, ErrInvitationUnavailable
	}
	if strings.TrimSpace(input.Name) == "" || len(input.Password) < 12 || len(input.Password) > 1<<20 {
		return ConsumeResult{}, ErrInvalidInput
	}
	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return ConsumeResult{}, err
	}
	hash := invitationDigest(s.pepper, string(tokenBytes))
	return s.store.ConsumeInvitation(ctx, encodeDigest(hash), strings.TrimSpace(input.Name), passwordHash, s.clock().UTC())
}

func (s *Service) UpdateMember(ctx context.Context, principal httpx.Principal, membershipID uuid.UUID, input UpdateMemberInput) (Member, error) {
	if principal.Role != "OWNER" {
		return Member{}, ErrForbidden
	}
	if membershipID == uuid.Nil {
		return Member{}, ErrInvalidInput
	}
	if input.Role != nil {
		role := strings.ToUpper(strings.TrimSpace(*input.Role))
		if role != "OWNER" && role != "ADMIN" && role != "OPERATOR" {
			return Member{}, ErrInvalidInput
		}
		input.Role = &role
	}
	if input.Status != nil {
		status := strings.ToUpper(strings.TrimSpace(*input.Status))
		if status != "ACTIVE" && status != "INACTIVE" {
			return Member{}, ErrInvalidInput
		}
		input.Status = &status
	}
	if input.Role == nil && input.Status == nil {
		return Member{}, ErrInvalidInput
	}
	return s.store.UpdateMember(ctx, principal.WorkshopID, principal.UserID, membershipID, input)
}

func canManageTeam(role string) bool {
	return role == "OWNER" || role == "ADMIN"
}

func canInvite(actorRole, invitedRole string) bool {
	switch actorRole {
	case "OWNER":
		return invitedRole == "ADMIN" || invitedRole == "OPERATOR"
	case "ADMIN":
		return invitedRole == "OPERATOR"
	default:
		return false
	}
}

func validEmail(value string) bool {
	if len(value) == 0 || len(value) > 254 {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func decodePublicToken(value string) ([]byte, error) {
	unescaped, err := url.QueryUnescape(value)
	if err != nil {
		return nil, err
	}
	return base64.RawURLEncoding.DecodeString(unescaped)
}
