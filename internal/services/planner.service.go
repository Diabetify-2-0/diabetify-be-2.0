package services

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"

	"gorm.io/gorm"
)

type PlannerService interface {
	SaveGoal(goal *models.PlannerGoal) error
	GetLatestGoal(userID uint) (*models.PlannerGoal, error)
	GetGoalHistory(userID uint, limit int) ([]models.PlannerGoal, error)
	CompleteGoal(userID uint, goalID string) (*models.PlannerGoal, error)
	ArchiveGoal(userID uint, goalID string) (*models.PlannerGoal, error)
	RecordCheckIn(entry *models.PlannerCheckInEntry) error
	GetCheckInHistory(userID uint, goalID string, limit int) ([]models.PlannerCheckInEntry, error)
	GetLastCheckIns(userID uint, goalID string) (map[string]int64, error)
}

type plannerService struct {
	repo repository.PlannerRepository
}

func NewPlannerService(repo repository.PlannerRepository) PlannerService {
	return &plannerService{repo: repo}
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

func (s *plannerService) GetGoalHistory(userID uint, limit int) ([]models.PlannerGoal, error) {
	return s.repo.FindGoalsByUserID(userID, limit)
}

func (s *plannerService) CompleteGoal(userID uint, goalID string) (*models.PlannerGoal, error) {
	return s.repo.UpdateGoalStatus(userID, goalID, models.PlannerGoalStatusCompleted)
}

func (s *plannerService) ArchiveGoal(userID uint, goalID string) (*models.PlannerGoal, error) {
	return s.repo.UpdateGoalStatus(userID, goalID, models.PlannerGoalStatusArchived)
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
