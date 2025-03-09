package cli

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ytjohn/fauxbase/pkg/server"
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().String("webdir", "", "Path to web content directory (uses embedded if not set)")
	if err := viper.BindPFlag("server.webdir", serverCmd.Flags().Lookup("webdir")); err != nil {
		panic(fmt.Sprintf("failed to bind webdir flag: %v", err))
	}
}

var serverCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Run: func(cmd *cobra.Command, args []string) {
		// Open database connection
		Log.Debug("opening database", "path", GlobalConfig.Database.Path)
		db, err := sql.Open("sqlite", GlobalConfig.Database.Path)
		if err != nil {
			Log.Error("failed to open database", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		config := &server.Config{
			Host:          GlobalConfig.Server.Host,
			Port:          GlobalConfig.Server.Port,
			Debug:         GlobalConfig.Debug,
			WebContentDir: viper.GetString("server.webdir"),
		}

		srv := server.New(config, Log, db)
		Log.Info("starting server", "host", config.Host, "port", config.Port)
		if err := srv.Start(); err != nil {
			Log.Error("server error", "error", err)
			os.Exit(1)
		}
	},
}
