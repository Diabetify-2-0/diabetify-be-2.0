package services

import (
	"errors"
	"fmt"
	"log"

	"diabetify/internal/models"
	"diabetify/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// IUserService defines the interface for user operations
type IUserService interface {
	CreateUser(email, password, name, role string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(userID uint) (*models.User, error)
	UpdateUserRole(userID uint, newRole string) error
}

// UserService implements IUserService
type UserService struct {
	repo repository.UserRepository
	log  *log.Logger
}

// NewUserService creates a new UserService instance
func NewUserService(repo repository.UserRepository, log *log.Logger) IUserService {
	return &UserService{
		repo: repo,
		log:  log,
	}
}

// CreateUser creates a new user with the specified role
func (s *UserService) CreateUser(email, password, name, role string) (*models.User, error) {
	// Validate role
	validRoles := map[string]bool{"USER": true, "DATA_SCIENTIST": true, "MEDICAL_EXPERT": true}
	if !validRoles[role] {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	// Validate email format (basic)
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	// Validate password strength
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	// Validate name
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}

	// Check if email already exists
	existing, err := s.repo.GetUserByEmail(email)
	if err == nil && existing != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Printf("Error hashing password: %v", err)
		return nil, errors.New("error processing password")
	}

	// Create user object
	user := &models.User{
		Email:    email,
		Password: string(hashedPassword),
		Name:     name,
		Role:     role,
		Verified: false,
	}

	// Save to database (repo.CreateUser expected)
	err = s.repo.CreateUser(user)
	if err != nil {
		s.log.Printf("Error creating user: %v", err)
		return nil, errors.New("error creating user")
	}

	// Log user creation
	s.log.Printf("User created: id=%d, email=%s, role=%s", user.ID, user.Email, user.Role)

	return user, nil
}

// CreateDataScientist creates a new Data Scientist user
// qualification: relevant qualifications/degrees/institution
func (s *UserService) CreateDataScientist(email, password, name, qualification string) (*models.User, error) {
	if qualification == "" {
		return nil, errors.New("qualification cannot be empty")
	}

	// Create user with DATA_SCIENTIST role
	user, err := s.CreateUser(email, password, name, "DATA_SCIENTIST")
	if err != nil {
		return nil, err
	}

	// TODO: Phase 2 - Store qualification in separate table
	// TODO: Phase 2 - Send notification to admin for approval
	// TODO: Phase 2 - Log in audit trail

	s.log.Printf("Data Scientist registered: id=%d, email=%s, qualification=%s", user.ID, user.Email, qualification)

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	return s.repo.GetUserByEmail(email)
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(userID uint) (*models.User, error) {
	return s.repo.GetUserByID(userID)
}

// UpdateUserRole updates a user's role
func (s *UserService) UpdateUserRole(userID uint, newRole string) error {
	// Validate new role
	validRoles := map[string]bool{"USER": true, "DATA_SCIENTIST": true, "MEDICAL_EXPERT": true}
	if !validRoles[newRole] {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	// Get current user
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %v", err)
	}

	oldRole := user.Role

	// Update role
	user.Role = newRole
	err = s.repo.UpdateUser(user)
	if err != nil {
		s.log.Printf("Error updating user role: %v", err)
		return errors.New("error updating user role")
	}

	// Log role change
	s.log.Printf("User role updated: id=%d, from=%s, to=%s", userID, oldRole, newRole)

	return nil
}
