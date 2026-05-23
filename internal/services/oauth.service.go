package services

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
)

// OAuthService defines OAuth-related business logic
type OAuthService interface {
	GetUserByEmail(email string) (*models.User, error)
	CreateGoogleUser(email, name string) (*models.User, error)
}

type oauthService struct {
	userRepo repository.UserRepository
}

// NewOAuthService creates a new OAuthService
func NewOAuthService(userRepo repository.UserRepository) OAuthService {
	return &oauthService{
		userRepo: userRepo,
	}
}

// GetUserByEmail retrieves a user by their email
func (os *oauthService) GetUserByEmail(email string) (*models.User, error) {
	return os.userRepo.GetUserByEmail(email)
}

// CreateGoogleUser creates a new Google-backed user account with safe defaults.
func (os *oauthService) CreateGoogleUser(email, name string) (*models.User, error) {
	user := &models.User{
		Email:    email,
		Name:     name,
		Password: "",
		Role:     "USER",
		Verified: true,
	}

	if err := os.userRepo.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}
