// Package db holds integration tests that require a real Postgres
// reachable via the DATABASE_URL env var (the same one dev uses). The
// tests connect, run a quick CRUD smoke check, and exit — they're not
// meant to be exhaustive, just to catch wiring regressions locally.
//
// Skipped automatically when DATABASE_URL is empty.
package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bayurstarcool/wingback/backend/internal/repo"
)

const testDSN = "postgres://wingback:wingback_dev_only@127.0.0.1:5432/wingback?sslmode=disable"

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	// We always connect to the wingback test database — never honor
	// DATABASE_URL from the environment, since shared dev shells
	// (SigardaPanel, etc.) may export a different project's DSN.
	_ = os.Getenv("DATABASE_URL")
	dsn := testDSN
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := Connect(ctx, dsn)
	if err != nil {
		t.Skipf("DB not reachable, skipping integration test: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestConnect_PingSucceeds(t *testing.T) {
	pool := openTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestRepo_UserCRUD(t *testing.T) {
	pool := openTestPool(t)
	r := repo.New(pool)
	ctx := context.Background()

	// Use a unique email per run so the test is rerunnable.
	email := "test-" + time.Now().Format("20060102-150405") + "@wingback.test"

	u, err := r.CreateUser(ctx, email, "hashed-pw", "Tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == "" {
		t.Fatal("expected non-empty id")
	}

	got, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("id mismatch: got %q want %q", got.ID, u.ID)
	}

	if err := r.UpdateUserLocation(ctx, u.ID, -6.2, 106.8); err != nil {
		t.Fatalf("update location: %v", err)
	}

	got2, err := r.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got2.LastLat == nil || *got2.LastLat != -6.2 {
		t.Fatalf("last_lat not persisted, got %v", got2.LastLat)
	}
}
