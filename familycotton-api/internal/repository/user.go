package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, name, login, password_hash, role)
		 VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Name, u.Login, u.PasswordHash, u.Role,
	)
	if err != nil {
		if isDuplicateKey(err) {
			return model.NewAppError(model.ErrConflict, "login already exists")
		}
		return err
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, login, password_hash, role, is_deleted, created_at, updated_at
		 FROM users WHERE id = $1 AND is_deleted = false`, id,
	).Scan(&u.ID, &u.Name, &u.Login, &u.PasswordHash, &u.Role, &u.IsDeleted, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "user not found")
	}
	return u, err
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, login, password_hash, role, is_deleted, created_at, updated_at
		 FROM users WHERE login = $1 AND is_deleted = false`, login,
	).Scan(&u.ID, &u.Name, &u.Login, &u.PasswordHash, &u.Role, &u.IsDeleted, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "user not found")
	}
	return u, err
}

func (r *UserRepository) List(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, login, password_hash, role, is_deleted, created_at, updated_at
		 FROM users WHERE is_deleted = false ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Login, &u.PasswordHash, &u.Role, &u.IsDeleted, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, u *model.User) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET name=$1, login=$2, password_hash=$3, role=$4, updated_at=NOW()
		 WHERE id=$5 AND is_deleted = false`,
		u.Name, u.Login, u.PasswordHash, u.Role, u.ID,
	)
	if err != nil && isDuplicateKey(err) {
		return model.NewAppError(model.ErrConflict, "login already exists")
	}
	return err
}

func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE users SET is_deleted = true, updated_at = NOW() WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "user not found")
	}
	return nil
}

func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
