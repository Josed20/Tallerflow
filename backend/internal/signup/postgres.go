package signup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// PostgresRepository writes through the runtime role. A new workshop's id is
// chosen before the transaction so app.workshop_id can be set to it, and the
// existing tenant policies accept the inserts (see V11). users and
// email_verification_tokens carry no tenant policy.
type PostgresRepository struct {
	db      *gorm.DB
	tenants database.TenantRunner
}

func NewPostgresRepository(db *gorm.DB, tenants database.TenantRunner) (*PostgresRepository, error) {
	if db == nil || tenants == nil {
		return nil, errors.New("database and tenant runner are required")
	}
	return &PostgresRepository{db: db, tenants: tenants}, nil
}

type accountRow struct {
	ID       uuid.UUID
	Name     string
	Verified bool
}

func (r *PostgresRepository) findAccount(ctx context.Context, email string) (accountRow, bool, error) {
	var row accountRow
	result := r.db.WithContext(ctx).Raw(`SELECT id, name, email_verified_at IS NOT NULL AS verified FROM users WHERE email = ?`, email).Scan(&row)
	if result.Error != nil {
		return accountRow{}, false, fmt.Errorf("find signup account: %w", result.Error)
	}
	return row, result.RowsAffected == 1, nil
}

// workshopOf resolves the owner's workshop before a tenant context exists,
// through the same SECURITY DEFINER function the login uses.
func (r *PostgresRepository) workshopOf(ctx context.Context, userID uuid.UUID) (uuid.UUID, bool, error) {
	var memberships []struct{ WorkshopID uuid.UUID }
	if err := r.db.WithContext(ctx).Raw(`SELECT workshop_id FROM resolve_active_memberships(?)`, userID).Scan(&memberships).Error; err != nil {
		return uuid.Nil, false, fmt.Errorf("resolve signup workshop: %w", err)
	}
	if len(memberships) != 1 {
		return uuid.Nil, false, nil
	}
	return memberships[0].WorkshopID, true, nil
}

func (r *PostgresRepository) Register(ctx context.Context, account PendingAccount) (bool, error) {
	existing, found, err := r.findAccount(ctx, account.Email)
	if err != nil {
		return false, err
	}
	if found {
		if existing.Verified {
			return false, nil
		}
		return r.replacePending(ctx, existing.ID, account)
	}

	workshopID := uuid.New()
	err = r.tenants.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var returnedID string
		if err := tx.Raw(`INSERT INTO users (email, name, status, terms_accepted_at) VALUES (?, ?, 'ACTIVE', now()) RETURNING id`,
			account.Email, account.OwnerName).Scan(&returnedID).Error; err != nil {
			return err
		}
		userID, err := uuid.Parse(returnedID)
		if err != nil {
			return fmt.Errorf("parse created user id: %w", err)
		}
		if err := tx.Exec(`INSERT INTO user_credentials (user_id, password_hash, must_change_password, password_changed_at) VALUES (?, ?, false, now())`,
			userID, account.PasswordHash).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO workshops (id, name, timezone) VALUES (?, ?, 'America/Lima')`, workshopID, account.WorkshopName).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO workshop_members (workshop_id, user_id, role, status) VALUES (?, ?, 'OWNER', 'ACTIVE')`, workshopID, userID).Error; err != nil {
			return err
		}
		if err := insertToken(tx, userID, account.TokenHash, account.ExpiresAt); err != nil {
			return err
		}
		return insertAudit(tx, workshopID, userID, "WORKSHOP_SIGNED_UP")
	})
	if isUniqueViolation(err) {
		// Another request registered the same address first; answer as if it
		// had been there all along.
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("create signup account: %w", err)
	}
	return true, nil
}

// replacePending lets a second registration of a never-verified address win,
// so someone who signed up with another person's email cannot block the
// mailbox owner. It cannot hand the account to whoever registered last: the
// emailed link only activates it for someone who also knows that password.
func (r *PostgresRepository) replacePending(ctx context.Context, userID uuid.UUID, account PendingAccount) (bool, error) {
	workshopID, ok, err := r.workshopOf(ctx, userID)
	if err != nil || !ok {
		return false, err
	}
	issued := false
	err = r.tenants.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var stillPending bool
		result := tx.Raw(`SELECT email_verified_at IS NULL FROM users WHERE id = ? FOR UPDATE`, userID).Scan(&stillPending)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 || !stillPending {
			return nil
		}
		if err := tx.Exec(`UPDATE users SET name = ?, terms_accepted_at = now(), updated_at = now() WHERE id = ?`, account.OwnerName, userID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE user_credentials SET password_hash = ?, password_changed_at = now(), updated_at = now() WHERE user_id = ?`,
			account.PasswordHash, userID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE workshops SET name = ?, updated_at = now() WHERE id = ?`, account.WorkshopName, workshopID).Error; err != nil {
			return err
		}
		if err := replaceToken(tx, userID, account.TokenHash, account.ExpiresAt); err != nil {
			return err
		}
		issued = true
		return insertAudit(tx, workshopID, userID, "WORKSHOP_SIGNUP_RESUBMITTED")
	})
	if err != nil {
		return false, fmt.Errorf("replace pending signup: %w", err)
	}
	return issued, nil
}

func (r *PostgresRepository) ReissueToken(ctx context.Context, email string, tokenHash []byte, expiresAt time.Time) (Recipient, bool, error) {
	existing, found, err := r.findAccount(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || !found || existing.Verified {
		return Recipient{}, false, err
	}
	issued := false
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stillPending bool
		result := tx.Raw(`SELECT email_verified_at IS NULL FROM users WHERE id = ? FOR UPDATE`, existing.ID).Scan(&stillPending)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 || !stillPending {
			return nil
		}
		if err := replaceToken(tx, existing.ID, tokenHash, expiresAt); err != nil {
			return err
		}
		issued = true
		return nil
	})
	if err != nil {
		return Recipient{}, false, fmt.Errorf("reissue verification token: %w", err)
	}
	return Recipient{Name: existing.Name}, issued, nil
}

func (r *PostgresRepository) PendingPasswordHash(ctx context.Context, tokenHash []byte) (string, error) {
	var owner struct{ UserID uuid.UUID }
	result := r.db.WithContext(ctx).Raw(`SELECT user_id FROM email_verification_tokens WHERE token_hash = ?`, tokenHash).Scan(&owner)
	if result.Error != nil {
		return "", fmt.Errorf("find verification token: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return "", ErrTokenInvalid
	}
	workshopID, ok, err := r.workshopOf(ctx, owner.UserID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrTokenInvalid
	}
	var pending struct{ PasswordHash string }
	err = r.tenants.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		result := tx.Raw(`SELECT c.password_hash
			FROM email_verification_tokens t
			JOIN user_credentials c ON c.user_id = t.user_id
			WHERE t.token_hash = ? AND t.used_at IS NULL AND t.expires_at > now()`, tokenHash).Scan(&pending)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrTokenInvalid
		}
		return nil
	})
	if errors.Is(err, ErrTokenInvalid) {
		return "", ErrTokenInvalid
	}
	if err != nil {
		return "", fmt.Errorf("read pending password: %w", err)
	}
	return pending.PasswordHash, nil
}

func (r *PostgresRepository) ConsumeToken(ctx context.Context, tokenHash []byte) (uuid.UUID, error) {
	var owner struct{ UserID uuid.UUID }
	result := r.db.WithContext(ctx).Raw(`SELECT user_id FROM email_verification_tokens WHERE token_hash = ?`, tokenHash).Scan(&owner)
	if result.Error != nil {
		return uuid.Nil, fmt.Errorf("find verification token: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return uuid.Nil, ErrTokenInvalid
	}
	workshopID, ok, err := r.workshopOf(ctx, owner.UserID)
	if err != nil {
		return uuid.Nil, err
	}
	if !ok {
		return uuid.Nil, ErrTokenInvalid
	}
	err = r.tenants.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var token struct {
			ID     uuid.UUID
			Usable bool
		}
		result := tx.Raw(`SELECT id, used_at IS NULL AND expires_at > now() AS usable FROM email_verification_tokens WHERE token_hash = ? FOR UPDATE`, tokenHash).Scan(&token)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 || !token.Usable {
			return ErrTokenInvalid
		}
		if err := tx.Exec(`UPDATE email_verification_tokens SET used_at = now() WHERE id = ?`, token.ID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE users SET email_verified_at = now(), updated_at = now() WHERE id = ? AND email_verified_at IS NULL`, owner.UserID).Error; err != nil {
			return err
		}
		return insertAudit(tx, workshopID, owner.UserID, "EMAIL_VERIFIED")
	})
	if errors.Is(err, ErrTokenInvalid) {
		return uuid.Nil, ErrTokenInvalid
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("consume verification token: %w", err)
	}
	return owner.UserID, nil
}

func insertToken(tx *gorm.DB, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) error {
	return tx.Exec(`INSERT INTO email_verification_tokens (user_id, token_hash, expires_at) VALUES (?, ?, ?)`, userID, tokenHash, expiresAt.UTC()).Error
}

// replaceToken keeps at most one working link per user, as the partial unique
// index in V11 requires.
func replaceToken(tx *gorm.DB, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) error {
	if err := tx.Exec(`DELETE FROM email_verification_tokens WHERE user_id = ? AND used_at IS NULL`, userID).Error; err != nil {
		return err
	}
	return insertToken(tx, userID, tokenHash, expiresAt)
}

func insertAudit(tx *gorm.DB, workshopID, userID uuid.UUID, event string) error {
	return tx.Exec(`INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details) VALUES (?, ?, ?, '{}'::jsonb)`, workshopID, userID, event).Error
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" || errors.Is(err, gorm.ErrDuplicatedKey)
}
