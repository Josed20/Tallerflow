package team

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	db     *gorm.DB
	tenant platformdb.TenantRunner
}

func NewPostgresRepository(db *gorm.DB, tenant platformdb.TenantRunner) (*PostgresRepository, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	if tenant == nil {
		return nil, errors.New("tenant runner is required")
	}
	return &PostgresRepository{db: db, tenant: tenant}, nil
}

func (r *PostgresRepository) ListMembers(ctx context.Context, workshopID uuid.UUID) ([]Member, error) {
	var members []Member
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		return tx.Raw(`
			SELECT wm.id, wm.user_id, u.email::text AS email, u.name AS display_name,
			       wm.role, wm.status, wm.created_at, wm.updated_at
			FROM workshop_members AS wm
			JOIN users AS u ON u.id = wm.user_id
			WHERE wm.workshop_id = ?
			ORDER BY
			  CASE wm.role WHEN 'OWNER' THEN 1 WHEN 'ADMIN' THEN 2 ELSE 3 END,
			  lower(u.name), lower(u.email)
		`, workshopID).Scan(&members).Error
	})
	return members, err
}

func (r *PostgresRepository) ListInvitations(ctx context.Context, workshopID uuid.UUID, now time.Time) ([]Invitation, error) {
	var invitations []Invitation
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		return tx.Raw(`
			SELECT id, email::text AS email, role, expires_at, created_at
			FROM team_invitations
			WHERE workshop_id = ? AND consumed_at IS NULL AND canceled_at IS NULL AND expires_at > ?
			ORDER BY created_at DESC
		`, workshopID, now).Scan(&invitations).Error
	})
	return invitations, err
}

func (r *PostgresRepository) RegenerateInvitation(ctx context.Context, workshopID, actorUserID, invitationID uuid.UUID, tokenHash []byte, expiresAt, now time.Time) (Invitation, error) {
	var invitation Invitation
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		result := tx.Raw(`
			UPDATE team_invitations
			SET token_hash = ?, expires_at = ?, updated_at = ?
			WHERE id = ? AND workshop_id = ? AND consumed_at IS NULL AND canceled_at IS NULL AND expires_at > ?
			RETURNING id, email::text AS email, role, expires_at, created_at
		`, tokenHash, expiresAt, now, invitationID, workshopID, now).Scan(&invitation)
		if result.Error != nil {
			return mapConstraint(result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrInvitationUnavailable
		}
		return tx.Exec(`
			INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details)
			VALUES (?, ?, 'TEAM_INVITATION_REGENERATED', jsonb_build_object('invitation_id', ?::text, 'email', ?::text))
		`, workshopID, actorUserID, invitationID, invitation.Email).Error
	})
	if err != nil {
		return Invitation{}, err
	}
	return invitation, nil
}

func (r *PostgresRepository) CancelInvitation(ctx context.Context, workshopID, actorUserID, invitationID uuid.UUID, now time.Time) error {
	return r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var invitation struct{ Email string }
		result := tx.Raw(`
			UPDATE team_invitations
			SET canceled_at = ?, updated_at = ?
			WHERE id = ? AND workshop_id = ? AND consumed_at IS NULL AND canceled_at IS NULL
			RETURNING email::text AS email
		`, now, now, invitationID, workshopID).Scan(&invitation)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvitationUnavailable
		}
		return tx.Exec(`
			INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details)
			VALUES (?, ?, 'TEAM_INVITATION_CANCELED', jsonb_build_object('invitation_id', ?::text, 'email', ?::text))
		`, workshopID, actorUserID, invitationID, invitation.Email).Error
	})
}

func (r *PostgresRepository) CreateInvitation(ctx context.Context, workshopID, actorUserID uuid.UUID, email, role string, tokenHash []byte, expiresAt time.Time) (Invitation, error) {
	var invitation Invitation
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		// Expired links are no longer actionable; retire them so a new invitation
		// for the same email is never blocked by stale state.
		if err := tx.Exec(`
			UPDATE team_invitations
			SET canceled_at = ?, updated_at = ?
			WHERE workshop_id = ? AND email = ? AND consumed_at IS NULL AND canceled_at IS NULL AND expires_at <= ?
		`, expiresAt.Add(-invitationLifetime), expiresAt.Add(-invitationLifetime), workshopID, email, expiresAt.Add(-invitationLifetime)).Error; err != nil {
			return err
		}
		var existing struct {
			UserID     uuid.UUID
			WorkshopID uuid.UUID
		}
		lookup := tx.Raw(`
			SELECT u.id AS user_id, m.workshop_id
			FROM users AS u
			LEFT JOIN LATERAL resolve_active_memberships(u.id) AS m ON true
			WHERE u.email = ?
			LIMIT 1
		`, email).Scan(&existing)
		if lookup.Error != nil {
			return lookup.Error
		}
		if existing.UserID != uuid.Nil && existing.WorkshopID != uuid.Nil && existing.WorkshopID != workshopID {
			return ErrCrossWorkshopUser
		}
		result := tx.Raw(`
			INSERT INTO team_invitations (workshop_id, email, role, token_hash, created_by, expires_at)
			VALUES (?, ?, ?, ?, ?, ?)
			RETURNING id, email::text AS email, role, expires_at, created_at
		`, workshopID, email, role, tokenHash, actorUserID, expiresAt).Scan(&invitation)
		if result.Error != nil {
			return mapConstraint(result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrInvalidInput
		}
		return nil
	})
	if err != nil {
		return Invitation{}, err
	}
	return invitation, nil
}

func (r *PostgresRepository) ConsumeInvitation(ctx context.Context, tokenHash []byte, name, passwordHash string, now time.Time) (ConsumeResult, error) {
	var result ConsumeResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT set_config('app.invitation_token_hash', ?, true)", hex.EncodeToString(tokenHash)).Error; err != nil {
			return fmt.Errorf("set invitation token context: %w", err)
		}
		var invitation struct {
			ID         uuid.UUID
			WorkshopID uuid.UUID
			Email      string
			Role       string
			ExpiresAt  time.Time
		}
		find := tx.Raw(`
			SELECT id, workshop_id, email::text AS email, role, expires_at
			FROM team_invitations
			WHERE token_hash = ? AND consumed_at IS NULL AND canceled_at IS NULL
			FOR UPDATE
		`, tokenHash).Scan(&invitation)
		if find.Error != nil {
			return find.Error
		}
		if find.RowsAffected != 1 || !invitation.ExpiresAt.After(now) {
			return ErrInvitationUnavailable
		}
		if err := tx.Exec("SELECT set_config('app.workshop_id', ?, true)", invitation.WorkshopID.String()).Error; err != nil {
			return fmt.Errorf("set invitation tenant context: %w", err)
		}

		var existing struct {
			UserID     uuid.UUID
			WorkshopID uuid.UUID
			Status     string
		}
		userLookup := tx.Raw(`
			SELECT u.id AS user_id, wm.workshop_id, wm.status
			FROM users AS u
			LEFT JOIN workshop_members AS wm ON wm.user_id = u.id
			WHERE u.email = ?
			FOR UPDATE OF u
		`, invitation.Email).Scan(&existing)
		if userLookup.Error != nil {
			return userLookup.Error
		}

		userID := existing.UserID
		if userID != uuid.Nil && existing.WorkshopID != uuid.Nil && existing.WorkshopID != invitation.WorkshopID {
			return ErrCrossWorkshopUser
		}
		if userID == uuid.Nil {
			var returnedUserID string
			if err := tx.Raw(`
				INSERT INTO users (email, name, status)
				VALUES (?, ?, 'ACTIVE')
				RETURNING id
			`, invitation.Email, name).Scan(&returnedUserID).Error; err != nil {
				return mapConstraint(err)
			}
			parsedUserID, err := uuid.Parse(returnedUserID)
			if err != nil {
				return fmt.Errorf("parse created user id: %w", err)
			}
			userID = parsedUserID
			if err := tx.Exec(`
				INSERT INTO user_credentials (user_id, password_hash, must_change_password, password_changed_at)
				VALUES (?, ?, false, ?)
			`, userID, passwordHash, now).Error; err != nil {
				return mapConstraint(err)
			}
			if err := tx.Exec(`
				INSERT INTO workshop_members (workshop_id, user_id, role, status)
				VALUES (?, ?, ?, 'ACTIVE')
			`, invitation.WorkshopID, userID, invitation.Role).Error; err != nil {
				return mapConstraint(err)
			}
		} else if existing.WorkshopID == uuid.Nil {
			if err := tx.Exec(`
				INSERT INTO workshop_members (workshop_id, user_id, role, status)
				VALUES (?, ?, ?, 'ACTIVE')
			`, invitation.WorkshopID, userID, invitation.Role).Error; err != nil {
				return mapConstraint(err)
			}
		} else if existing.Status == "INACTIVE" {
			if err := tx.Exec(`
				UPDATE workshop_members
				SET role = ?, status = 'ACTIVE', updated_at = ?
				WHERE workshop_id = ? AND user_id = ?
			`, invitation.Role, now, invitation.WorkshopID, userID).Error; err != nil {
				return mapConstraint(err)
			}
		}

		update := tx.Exec(`
			UPDATE team_invitations
			SET consumed_at = ?, consumed_by = ?, updated_at = ?
			WHERE id = ? AND consumed_at IS NULL
		`, now, userID, now, invitation.ID)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrInvitationUnavailable
		}
		if err := tx.Exec(`
			INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details)
			VALUES (?, ?, 'TEAM_INVITATION_CONSUMED', jsonb_build_object('email', ?::text, 'role', ?::text))
		`, invitation.WorkshopID, userID, invitation.Email, invitation.Role).Error; err != nil {
			return err
		}
		result = ConsumeResult{Email: invitation.Email, Role: invitation.Role}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrCrossWorkshopUser) {
			return ConsumeResult{}, ErrInvitationUnavailable
		}
		return ConsumeResult{}, err
	}
	return result, nil
}

func (r *PostgresRepository) UpdateMember(ctx context.Context, workshopID, actorUserID, membershipID uuid.UUID, input UpdateMemberInput) (Member, error) {
	var member Member
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var current struct {
			UserID uuid.UUID
			Role   string
			Status string
		}
		found := tx.Raw(`
			SELECT user_id, role, status
			FROM workshop_members
			WHERE id = ? AND workshop_id = ?
			FOR UPDATE
		`, membershipID, workshopID).Scan(&current)
		if found.Error != nil {
			return found.Error
		}
		if found.RowsAffected != 1 {
			return ErrMemberNotFound
		}
		nextRole := current.Role
		nextStatus := current.Status
		if input.Role != nil {
			nextRole = *input.Role
		}
		if input.Status != nil {
			nextStatus = *input.Status
		}
		if current.Role == "OWNER" && (nextRole != "OWNER" || nextStatus != "ACTIVE") {
			var ownerLocks []uuid.UUID
			if err := tx.Raw(`
				SELECT id
				FROM workshop_members
				WHERE workshop_id = ? AND role = 'OWNER' AND status = 'ACTIVE'
				FOR UPDATE
			`, workshopID).Scan(&ownerLocks).Error; err != nil {
				return err
			}
			if len(ownerLocks) <= 1 {
				return ErrLastOwner
			}
		}
		update := tx.Exec(`
			UPDATE workshop_members
			SET role = ?, status = ?, updated_at = ?
			WHERE id = ? AND workshop_id = ?
		`, nextRole, nextStatus, time.Now().UTC(), membershipID, workshopID)
		if update.Error != nil {
			return mapConstraint(update.Error)
		}
		if update.RowsAffected != 1 {
			return ErrMemberNotFound
		}
		if err := tx.Raw(`
			SELECT wm.id, wm.user_id, u.email::text AS email, u.name AS display_name,
			       wm.role, wm.status, wm.created_at, wm.updated_at
			FROM workshop_members AS wm
			JOIN users AS u ON u.id = wm.user_id
			WHERE wm.id = ?
		`, membershipID).Scan(&member).Error; err != nil {
			return err
		}
		return tx.Exec(`
			INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details)
			VALUES (?, ?, 'TEAM_MEMBER_UPDATED', jsonb_build_object('membership_id', ?::text, 'role', ?::text, 'status', ?::text))
		`, workshopID, actorUserID, membershipID, nextRole, nextStatus).Error
	})
	if err != nil {
		return Member{}, err
	}
	return member, nil
}

func mapConstraint(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "team_invitations_active_email"):
			return ErrDuplicateInvitation
		case pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "workshop_members_one_workshop"):
			return ErrCrossWorkshopUser
		case pgErr.Code == "23514":
			return ErrInvalidInput
		}
	}
	return fmt.Errorf("team persistence: %w", err)
}
