package auth

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	appdb "github.com/ytjohn/fauxbase/pkg/appdb"
	"github.com/ytjohn/fauxbase/pkg/keys"
)

type TokenService struct {
	db         *sql.DB
	keyManager *keys.KeyManager
	issuer     string
	blacklist  map[string]time.Time
	mu         sync.RWMutex // For thread safety
}

type TokenType string

const (
	AccessToken      TokenType = "access"
	RefreshToken     TokenType = "refresh"
	ResetToken       TokenType = "reset"
	keyRetentionDays           = -60 // negative because we're looking back in time
)

func NewTokenService(db *sql.DB) (*TokenService, error) {
	ts := &TokenService{
		db:        db,
		blacklist: make(map[string]time.Time),
	}

	if err := ts.initializeKeyManager(); err != nil {
		return nil, fmt.Errorf("failed to initialize key manager: %w", err)
	}

	return ts, nil
}

func (s *TokenService) initializeKeyManager() error {
	queries := appdb.New(s.db)

	err := s.rotateKeys()
	if err != nil {
		slog.Error("failed to rotate keys", "error", err)
	}

	// Get all valid keys from database
	signingKeys, err := queries.GetAllValidSigningKeys(context.Background())
	if err != nil {
		slog.Error("failed to query signing keys", "error", err)
		return fmt.Errorf("failed to query signing keys: %w", err)
	}

	slog.Debug("found signing keys", "count", len(signingKeys))
	if len(signingKeys) == 0 {
		slog.Info("no valid signing keys found, generating new one")
		return s.generateAndStoreNewKey()
	}

	// Initialize key manager with newest key
	newestKey := signingKeys[0]
	km, err := keys.NewKeyManagerWithKey([]byte(newestKey.KeyData), fmt.Sprintf("%d", newestKey.ID))
	if err != nil {
		return fmt.Errorf("failed to initialize key manager: %w", err)
	}

	slog.Debug("initialized key manager",
		"first_key_id", newestKey.ID,
		"keys_count", len(km.GetSigningKeys()))

	// Add remaining keys
	for i := 1; i < len(signingKeys); i++ {
		key := signingKeys[i]
		slog.Debug("adding additional key", "key_id", key.ID)

		set, err := jwk.Parse([]byte(key.KeyData))
		if err != nil {
			slog.Error("failed to parse key", "error", err)
			continue
		}
		if parsedKey, ok := set.Key(0); ok {
			if err := parsedKey.Set(jwk.KeyIDKey, fmt.Sprintf("%d", key.ID)); err != nil {
				slog.Error("failed to set key ID", "error", err)
				continue
			}
			if err := km.AddKey(parsedKey); err != nil {
				slog.Error("failed to add key", "error", err)
			}
		}
	}

	// Start background key rotation
	go s.rotateKeysInBackground()

	s.keyManager = km

	return nil
}

func (s *TokenService) rotateKeysInBackground() {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	slog.Debug("starting key rotation background task")
	for range ticker.C {
		err := s.rotateKeys()
		if err != nil {
			slog.Error("failed to rotate keys", "error", err)
		}
	}
}

func (s *TokenService) rotateKeys() error {
	queries := appdb.New(s.db)

	// Mark expired keys as inactive
	slog.Debug("marking expired keys as inactive")
	if err := queries.MarkExpiredKeysInactive(context.Background()); err != nil {
		slog.Error("failed to mark expired keys inactive", "error", err)
	}

	// Delete old inactive keys
	days := sql.NullString{String: fmt.Sprintf("%d", keyRetentionDays), Valid: true}
	slog.Debug("cleaning up old keys", "days_threshold", days)
	if err := queries.DeleteExpiredKeys(context.Background(), days); err != nil {
		slog.Error("failed to delete old keys", "error", err)
	}

	// Check for key rotation
	key, err := queries.GetActiveSigningKey(context.Background())
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Debug("no active signing keys found")
			return nil // Not an error condition
		}
		slog.Error("failed to check key expiry", "error", err)
		return err
	}

	// Generate new key if current one expires within 24 hours
	if time.Until(key.ExpiresAt) < 24*time.Hour {
		slog.Info("rotating signing key")
		if err := s.generateAndStoreNewKey(); err != nil {
			slog.Error("failed to rotate key", "error", err)
		}
	}
	return nil
}

func (s *TokenService) generateAndStoreNewKey() error {
	// Generate new key
	km, err := keys.NewKeyManager("1")
	if err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	// Export key data
	keyData, err := km.ExportKey()
	if err != nil {
		return fmt.Errorf("failed to export key: %w", err)
	}

	// Store in database
	queries := appdb.New(s.db)
	_, err = queries.CreateSigningKey(context.Background(), string(keyData))
	if err != nil {
		return fmt.Errorf("failed to store signing key: %w", err)
	}

	s.keyManager = km
	return nil
}

func (s *TokenService) CreateToken(userID int64, tokenType TokenType, duration time.Duration) (string, error) {
	// Check if we need to rotate keys
	queries := appdb.New(s.db)
	key, err := queries.GetActiveSigningKey(context.Background())
	if err != nil {
		slog.Error("failed to check signing key", "error", err)
		// Continue with in-memory key
	} else if time.Until(key.ExpiresAt) < 24*time.Hour {
		slog.Info("signing key approaching expiry, generating new one")
		if err := s.generateAndStoreNewKey(); err != nil {
			slog.Error("failed to generate new key", "error", err)
			// Continue with existing key
		}
	}

	now := time.Now()
	token, err := jwt.NewBuilder().
		IssuedAt(now).
		Issuer(s.issuer).
		Subject(fmt.Sprintf("%d", userID)).
		Expiration(now.Add(duration)).
		Claim("type", string(tokenType)).
		Build()

	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	signKey := s.keyManager.GetSigningKey()
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, signKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

func (s *TokenService) ValidateToken(tokenString string, expectedType TokenType) (int64, error) {
	// Check blacklist first
	s.mu.RLock()
	if _, blacklisted := s.blacklist[tokenString]; blacklisted {
		s.mu.RUnlock()
		return 0, fmt.Errorf("token has been invalidated")
	}
	s.mu.RUnlock()

	token, err := jwt.Parse(
		[]byte(tokenString),
		jwt.WithKey(jwa.RS256, s.keyManager.GetSigningKey()),
		jwt.WithValidate(true),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}

	// Verify token type
	tokenType, ok := token.Get("type")
	if !ok || TokenType(tokenType.(string)) != expectedType {
		return 0, fmt.Errorf("invalid token type")
	}

	// Get user ID from subject
	userID := token.Subject()
	if userID == "" {
		return 0, fmt.Errorf("token missing subject (user ID)")
	}

	var id int64
	_, err = fmt.Sscanf(userID, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID in token: %w", err)
	}

	return id, nil
}

// Helper function to create an access token
func (s *TokenService) CreateAccessToken(userID int64) (string, error) {
	return s.CreateToken(userID, AccessToken, 24*time.Hour)
}

// Helper function to create a refresh token
func (s *TokenService) CreateRefreshToken(userID int64) (string, error) {
	return s.CreateToken(userID, RefreshToken, 30*24*time.Hour)
}

// Helper function to create a password reset token
func (s *TokenService) CreateResetToken(userID int64) (string, error) {
	return s.CreateToken(userID, ResetToken, 24*time.Hour)
}

// Helper function to validate an access token
func (s *TokenService) ValidateAccessToken(tokenString string) (int64, error) {
	return s.ValidateToken(tokenString, AccessToken)
}

// Helper function to validate a refresh token
func (s *TokenService) ValidateRefreshToken(tokenString string) (int64, error) {
	return s.ValidateToken(tokenString, RefreshToken)
}

// Helper function to validate a reset token
func (s *TokenService) ValidateResetToken(tokenString string) (int64, error) {
	return s.ValidateToken(tokenString, ResetToken)
}

func (s *TokenService) InvalidateToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blacklist[token] = time.Now()
	return nil
}

// GetKeyManager returns the key manager instance
func (s *TokenService) GetKeyManager() *keys.KeyManager {
	return s.keyManager
}
