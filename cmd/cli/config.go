package cli

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Database struct {
		Path string
	}
	Server struct {
		Host string
		Port int
	}
	Debug bool
}

// GlobalConfig is the global configuration instance
var GlobalConfig Config

// Global logger instance
var Log *slog.Logger

// initConfig initializes the configuration with defaults and environment variables
func initConfig() {
	// Set defaults
	viper.SetDefault("database.path", "data/devdatabase.db")
	viper.SetDefault("server.host", "127.0.0.1")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("debug", false)

	// Environment variables
	viper.SetEnvPrefix("FAUXBASE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read into struct
	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		panic(err)
	}

	// Initialize logger
	var logLevel slog.Level
	if GlobalConfig.Debug {
		logLevel = slog.LevelDebug
	} else {
		logLevel = slog.LevelInfo
	}

	Log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	Log.Debug("debug logging enabled")
}
