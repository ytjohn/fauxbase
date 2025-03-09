package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordConfig struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

var DefaultConfig = &PasswordConfig{
	time:    1,
	memory:  64 * 1024,
	threads: 4,
	keyLen:  32,
}

// HashPassword creates an Argon2id hash of a plain text password
func HashPassword(password string, config *PasswordConfig) (string, error) {
	if config == nil {
		config = DefaultConfig
	}

	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt,
		config.time,
		config.memory,
		config.threads,
		config.keyLen,
	)

	// Base64 encode the salt and hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	encodedHash := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		config.memory,
		config.time,
		config.threads,
		b64Salt,
		b64Hash)

	return encodedHash, nil
}

// VerifyPassword checks if a password matches a hash
func VerifyPassword(password, encodedHash string) (bool, error) {
	// Extract the parameters, salt and derived key from the encoded password hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	var config PasswordConfig
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&config.memory,
		&config.time,
		&config.threads)
	if err != nil {
		return false, err
	}
	config.keyLen = 32

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	// Compute the hash of the provided password using the same parameters
	otherHash := argon2.IDKey([]byte(password), salt,
		config.time,
		config.memory,
		config.threads,
		config.keyLen,
	)

	// Check if the hashes match
	return subtle.ConstantTimeCompare(hash, otherHash) == 1, nil
}
