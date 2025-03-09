package sql

import (
	"database/sql"
	"embed"
	"fmt"
)

//go:embed schema/schema.sql
var schemaFS embed.FS

// InitializeDatabase creates and initializes a new database with the schema
func InitializeDatabase(dbPath string) error {
	// Read schema
	schema, err := schemaFS.ReadFile("schema/schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Execute schema
	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}
