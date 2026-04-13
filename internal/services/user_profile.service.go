package services

import (
	"errors"

	"diabetify/internal/models"
	"diabetify/internal/repository"
)

// UserProfileService defines the interface for user profile service
type UserProfileService interface {
	GetUserProfile(userID uint) (*models.UserProfile, error)
	CreateUserProfile(profile *models.UserProfile) (*models.UserProfile, error)
	UpdateUserProfile(profile *models.UserProfile) (*models.UserProfile, error)
	DeleteUserProfile(userID uint) error
	PatchUserProfile(userID uint, data map[string]interface{}) error
}

type userProfileService struct {
	repo repository.UserProfileRepository
}

// NewUserProfileService creates a new user profile service
func NewUserProfileService(repo repository.UserProfileRepository) UserProfileService {
	return &userProfileService{
		repo: repo,
	}
}

// calculateBMI calculates BMI from weight (kg) and height (cm)
// Returns nil if inputs are invalid
func calculateBMI(weight, height *int) *float64 {
	if weight == nil || height == nil || *height <= 0 {
		return nil
	}

	heightInMeters := float64(*height) / 100.0
	bmi := float64(*weight) / (heightInMeters * heightInMeters)
	return &bmi
}

// GetUserProfile retrieves user profile by user ID
func (ups *userProfileService) GetUserProfile(userID uint) (*models.UserProfile, error) {
	if userID == 0 {
		return nil, errors.New("invalid user ID")
	}

	profile, err := ups.repo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, errors.New("profile not found")
	}

	return profile, nil
}

// CreateUserProfile creates a new user profile with BMI calculation
func (ups *userProfileService) CreateUserProfile(profile *models.UserProfile) (*models.UserProfile, error) {
	if profile == nil {
		return nil, errors.New("profile cannot be nil")
	}

	if profile.UserID == 0 {
		return nil, errors.New("user ID is required")
	}

	// Calculate BMI if weight and height are provided
	if profile.Weight != nil && profile.Height != nil {
		profile.BMI = calculateBMI(profile.Weight, profile.Height)
	}

	if err := ups.repo.Create(profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// UpdateUserProfile updates user profile with BMI recalculation
func (ups *userProfileService) UpdateUserProfile(profile *models.UserProfile) (*models.UserProfile, error) {
	if profile == nil {
		return nil, errors.New("profile cannot be nil")
	}

	if profile.UserID == 0 {
		return nil, errors.New("user ID is required")
	}

	// Verify profile exists
	existingProfile, err := ups.repo.FindByUserID(profile.UserID)
	if err != nil || existingProfile == nil {
		return nil, errors.New("profile not found")
	}

	// Preserve ID and UserID
	profile.ID = existingProfile.ID
	profile.UserID = existingProfile.UserID

	// Recalculate BMI if weight and height are provided
	if profile.Weight != nil && profile.Height != nil {
		profile.BMI = calculateBMI(profile.Weight, profile.Height)
	}

	if err := ups.repo.Update(profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// DeleteUserProfile deletes user profile by user ID
func (ups *userProfileService) DeleteUserProfile(userID uint) error {
	if userID == 0 {
		return errors.New("invalid user ID")
	}

	return ups.repo.DeleteByUserID(userID)
}

// PatchUserProfile patches specific fields of user profile with BMI recalculation if needed
func (ups *userProfileService) PatchUserProfile(userID uint, data map[string]interface{}) error {
	if userID == 0 {
		return errors.New("invalid user ID")
	}

	if len(data) == 0 {
		return errors.New("no data to patch")
	}

	// If weight or height is being updated, recalculate BMI
	if _, hasWeight := data["weight"]; hasWeight || data["height"] != nil {
		profile, err := ups.repo.FindByUserID(userID)
		if err != nil || profile == nil {
			return errors.New("profile not found")
		}

		// Recalculate BMI with existing or new values
		weight := profile.Weight
		height := profile.Height

		if w, ok := data["weight"]; ok {
			if wVal, ok := w.(float64); ok {
				iVal := int(wVal)
				weight = &iVal
			}
		}

		if h, ok := data["height"]; ok {
			if hVal, ok := h.(float64); ok {
				iVal := int(hVal)
				height = &iVal
			}
		}

		if weight != nil && height != nil {
			bmi := calculateBMI(weight, height)
			data["BMI"] = bmi
		}
	}

	return ups.repo.Patch(userID, data)
}
