package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"diplomaMeetHelper/internal/adapters/db/postgres"
	"diplomaMeetHelper/internal/domain"
	"diplomaMeetHelper/migrations"
	"diplomaMeetHelper/pkg/migrator"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestDSN() string {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgrespassword@localhost:5432/meethelper?sslmode=disable&search_path=meethelper"
	}
	return dsn
}

func setupTestDB(t *testing.T) (*pgxpool.Pool, context.Context) {
	ctx := context.Background()
	dsn := getTestDSN()

	err := migrator.Run(migrations.FS, ".", dsn)
	if err != nil {
		t.Skipf("skipping integration test: database unavailable (%v)", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test: failed to connect to pool (%v)", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration test: failed to ping database (%v)", err)
	}

	return pool, ctx
}

func TestPostgresUserRepository_Integration(t *testing.T) {
	pool, ctx := setupTestDB(t)
	defer pool.Close()

	repo := postgres.NewUserRepository(pool)
	testUserID := "integration-user-" + uuid.NewString()

	user, created, err := repo.EnsureUser(ctx, testUserID)
	if err != nil {
		t.Fatalf("EnsureUser failed: %v", err)
	}
	if !created {
		t.Errorf("expected created = true for new user, got false")
	}
	if user.ID != testUserID {
		t.Errorf("expected user ID %q, got %q", testUserID, user.ID)
	}

	userSecond, createdSecond, err := repo.EnsureUser(ctx, testUserID)
	if err != nil {
		t.Fatalf("EnsureUser second call failed: %v", err)
	}
	if createdSecond {
		t.Errorf("expected created = false for existing user, got true")
	}
	if userSecond.ID != testUserID {
		t.Errorf("expected user ID %q, got %q", testUserID, userSecond.ID)
	}

	fetchedUser, err := repo.GetUser(ctx, testUserID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if fetchedUser.ID != testUserID {
		t.Errorf("expected fetched ID %q, got %q", testUserID, fetchedUser.ID)
	}

	_, err = repo.GetUser(ctx, "non-existent-"+uuid.NewString())
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}
