package middleware

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	appdb "github.com/ytjohn/fauxbase/pkg/appdb"
	"github.com/ytjohn/fauxbase/pkg/auth"
)

type UserContext struct {
	ID        int64
	Email     string
	Roles     []string
	IsActive  bool
	LastLogin time.Time
}

type AuthConfig struct {
	TokenService *auth.TokenService
	API          huma.API
	// Remove PublicPaths and ExactPaths as they're no longer needed
}

func WithAuth(ctx huma.Context, next func(huma.Context)) {
	// Check if authentication is required for this path
	op := ctx.Operation()
	if op == nil || len(op.Security) == 0 {
		// No security requirements, continue
		next(ctx)
		return
	}

	// Get the token from the Authorization header
	authHeader := ctx.Header("Authorization")
	// slog.Debug("auth middleware", "header", authHeader)

	if authHeader == "" {
		ctx.SetStatus(http.StatusUnauthorized)
		return
	}

	// Check if it's a Bearer token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		ctx.SetStatus(http.StatusUnauthorized)
		return
	}

	// Get services from context
	tokenService := ctx.Context().Value(TokenServiceKey).(*auth.TokenService)
	claims, err := tokenService.ValidateToken(parts[1], auth.AccessToken)
	if err != nil {
		ctx.SetStatus(http.StatusUnauthorized)
		return
	}

	// Get user from database
	db := ctx.Context().Value(appdb.DbContextKey).(*sql.DB)
	queries := appdb.New(db)

	user, err := queries.GetUser(ctx.Context(), claims)
	if err != nil {
		slog.Debug("user lookup failed", "error", err)
		next(ctx)
		return
	}

	// Set user and token in context
	ctx = huma.WithValue(ctx, UserContextKey, user)
	ctx = huma.WithValue(ctx, TokenContextKey, parts[1])
	slog.Debug("auth successful", "user_id", user.ID)

	next(ctx)
}
