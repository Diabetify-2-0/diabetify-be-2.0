package services

import (
	"crypto/sha256"
	"diabetify/internal/models"
	"diabetify/tests/mocks"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestResetPasswordValidCodeHashesAndDeletesResetRecord(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	resetRepo := new(mocks.MockResetPasswordRepository)
	service := NewUserService(userRepo, resetRepo)

	resetRepo.On("FindByEmailAndCode", "user@example.com", "123456").Return(&models.ResetPassword{
		Email:     "user@example.com",
		Code:      "123456",
		ExpiresAt: time.Now().Add(time.Minute),
	}, nil)
	userRepo.On("GetUserByEmail", "user@example.com").Return(&models.User{
		ID:       1,
		Email:    "user@example.com",
		Password: "old-hash",
	}, nil)
	userRepo.On("UpdateUser", mock.MatchedBy(func(user *models.User) bool {
		return user.ID == 1 &&
			user.Password != "newpassword123" &&
			service.VerifyPassword(user.Password, "newpassword123")
	})).Return(nil)
	resetRepo.On("DeleteByEmail", "user@example.com").Return(nil)

	err := service.ResetPassword("user@example.com", "123456", "newpassword123")

	assert.NoError(t, err)
	userRepo.AssertExpectations(t)
	resetRepo.AssertExpectations(t)
}
