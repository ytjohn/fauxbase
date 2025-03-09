package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ytjohn/fauxbase/pkg/appdb/sql"
)

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(dbInitCmd)
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
}

var dbInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the database with schema",
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := viper.GetString("database.path")

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			Log.Error("failed to create database directory", "error", err)
			os.Exit(1)
		}

		Log.Info("initializing database", "path", dbPath)
		if err := sql.InitializeDatabase(dbPath); err != nil {
			Log.Error("failed to initialize database", "error", err)
			os.Exit(1)
		}
		Log.Info("database initialized successfully")
	},
}
