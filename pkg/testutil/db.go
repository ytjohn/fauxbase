package testutil

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	appdb "github.com/ytjohn/fauxbase/pkg/appdb"
	"github.com/ytjohn/fauxbase/pkg/server/middleware"
)

// TestDB represents a test database instance
type TestDB struct {
	*sql.DB
	t *testing.T
}

// getProjectRoot returns the absolute path to the project root
func getProjectRoot() (string, error) {
	// Get the current file's path
	_, currentFile, _, _ := runtime.Caller(0)
	// Go up two directories (from pkg/testutil to project root)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")
	return filepath.Abs(projectRoot)
}

// NewTestDB creates a new test database
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	projectRoot, err := getProjectRoot()
	if err != nil {
		t.Fatalf("failed to get project root: %v", err)
	}

	// Create a temporary file for SQLite
	f, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(f.Name())
	})

	// Connect to the database
	db, err := sql.Open("sqlite3", f.Name())
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Initialize schema
	schema := []string{
		"database/schema/auth.sql",
		"database/schema/members.sql",
		"database/schema/positions.sql",
		"database/schema/officer_roles.sql",
		"database/schema/ares_details.sql",
	}

	for _, file := range schema {
		fullPath := filepath.Join(projectRoot, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("failed to read schema file %s: %v", fullPath, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			t.Fatalf("failed to execute schema %s: %v", file, err)
		}
	}

	return &TestDB{DB: db, t: t}
}

// CreateTestContext creates a new context with test database and response writer
func CreateTestContext(t *testing.T, db *sql.DB) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, appdb.DbContextKey, db)

	// Add mock response writer to context
	w := CreateTestResponseWriter()
	ctx = context.WithValue(ctx, middleware.ResponseWriterKey, w)

	return ctx
}
