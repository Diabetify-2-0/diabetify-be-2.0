package services

import (
	"context"
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"net/http"

	"gorm.io/gorm"
)

type PlannerService interface {
	SaveGoal(goal *models.PlannerGoal) error
	GetLatestGoal(userID uint) (*models.PlannerGoal, error)
	GetActiveCoach(ctx context.Context, userID uint) (*models.PlannerCoachResponse, error)
	DeleteGoal(userID uint, goalID string) error
	RecordCheckIn(entry *models.PlannerCheckInEntry) error
	GetCheckInHistory(userID uint, goalID string, limit int) ([]models.PlannerCheckInEntry, error)
	GetLastCheckIns(userID uint, goalID string) (map[string]int64, error)
}

type plannerService struct {
	repo         repository.PlannerRepository
	profileRepo  repository.UserProfileRepository
	activityRepo repository.ActivityRepository
	httpClient   *http.Client
}

func NewPlannerService(
	repo repository.PlannerRepository,
	profileRepo repository.UserProfileRepository,
	activityRepo repository.ActivityRepository,
) PlannerService {
	return &plannerService{
		repo:         repo,
		profileRepo:  profileRepo,
		activityRepo: activityRepo,
		httpClient: &http.Client{
			Timeout: plannerCoachTimeout(),
		},
	}
}

func (s *plannerService) SaveGoal(goal *models.PlannerGoal) error {
	if goal.Status == "" {
		goal.Status = models.PlannerGoalStatusActive
	}
	return s.repo.SaveGoal(goal)
}

func (s *plannerService) GetLatestGoal(userID uint) (*models.PlannerGoal, error) {
	goal, err := s.repo.FindLatestGoalByUserID(userID)
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return goal, err
}

func (s *plannerService) DeleteGoal(userID uint, goalID string) error {
	return s.repo.DeleteGoal(userID, goalID)
}

func (s *plannerService) RecordCheckIn(entry *models.PlannerCheckInEntry) error {
	goal, err := s.repo.FindGoalByID(entry.GoalID)
	if err != nil {
		return err
	}
	if goal.UserID != entry.UserID {
		return gorm.ErrRecordNotFound
	}
	return s.repo.CreateCheckIn(entry)
}

func (s *plannerService) GetCheckInHistory(userID uint, goalID string, limit int) ([]models.PlannerCheckInEntry, error) {
	return s.repo.FindCheckInsByGoalID(userID, goalID, limit)
}

func (s *plannerService) GetLastCheckIns(userID uint, goalID string) (map[string]int64, error) {
	return s.repo.FindLastCheckInsByGoalID(userID, goalID)
}
