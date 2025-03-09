package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"sync"

	"log/slog"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type KeyManager struct {
	mu       sync.RWMutex
	signKey  jwk.Key   // Current signing key
	signKeys []jwk.Key // All valid signing keys
}

func NewKeyManager(keyID string) (*KeyManager, error) {
	km := &KeyManager{}

	if err := km.generateKey(keyID); err != nil {
		return nil, fmt.Errorf("failed to generate initial key: %w", err)
	}

	return km, nil
}

func (km *KeyManager) generateKey(keyID string) error {
	// Generate a new RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Create a JWK from the private key
	privKey, err := jwk.FromRaw(privateKey)
	if err != nil {
		return fmt.Errorf("failed to create private JWK: %w", err)
	}

	// Set standard headers
	if err := privKey.Set(jwk.KeyIDKey, keyID); err != nil {
		return fmt.Errorf("failed to set key ID: %w", err)
	}
	if err := privKey.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		return fmt.Errorf("failed to set algorithm: %w", err)
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	// Store the keys
	km.signKey = privKey
	if len(km.signKeys) == 0 {
		slog.Debug("adding first private key to manager signKeys",
			"key_id", privKey.KeyID(),
			"current_keys", len(km.signKeys))
		km.signKeys = []jwk.Key{privKey}
	} else {
		slog.Debug("adding private key to manager signKeys",
			"key_id", privKey.KeyID(),
			"current_keys", len(km.signKeys))
		km.signKeys = append(km.signKeys, privKey)
	}

	return nil
}

// GetSigningKey returns the current private key for signing
func (km *KeyManager) GetSigningKey() jwk.Key {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.signKey
}

// GetJWKS returns the public key set in JWKS format
func (km *KeyManager) GetJWKS() jwk.Set {
	km.mu.RLock()
	defer km.mu.RUnlock()

	slog.Debug("getting JWKS", "signKeys_count", len(km.signKeys))

	set := jwk.NewSet()
	for i, key := range km.signKeys {
		slog.Debug("processing key for JWKS",
			"key_id", key.KeyID(),
			"index", i)

		pubKey, err := jwk.PublicKeyOf(key)
		if err != nil {
			slog.Error("failed to get public key",
				"error", err,
				"key_index", i,
				"key_id", key.KeyID())
			continue
		}
		slog.Debug("adding key to JWKS",
			"key_id", pubKey.KeyID(),
			"key_index", i)
		if err := set.AddKey(pubKey); err != nil {
			slog.Error("failed to add key to set",
				"error", err,
				"key_index", i,
				"key_id", pubKey.KeyID())
		}
	}
	return set
}

// GetCurrentKeyID returns the current key ID
func (km *KeyManager) GetCurrentKeyID() string {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.signKeys[0].KeyID()
}

func NewKeyManagerWithKey(keyData []byte, keyID string) (*KeyManager, error) {
	set, err := jwk.Parse(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key data: %w", err)
	}

	key, ok := set.Key(0)
	if !ok {
		return nil, fmt.Errorf("no key found in set")
	}

	// Set the key ID from the database
	if err := key.Set(jwk.KeyIDKey, keyID); err != nil {
		return nil, fmt.Errorf("failed to set key ID: %w", err)
	}

	slog.Debug("creating new key manager with key", "key_id", keyID)

	km := &KeyManager{
		signKey:  key,
		signKeys: []jwk.Key{key},
	}

	slog.Debug("key manager created",
		"key_id", keyID,
		"signKeys_count", len(km.signKeys))

	return km, nil
}

func (km *KeyManager) ExportKey() ([]byte, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return json.Marshal(km.signKey)
}

// AddKey adds a signing key to the manager
func (km *KeyManager) AddKey(key jwk.Key) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	slog.Debug("adding private key to manager signKeys",
		"key_id", key.KeyID(),
		"current_keys", len(km.signKeys))

	// Add to signing keys
	km.signKeys = append(km.signKeys, key)

	// Add public key to JWKS
	pubKey, err := jwk.PublicKeyOf(key)
	if err != nil {
		return fmt.Errorf("failed to create public key: %w", err)
	}

	slog.Debug("adding public key to JWKS signKeys",
		"key_id", pubKey.KeyID(),
		"total_keys", len(km.signKeys))

	return nil
}

// GetSigningKeys returns all signing keys
func (km *KeyManager) GetSigningKeys() []jwk.Key {
	km.mu.RLock()
	defer km.mu.RUnlock()
	slog.Debug("getting signing keys",
		"count", len(km.signKeys),
		"first_key_id", km.signKeys[0].KeyID())
	return km.signKeys
}
