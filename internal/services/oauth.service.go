package services

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
)

// OAuthService defines OAuth-related business logic
type OAuthService interface {
	GetUserByEmail(email string) (*models.User, error)
	CreateUser(user *models.User) error
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

// CreateUser creates a new user account
func (os *oauthService) CreateUser(user *models.User) error {
	return os.userRepo.CreateUser(user)
}
