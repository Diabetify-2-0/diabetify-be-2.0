package services

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	mathrand "math/rand"
	"time"

	"diabetify/internal/models"
	"diabetify/internal/repository"
)

// UserService defines the interface for user service
type UserService interface {
	CreateUser(email, password, name, role string, gender, dob *string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id uint) (*models.User, error)
	GetAllUsers() ([]*models.User, error)
	VerifyPassword(hashedPassword, password string) bool
	UpdateUser(user *models.User) error
	DeleteUser(id uint) error
	SetUserVerified(email string) error
	PatchUser(id uint, data map[string]interface{}) error
	ForgotPassword(email string) (string, error)
	ResetPassword(email, code, newPassword string) error
}

// userService implements UserService
type userService struct {
	userRepo repository.UserRepository
	rpRepo   repository.ResetPasswordRepository
}

// NewUserService creates a new UserService instance
func NewUserService(userRepo repository.UserRepository, rpRepo repository.ResetPasswordRepository) UserService {
	return &userService{
		userRepo: userRepo,
		rpRepo:   rpRepo,
	}
}

// hashPassword hashes a password using SHA256 + salt
func (s *userService) hashPassword(password string) (string, error) {
	salt := make([]byte, 8)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	// SHA256
	h := sha256.New()
	h.Write([]byte(password))
	h.Write(salt)
	hash := h.Sum(nil)

	return hex.EncodeToString(salt) + hex.EncodeToString(hash), nil
}

// CreateUser creates a new user with validation
func (s *userService) CreateUser(email, password, name, role string, gender, dob *string) (*models.User, error) {
	// Validate inputs
	if email == "" {
		return nil, errors.New("email is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}
	if role == "" {
		return nil, errors.New("role is required")
	}

	// Validate role
	validRoles := map[string]bool{"ADMIN": true, "USER": true, "DATA_SCIENTIST": true, "MEDICAL_EXPERT": true}
	if !validRoles[role] {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	// Check if email already exists
	existingUser, err := s.userRepo.GetUserByEmail(email)
	if err == nil && existingUser != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create user
	user := &models.User{
		Email:    email,
		Password: hashedPassword,
		Name:     name,
		Role:     role,
		Gender:   gender,
		DOB:      dob,
		Verified: false,
	}

	if err := s.userRepo.CreateUser(user); err != nil {
		return nil, errors.New("failed to create user")
	}

	return user, nil
}

// VerifyPassword verifies a password against a hash
func (s *userService) VerifyPassword(hashedPassword, password string) bool {
	if len(hashedPassword) < 16 {
		return false
	}

	salt, err := hex.DecodeString(hashedPassword[:16])
	if err != nil {
		return false
	}

	expectedHash, err := hex.DecodeString(hashedPassword[16:])
	if err != nil {
		return false
	}

	h := sha256.New()
	h.Write([]byte(password))
	h.Write(salt)
	hash := h.Sum(nil)

	return bytes.Equal(hash, expectedHash)
}

// GetUserByEmail retrieves a user by email
func (s *userService) GetUserByEmail(email string) (*models.User, error) {
	return s.userRepo.GetUserByEmail(email)
}

// GetUserByID retrieves a user by ID
func (s *userService) GetUserByID(id uint) (*models.User, error) {
	return s.userRepo.GetUserByID(id)
}

func (s *userService) GetAllUsers() ([]*models.User, error) {
	return s.userRepo.GetAllUsers()
}

// UpdateUser updates a user
func (s *userService) UpdateUser(user *models.User) error {
	if user.ID == 0 {
		return errors.New("user ID is required")
	}

	// If password is being updated, hash it
	if user.Password != "" {
		if len(user.Password) < 8 {
			return errors.New("password must be at least 8 characters")
		}

		hashedPassword, err := s.hashPassword(user.Password)
		if err != nil {
			return errors.New("failed to hash password")
		}
		user.Password = hashedPassword
	}

	return s.userRepo.UpdateUser(user)
}

// DeleteUser deletes a user
func (s *userService) DeleteUser(id uint) error {
	if id == 0 {
		return errors.New("user ID is required")
	}
	return s.userRepo.DeleteUser(id)
}

// SetUserVerified marks a user as verified
func (s *userService) SetUserVerified(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	return s.userRepo.SetUserVerified(email)
}

// PatchUser partially updates a user
func (s *userService) PatchUser(id uint, data map[string]interface{}) error {
	if id == 0 {
		return errors.New("user ID is required")
	}

	// If password is being updated in patch, validate and hash it
	if password, exists := data["password"]; exists {
		passwordStr, ok := password.(string)
		if !ok {
			return errors.New("password must be a string")
		}
		if len(passwordStr) < 8 {
			return errors.New("password must be at least 8 characters")
		}

		hashedPassword, err := s.hashPassword(passwordStr)
		if err != nil {
			return errors.New("failed to hash password")
		}
		data["password"] = hashedPassword
	}

	return s.userRepo.PatchUser(id, data)
}

// ForgotPassword creates a password reset code and sends to email
func (s *userService) ForgotPassword(email string) (string, error) {
	if email == "" {
		return "", errors.New("email is required")
	}

	// Check if user exists
	if _, err := s.userRepo.GetUserByEmail(email); err != nil {
		return "", errors.New("email does not exist")
	}

	// Generate verification code (6 digits)
	code := fmt.Sprintf("%06d", mathrand.Intn(1000000))

	// Delete any existing codes for this email
	s.rpRepo.DeleteByEmail(email)

	// Create new reset password record
	resetPassword := &models.ResetPassword{
		Email:     email,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := s.rpRepo.CreateResetPassword(resetPassword); err != nil {
		return "", errors.New("failed to create reset password code")
	}

	return code, nil
}

// ResetPassword resets user password using verification code
func (s *userService) ResetPassword(email, code, newPassword string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if code == "" {
		return errors.New("code is required")
	}
	if newPassword == "" {
		return errors.New("new password is required")
	}

	// Validate password length
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	// Find reset password record
	resetRecord, err := s.rpRepo.FindByEmailAndCode(email, code)
	if err != nil {
		return errors.New("invalid or expired code")
	}

	// Check if code has expired
	if time.Now().After(resetRecord.ExpiresAt) {
		return errors.New("code has expired")
	}

	// Get user
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return errors.New("user not found")
	}

	// Update password
	user.Password = newPassword
	if err := s.UpdateUser(user); err != nil {
		return err
	}

	// Delete reset password code
	s.rpRepo.DeleteByEmail(email)

	return nil
}
