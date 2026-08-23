package postgres

import (
	"context"
	"errors"
	"fmt"

	"diplomaMeetHelper/internal/domain"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

func (r *UserRepository) EnsureUser(ctx context.Context, userID string) (*domain.User, bool, error) {
	queryInsert := `
		INSERT INTO meethelper.users (id, name, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (id) DO NOTHING
		RETURNING id, name, created_at;
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, queryInsert, userID, "").Scan(
		&user.ID,
		&user.Name,
		&user.CreatedAt,
	)

	if err == nil {
		return &user, true, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		existingUser, err := r.GetUser(ctx, userID)
		if err != nil {
			return nil, false, err
		}
		return existingUser, false, nil
	}

	return nil, false, r.mapError(err)
}

func (r *UserRepository) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT id, name, created_at
		FROM meethelper.users
		WHERE id = $1;
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Name,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, r.mapError(err)
	}

	return &user, nil
}

func (r *UserRepository) mapError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return domain.ErrUserAlreadyExists
		case pgerrcode.ConnectionFailure, pgerrcode.ConnectionDoesNotExist, pgerrcode.ConnectionException:
			return domain.ErrDatabaseUnavailable
		}
	}

	return fmt.Errorf("database query error: %w", err)
}
