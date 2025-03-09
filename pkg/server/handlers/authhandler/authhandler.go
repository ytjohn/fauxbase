package authhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lestrrat-go/jwx/v2/jwk"
	appdb "github.com/ytjohn/fauxbase/pkg/appdb"
	"github.com/ytjohn/fauxbase/pkg/auth"
	"github.com/ytjohn/fauxbase/pkg/keys"
	"github.com/ytjohn/fauxbase/pkg/server/middleware"
)

// WhoAmIResponse represents the response structure for user info
type WhoAmIResponse struct {
	Body struct {
		UserID    int64  `json:"userId"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		LastLogin string `json:"lastLogin,omitempty"`
	} `json:"body"`
}

// JWKSResponse represents the response structure for JWKS
type JWKSResponse struct {
	Body jwk.Set `json:"body"`
}

type LoginRequest struct {
	Body struct {
		Email    string `json:"email" huma:"required,format=email"`
		Password string `json:"password" huma:"required,min=1"`
	} `json:"body"`
}

type LoginResponse struct {
	Body struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		TokenType    string `json:"tokenType"`
		ExpiresIn    int    `json:"expiresIn"` // seconds
	} `json:"body"`
}

type RefreshTokenRequest struct {
	Body struct {
		RefreshToken string `json:"refreshToken" huma:"required"`
	}
}

type RefreshTokenResponse struct {
	Body struct {
		AccessToken string `json:"accessToken"`
		TokenType   string `json:"tokenType"`
		ExpiresIn   int    `json:"expiresIn"`
	}
}

// RegisterAuthHandlers registers all authentication-related handlers
func RegisterAuthHandlers(api huma.API) {

	// WhoAmI endpoint (auth required)
	huma.Register(api, huma.Operation{
		OperationID: "whoami",
		Method:      "GET",
		Path:        "/api/v1/whoami",
		Summary:     "Get current user information",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, WhoAmI)

	// JWKS endpoint
	huma.Register(api, huma.Operation{
		OperationID: "getJWKS",
		Method:      "GET",
		Path:        "/.well-known/jwks.json",
		Summary:     "Get JSON Web Key Set (JWKS)",
		Tags:        []string{"auth"},
	}, GetJWKS)

	// Login endpoint
	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      "POST",
		Path:        "/api/v1/auth/login",
		Summary:     "Login with email and password",
		Tags:        []string{"auth"},
	}, Login)

	huma.Register(api, huma.Operation{
		OperationID: "refreshToken",
		Method:      "POST",
		Path:        "/api/v1/auth/refresh",
		Summary:     "Refresh access token using refresh token",
		Tags:        []string{"auth"},
	}, RefreshToken)

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      "POST",
		Path:        "/api/v1/auth/logout",
		Summary:     "Logout and invalidate current token",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, Logout)
}

func WhoAmI(ctx context.Context, _ *struct{}) (*WhoAmIResponse, error) {
	userVal := ctx.Value(middleware.UserContextKey)
	if userVal == nil {
		return nil, huma.Error401Unauthorized("not authenticated")
	}

	var user *appdb.User
	switch u := userVal.(type) {
	case *appdb.User:
		user = u
	case appdb.User:
		user = &u
	default:
		return nil, fmt.Errorf("invalid user type in context")
	}

	// Get database connection
	db := ctx.Value(appdb.DbContextKey).(*sql.DB)
	queries := appdb.New(db)

	me, err := queries.GetUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	response := &WhoAmIResponse{}
	response.Body.UserID = user.ID
	response.Body.Email = me.Email
	response.Body.Role = me.Role
	response.Body.LastLogin = me.Lastlogin.Time.Format(time.RFC3339)

	return response, nil
}

func GetJWKS(ctx context.Context, input *struct{}) (*JWKSResponse, error) {
	keyManager := ctx.Value(middleware.KeyManagerKey).(*keys.KeyManager)
	return &JWKSResponse{
		Body: keyManager.GetJWKS(),
	}, nil
}

func Login(ctx context.Context, input *LoginRequest) (*LoginResponse, error) {
	logger := middleware.GetLogger(ctx)
	logger.Debug("login attempt", "email", input.Body.Email)

	// Get database from context
	db, ok := ctx.Value(appdb.DbContextKey).(*sql.DB)
	if !ok {
		logger.Error("database not found in context")
		return nil, fmt.Errorf("database connection not found in context")
	}

	queries := appdb.New(db)

	// Get user by email
	user, err := queries.GetUserByEmail(ctx, input.Body.Email)
	if err != nil {
		logger.Error("login failed", "error", err, "email", input.Body.Email)
		return nil, huma.Error401Unauthorized("invalid credentials")
	}

	// Verify password
	valid, err := auth.VerifyPassword(input.Body.Password, user.Password)
	if err != nil {
		logger.Error("password verification failed", "error", err)
		return nil, huma.Error401Unauthorized("invalid credentials")
	}
	if !valid {
		logger.Debug("invalid password", "email", input.Body.Email)
		return nil, huma.Error401Unauthorized("invalid credentials")
	}

	// Get token service from context
	tokenService := ctx.Value(middleware.TokenServiceKey).(*auth.TokenService)

	// Generate tokens
	accessToken, err := tokenService.CreateAccessToken(user.ID)
	if err != nil {
		logger.Error("failed to create access token", "error", err)
		return nil, fmt.Errorf("failed to create access token")
	}

	refreshToken, err := tokenService.CreateRefreshToken(user.ID)
	if err != nil {
		logger.Error("failed to create refresh token", "error", err)
		return nil, fmt.Errorf("failed to create refresh token")
	}

	// Update last login
	if err := queries.UpdateUserLastLogin(ctx, user.ID); err != nil {
		logger.Error("failed to update last login", "error", err)
		// Non-critical error, continue
	}

	return &LoginResponse{
		Body: struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			TokenType    string `json:"tokenType"`
			ExpiresIn    int    `json:"expiresIn"`
		}{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    24 * 60 * 60, // 24 hours in seconds
		},
	}, nil
}

func RefreshToken(ctx context.Context, input *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	logger := middleware.GetLogger(ctx)

	// Get token service from context
	tokenService := ctx.Value(middleware.TokenServiceKey).(*auth.TokenService)

	// Validate refresh token
	userID, err := tokenService.ValidateToken(input.Body.RefreshToken, auth.RefreshToken)
	if err != nil {
		logger.Debug("invalid refresh token", "error", err)
		return nil, huma.Error401Unauthorized("invalid refresh token")
	}

	// Generate new access token
	accessToken, err := tokenService.CreateAccessToken(userID)
	if err != nil {
		logger.Error("failed to create access token", "error", err)
		return nil, fmt.Errorf("failed to create access token")
	}

	return &RefreshTokenResponse{
		Body: struct {
			AccessToken string `json:"accessToken"`
			TokenType   string `json:"tokenType"`
			ExpiresIn   int    `json:"expiresIn"`
		}{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   24 * 60 * 60, // 24 hours in seconds
		},
	}, nil
}

func Logout(ctx context.Context, _ *struct{}) (*struct{}, error) {
	// Get user from context (set by auth middleware)
	userVal := ctx.Value(middleware.UserContextKey)
	if userVal == nil {
		return nil, huma.Error401Unauthorized("not authenticated")
	}

	// Get token service from context
	tokenService := ctx.Value(middleware.TokenServiceKey).(*auth.TokenService)

	// Get token from context
	token := ctx.Value(middleware.TokenContextKey).(string)
	if token == "" {
		return nil, huma.Error401Unauthorized("no token found")
	}

	// Invalidate the token
	if err := tokenService.InvalidateToken(token); err != nil {
		return nil, fmt.Errorf("failed to invalidate token: %w", err)
	}

	return &struct{}{}, nil
}
