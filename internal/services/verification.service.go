package services

import (
	"errors"
	"time"

	"diabetify/internal/models"
	"diabetify/internal/repository"
	"diabetify/internal/utils"
)

// VerificationService defines the interface for email verification service
type VerificationService interface {
	SendVerificationCode(email string) (string, error)
	VerifyCode(email, code string) error
	ResendVerificationCode(email string) (string, error)
}

type verificationService struct {
	verificationRepo repository.VerificationRepository
	userRepo         repository.UserRepository
	mailConfig       utils.MailConfig
}

// NewVerificationService creates a new verification service
func NewVerificationService(verificationRepo repository.VerificationRepository, userRepo repository.UserRepository) VerificationService {
	return &verificationService{
		verificationRepo: verificationRepo,
		userRepo:         userRepo,
		mailConfig:       utils.LoadMailConfig(),
	}
}

// SendVerificationCode generates a verification code and sends it via email
func (vs *verificationService) SendVerificationCode(email string) (string, error) {
	if email == "" {
		return "", errors.New("email cannot be empty")
	}

	// Check if user exists
	_, err := vs.userRepo.GetUserByEmail(email)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Generate a 6-digit code
	code := utils.GenerateVerificationCode()

	// Delete old verification code if exists
	vs.verificationRepo.DeleteByEmail(email)

	// Store new verification code in DB
	verification := &models.Verification{
		Email:     email,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := vs.verificationRepo.CreateVerification(verification); err != nil {
		return "", errors.New("failed to create verification code")
	}

	return code, nil
}

// VerifyCode verifies the provided code for the user's email
func (vs *verificationService) VerifyCode(email, code string) error {
	if email == "" || code == "" {
		return errors.New("email and code cannot be empty")
	}

	// Find verification record by email and code
	verification, err := vs.verificationRepo.FindByEmailAndCode(email, code)
	if err != nil {
		return errors.New("invalid or expired verification code")
	}

	if verification == nil {
		return errors.New("verification record not found")
	}

	// Check if code has expired
	if time.Now().After(verification.ExpiresAt) {
		return errors.New("verification code has expired")
	}

	// Mark user as verified
	if err := vs.userRepo.SetUserVerified(email); err != nil {
		return errors.New("failed to verify user")
	}

	// Delete verification code after successful verification
	if err := vs.verificationRepo.DeleteByEmail(email); err != nil {
		// Log error but don't fail - user is already verified
	}

	return nil
}

// ResendVerificationCode resends the verification code to user's email
func (vs *verificationService) ResendVerificationCode(email string) (string, error) {
	// Simply call SendVerificationCode - it handles everything
	return vs.SendVerificationCode(email)
}
