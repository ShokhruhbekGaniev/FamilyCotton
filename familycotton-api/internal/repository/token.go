package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository struct {
	db *pgxpool.Pool
}

func NewTokenRepository(db *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (r *TokenRepository) ExistsByHash(ctx context.Context, tokenHash string) (bool, uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT user_id FROM refresh_tokens WHERE token_hash = $1 AND expires_at > NOW()`,
		tokenHash,
	).Scan(&userID)
	if err != nil {
		return false, uuid.Nil, nil
	}
	return true, userID, nil
}

func (r *TokenRepository) DeleteByHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *TokenRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}

func (r *TokenRepository) CleanExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < NOW()`)
	return err
}
