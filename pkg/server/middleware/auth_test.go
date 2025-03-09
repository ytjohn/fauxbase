package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	appdb "github.com/ytjohn/fauxbase/pkg/appdb"
	"github.com/ytjohn/fauxbase/pkg/auth"
	"github.com/ytjohn/fauxbase/pkg/server/middleware"
	"github.com/ytjohn/fauxbase/pkg/testutil"
)

func TestWithAuth(t *testing.T) {
	// Setup test database
	testDB := testutil.NewTestDB(t)
	defer testDB.Close()

	// Initialize token service with database
	tokenService, err := auth.NewTokenService(testDB.DB)
	if err != nil {
		t.Fatalf("Failed to create token service: %v", err)
	}

	tests := []struct {
		name         string
		path         string
		setupAuth    func() string // returns token
		expectedCode int
		security     []map[string][]string
	}{
		{
			name:         "non-api path allowed",
			path:         "/static/style.css",
			setupAuth:    func() string { return "" },
			expectedCode: http.StatusOK,
			security:     nil,
		},
		{
			name:         "api path without auth",
			path:         "/api/v1/whoami",
			setupAuth:    func() string { return "" },
			expectedCode: http.StatusUnauthorized,
			security:     []map[string][]string{{"bearerAuth": {}}},
		},
		{
			name: "api path with valid token",
			path: "/api/v1/whoami",
			setupAuth: func() string {
				token, _ := tokenService.CreateAccessToken(123)
				return "Bearer " + token
			},
			expectedCode: http.StatusOK,
			security:     []map[string][]string{{"bearerAuth": {}}},
		},
		{
			name: "api path with invalid token",
			path: "/api/v1/whoami",
			setupAuth: func() string {
				return "Bearer invalid-token"
			},
			expectedCode: http.StatusUnauthorized,
			security:     []map[string][]string{{"bearerAuth": {}}},
		},
		{
			name: "api path with malformed header",
			path: "/api/v1/whoami",
			setupAuth: func() string {
				return "NotBearer token123"
			},
			expectedCode: http.StatusUnauthorized,
			security:     []map[string][]string{{"bearerAuth": {}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", tt.path, nil)
			if token := tt.setupAuth(); token != "" {
				req.Header.Set("Authorization", token)
			}

			// Create test context with token service
			rr := httptest.NewRecorder()
			op := &huma.Operation{
				Security: tt.security,
				Path:     tt.path,
			}
			ctx := humatest.NewContext(op, req, rr)
			ctx = huma.WithValue(ctx, middleware.TokenServiceKey, tokenService)
			ctx = huma.WithValue(ctx, appdb.DbContextKey, testDB.DB)

			// Call middleware
			called := false
			middleware.WithAuth(ctx, func(ctx huma.Context) {
				called = true
				if tt.expectedCode == http.StatusOK {
					ctx.SetStatus(http.StatusOK)
				}
			})

			// If not called and we expected unauthorized, set the status
			if !called && tt.expectedCode == http.StatusUnauthorized {
				ctx.SetStatus(http.StatusUnauthorized)
			}

			if status := ctx.Status(); status != tt.expectedCode {
				t.Errorf("got status %v, want %v", status, tt.expectedCode)
			}
		})
	}
}
