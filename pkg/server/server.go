package server

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	_ "github.com/danielgtaylor/huma/v2/formats/cbor"

	"github.com/ytjohn/fauxbase/pkg/about"
	appdb "github.com/ytjohn/fauxbase/pkg/appdb"
	"github.com/ytjohn/fauxbase/pkg/auth"
	"github.com/ytjohn/fauxbase/pkg/server/handlers/authhandler"
	"github.com/ytjohn/fauxbase/pkg/server/middleware"
)

//go:embed web
var embeddedFiles embed.FS

// FileSystem interface for handling both embedded and local files
type FileSystem interface {
	Open(name string) (fs.File, error)
	ReadFile(name string) ([]byte, error)
}

// LocalFS implements FileSystem for local directory
type LocalFS struct {
	root string
}

// EmbeddedFS implements FileSystem for embedded files
type EmbeddedFS struct {
	fs fs.FS
}

func (l LocalFS) Open(name string) (fs.File, error) {
	return os.Open(l.root + "/" + name)
}

func (l LocalFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(l.root + "/" + name)
}

func (e EmbeddedFS) Open(name string) (fs.File, error) {
	return e.fs.Open(name)
}

func (e EmbeddedFS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(e.fs, name)
}

// Server represents our HTTP server
type Server struct {
	config     *Config
	log        *slog.Logger
	mainRouter *http.ServeMux
	db         *sql.DB
}

// Config holds server configuration
type Config struct {
	Host          string
	Port          int
	Debug         bool
	WebContentDir string
}

// New creates a new server instance
func New(config *Config, log *slog.Logger, db *sql.DB) *Server {
	return &Server{
		config:     config,
		log:        log,
		mainRouter: http.NewServeMux(),
		db:         db,
	}
}

// chooseFileSystem selects between embedded and local filesystem
func (s *Server) chooseFileSystem() FileSystem {
	if s.config.WebContentDir != "" {
		return LocalFS{root: s.config.WebContentDir}
	}
	subFS, err := fs.Sub(embeddedFiles, "web")
	if err != nil {
		s.log.Error("Failed to create sub filesystem", "error", err)
		return nil
	}
	return EmbeddedFS{fs: subFS}
}

// spaFileServer creates a file server that falls back to index.html
func spaFileServer(staticFS http.FileSystem) http.Handler {
	fileServer := http.FileServer(staticFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try to serve the file directly
		f, err := staticFS.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Check if it's a directory
		if path != "/" {
			f, err = staticFS.Open(path + "/index.html")
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fall back to index.html
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

// setupAPI configures the API routes
func (s *Server) setupAPI() *http.ServeMux {
	apiRouter := http.NewServeMux()

	config := huma.DefaultConfig("fauxbase", about.Version)
	config.DocsPath = "/api/v1/docs"
	config.SchemasPath = "/api/v1/schemas"
	config.OpenAPIPath = "/api/v1/openapi"
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearerAuth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

	// Initialize token service
	tokenService, err := auth.NewTokenService(s.db)
	if err != nil {
		fmt.Printf("Failed to initialize token service: %v", err)
		return nil
	}

	api := humago.New(apiRouter, config)

	// Add middleware with the server's logger
	api.UseMiddleware(middleware.WithLogger(s.log))
	api.UseMiddleware(withDB(s.db))
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		ctx = huma.WithValue(ctx, middleware.TokenServiceKey, tokenService)
		ctx = huma.WithValue(ctx, middleware.KeyManagerKey, tokenService.GetKeyManager())
		next(ctx)
	})
	api.UseMiddleware(middleware.WithAuth)
	// Initialize token service

	// Add version endpoint
	huma.Register(api, huma.Operation{
		OperationID: "getVersion",
		Method:      "GET",
		Path:        "/api/v1/version",
		Summary:     "Get application version information",
		// Tags:        []string{"version"},
	}, GetVersion)

	authhandler.RegisterAuthHandlers(api)

	return apiRouter
}

// Start initializes and starts the server
func (s *Server) Start() error {
	// Setup API with database context
	apiRouter := s.setupAPI()

	// // Add middleware that includes our db connection
	// api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
	// 	// Add database to context
	// 	ctx = huma.WithValue(ctx, appdb.DbContextKey, s.db)
	// 	next(ctx)
	// })

	// // Add logging middleware
	// api.UseMiddleware(middleware.WithLogger(s.log))

	s.mainRouter.Handle("/api/v1/", apiRouter)

	// Setup static file serving
	fs := s.chooseFileSystem()
	staticFS := http.FS(fs)
	fsType := "embedded"
	if s.config.WebContentDir != "" {
		fsType = "local"
	}
	s.log.Info("Using filesystem", "type", fsType, "path", s.config.WebContentDir)
	s.mainRouter.Handle("/", spaFileServer(staticFS))

	// Start server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.log.Info("Server starting", "addr", addr, "debug", s.config.Debug, "version", about.Version)
	return http.ListenAndServe(addr, s.mainRouter)
}

// VersionResponse represents the response structure for version info
type VersionResponse struct {
	Body struct {
		AppName   string `json:"appName"`
		ApiName   string `json:"apiName"`
		Project   string `json:"project"`
		Version   string `json:"version"`
		Copyright string `json:"copyright"`
		URL       string `json:"url"`
	} `json:"body"`
}

func GetVersion(ctx context.Context, input *struct{}) (*VersionResponse, error) {
	return &VersionResponse{
		Body: struct {
			AppName   string `json:"appName"`
			ApiName   string `json:"apiName"`
			Project   string `json:"project"`
			Version   string `json:"version"`
			Copyright string `json:"copyright"`
			URL       string `json:"url"`
		}{
			AppName:   about.AppName,
			ApiName:   about.ApiName,
			Project:   about.Project,
			Version:   about.Version,
			Copyright: about.Copyright,
			URL:       about.URL,
		},
	}, nil
}

// Middleware to inject DB into context
func withDB(db *sql.DB) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		// Add DB to context
		ctx = huma.WithValue(ctx, appdb.DbContextKey, db)
		next(ctx)
	}
}

// func connectDB(options *Options) (*sql.DB, error) {
// 	db, err := sql.Open("sqlite3", options.SQLitePath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Enable foreign keys for SQLite
// 	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
// 		return nil, fmt.Errorf("error enabling foreign keys: %w", err)
// 	}

// 	return db, nil
// }
