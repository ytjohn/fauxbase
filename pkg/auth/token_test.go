package auth_test

import (
	"testing"
	"time"

	"github.com/ytjohn/fauxbase/pkg/auth"
	"github.com/ytjohn/fauxbase/pkg/testutil"
)

func TestTokenService(t *testing.T) {
	// Setup test database
	testDB := testutil.NewTestDB(t)
	defer testDB.Close()

	service, err := auth.NewTokenService(testDB.DB)
	if err != nil {
		t.Fatalf("Failed to create token service: %v", err)
	}

	userID := int64(123)

	tests := []struct {
		name       string
		tokenType  auth.TokenType
		duration   time.Duration
		shouldFail bool
	}{
		{
			name:      "access token",
			tokenType: auth.AccessToken,
			duration:  24 * time.Hour,
		},
		{
			name:      "refresh token",
			tokenType: auth.RefreshToken,
			duration:  30 * 24 * time.Hour,
		},
		{
			name:      "reset token",
			tokenType: auth.ResetToken,
			duration:  24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create token
			token, err := service.CreateToken(userID, tt.tokenType, tt.duration)
			if err != nil {
				t.Fatalf("Failed to create token: %v", err)
			}

			// Validate token
			gotUserID, err := service.ValidateToken(token, tt.tokenType)
			if err != nil {
				t.Fatalf("Failed to validate token: %v", err)
			}

			if gotUserID != userID {
				t.Errorf("Got user ID %d, want %d", gotUserID, userID)
			}

			// Test wrong token type
			_, err = service.ValidateToken(token, "wrong-type")
			if err == nil {
				t.Error("Expected error for wrong token type, got nil")
			}
		})
	}

	// Test helper functions
	t.Run("helper functions", func(t *testing.T) {
		// Access token
		accessToken, err := service.CreateAccessToken(userID)
		if err != nil {
			t.Fatalf("Failed to create access token: %v", err)
		}
		gotUserID, err := service.ValidateAccessToken(accessToken)
		if err != nil {
			t.Fatalf("Failed to validate access token: %v", err)
		}
		if gotUserID != userID {
			t.Errorf("Got user ID %d, want %d", gotUserID, userID)
		}

		// Refresh token
		refreshToken, err := service.CreateRefreshToken(userID)
		if err != nil {
			t.Fatalf("Failed to create refresh token: %v", err)
		}
		gotUserID, err = service.ValidateRefreshToken(refreshToken)
		if err != nil {
			t.Fatalf("Failed to validate refresh token: %v", err)
		}
		if gotUserID != userID {
			t.Errorf("Got user ID %d, want %d", gotUserID, userID)
		}
	})
}
