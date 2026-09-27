package onboarding

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type statusQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PostgresStatusStore struct {
	pool statusQueryer
}

func NewPostgresStatusStore(pool *pgxpool.Pool) *PostgresStatusStore {
	return &PostgresStatusStore{pool: pool}
}

func (s *PostgresStatusStore) Claimed(ctx context.Context) (bool, error) {
	if s == nil || s.pool == nil {
		return false, errors.New("onboarding status database is not configured")
	}
	var claimed bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM bootstrap_state WHERE singleton = true)`).Scan(&claimed); err != nil {
		return false, fmt.Errorf("read onboarding status: %w", err)
	}
	return claimed, nil
}
