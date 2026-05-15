package services

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createLegacySHA256PasswordHash(password string) string {
	salt := make([]byte, 8)
	for i := range salt {
		salt[i] = byte(i)
	}

	h := sha256.New()
	h.Write([]byte(password))
	h.Write(salt)
	hash := h.Sum(nil)

	return hex.EncodeToString(salt) + hex.EncodeToString(hash)
}

func TestHashPasswordUsesBcrypt(t *testing.T) {
	service := &userService{}

	hash, err := service.hashPassword("password123")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Contains(t, hash, "$2")
	assert.True(t, service.VerifyPassword(hash, "password123"))
	assert.False(t, service.VerifyPassword(hash, "wrong-password"))
}

func TestVerifyPasswordSupportsLegacySHA256Hashes(t *testing.T) {
	service := &userService{}
	legacyHash := createLegacySHA256PasswordHash("password123")

	assert.True(t, service.VerifyPassword(legacyHash, "password123"))
	assert.False(t, service.VerifyPassword(legacyHash, "wrong-password"))
}
