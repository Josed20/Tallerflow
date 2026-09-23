package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresBootstrapStore performs bootstrap writes with the administrative
// connection used only by the short-lived bootstrap command.
type PostgresBootstrapStore struct {
	pool *pgxpool.Pool
}

func NewPostgresBootstrapStore(pool *pgxpool.Pool) *PostgresBootstrapStore {
	return &PostgresBootstrapStore{pool: pool}
}

func (s *PostgresBootstrapStore) WithinTransaction(ctx context.Context, fn func(BootstrapTx) error) error {
	if s == nil || s.pool == nil {
		return errors.New("bootstrap database is not configured")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(postgresBootstrapTx{tx: tx}); err != nil {
		return translateBootstrapError(err)
	}
	return translateBootstrapError(tx.Commit(ctx))
}

type postgresBootstrapTx struct {
	tx pgx.Tx
}

func (tx postgresBootstrapTx) InsertUser(ctx context.Context, email, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.tx.QueryRow(ctx, `INSERT INTO users (email, name, status) VALUES ($1, $2, 'ACTIVE') RETURNING id`, email, name).Scan(&id)
	return id, err
}

func (tx postgresBootstrapTx) InsertCredential(ctx context.Context, userID uuid.UUID, passwordHash string, mustChange bool) error {
	_, err := tx.tx.Exec(ctx, `INSERT INTO user_credentials (user_id, password_hash, must_change_password) VALUES ($1, $2, $3)`, userID, passwordHash, mustChange)
	return err
}

func (tx postgresBootstrapTx) InsertWorkshop(ctx context.Context, name, timezone string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.tx.QueryRow(ctx, `INSERT INTO workshops (name, timezone) VALUES ($1, $2) RETURNING id`, name, timezone).Scan(&id)
	return id, err
}

func (tx postgresBootstrapTx) InsertMembership(ctx context.Context, userID, workshopID uuid.UUID, role string) error {
	_, err := tx.tx.Exec(ctx, `INSERT INTO workshop_members (user_id, workshop_id, role, status) VALUES ($1, $2, $3, 'ACTIVE')`, userID, workshopID, role)
	return err
}

func (tx postgresBootstrapTx) InsertAudit(ctx context.Context, event string, userID, workshopID uuid.UUID) error {
	_, err := tx.tx.Exec(ctx, `INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details) VALUES ($1, $2, $3, '{}'::jsonb)`, workshopID, userID, event)
	return err
}

func translateBootstrapError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return ErrBootstrapAlreadyExists
	}
	return err
}
