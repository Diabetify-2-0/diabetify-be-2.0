package services

import (
	"time"

	"diabetify/internal/models"
	"diabetify/internal/repository"
)

// ActivityService defines the interface for activity service
type ActivityService interface {
	CreateActivity(activity *models.Activity) error
	GetActivityByID(id uint) (*models.Activity, error)
	GetCurrentUserActivities(userID uint, limit int) ([]models.Activity, error)
	GetActivitiesByDateRange(userID uint, startDate, endDate time.Time) ([]models.Activity, error)
	UpdateActivity(activity *models.Activity) error
	DeleteActivity(id uint) error
	CountUserActivities(userID uint) (int64, error)
}

// activityService implements ActivityService
type activityService struct {
	repo repository.ActivityRepository
}

// NewActivityService creates a new ActivityService instance
func NewActivityService(repo repository.ActivityRepository) ActivityService {
	return &activityService{
		repo: repo,
	}
}

// CreateActivity creates a new activity
func (s *activityService) CreateActivity(activity *models.Activity) error {
	return s.repo.Create(activity)
}

// GetActivityByID retrieves an activity by ID
func (s *activityService) GetActivityByID(id uint) (*models.Activity, error) {
	return s.repo.FindByID(id)
}

// GetCurrentUserActivities retrieves all activities for a user
func (s *activityService) GetCurrentUserActivities(userID uint, limit int) ([]models.Activity, error) {
	return s.repo.FindAllByUserID(userID, limit)
}

// GetActivitiesByDateRange retrieves activities within a date range
func (s *activityService) GetActivitiesByDateRange(userID uint, startDate, endDate time.Time) ([]models.Activity, error) {
	return s.repo.FindByUserIDAndActivityDateRange(userID, startDate, endDate)
}

// UpdateActivity updates an existing activity
func (s *activityService) UpdateActivity(activity *models.Activity) error {
	return s.repo.Update(activity)
}

// DeleteActivity deletes an activity
func (s *activityService) DeleteActivity(id uint) error {
	return s.repo.Delete(id)
}

// CountUserActivities counts the total number of activities for a user
func (s *activityService) CountUserActivities(userID uint) (int64, error) {
	return s.repo.CountUserActivities(userID)
}
