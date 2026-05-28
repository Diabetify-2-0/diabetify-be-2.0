package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"fmt"

	"gorm.io/gorm"
)

type PlannerRepository interface {
	SaveGoal(goal *models.PlannerGoal) error
	FindGoalByID(id string) (*models.PlannerGoal, error)
	FindLatestGoalByUserID(userID uint) (*models.PlannerGoal, error)
	FindGoalsByUserID(userID uint, limit int) ([]models.PlannerGoal, error)
	UpdateGoalStatus(userID uint, goalID string, status models.PlannerGoalStatus) (*models.PlannerGoal, error)
	CreateCheckIn(entry *models.PlannerCheckInEntry) error
	FindCheckInsByGoalID(userID uint, goalID string, limit int) ([]models.PlannerCheckInEntry, error)
	FindLastCheckInsByGoalID(userID uint, goalID string) (map[string]int64, error)
}

type plannerRepository struct {
	db        *gorm.DB
	useShards bool
}

func NewPlannerRepository(db *gorm.DB) PlannerRepository {
	return &plannerRepository{
		db:        db,
		useShards: db == nil,
	}
}

func NewShardedPlannerRepository() PlannerRepository {
	return &plannerRepository{
		db:        nil,
		useShards: true,
	}
}

func (r *plannerRepository) SaveGoal(goal *models.PlannerGoal) error {
	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(goal.UserID), func(db *gorm.DB) error {
			return db.Save(goal).Error
		})
	}

	return r.db.Save(goal).Error
}

func (r *plannerRepository) FindGoalByID(id string) (*models.PlannerGoal, error) {
	if r.useShards {
		for shardName, db := range database.Manager.GetAllShards() {
			var goal models.PlannerGoal
			err := db.First(&goal, "id = ?", id).Error
			if err == nil {
				return &goal, nil
			}
			if err != gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("error searching planner goal on shard %s: %v", shardName, err)
			}
		}
		return nil, gorm.ErrRecordNotFound
	}

	var goal models.PlannerGoal
	if err := r.db.First(&goal, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &goal, nil
}

func (r *plannerRepository) FindLatestGoalByUserID(userID uint) (*models.PlannerGoal, error) {
	var goal models.PlannerGoal
	query := func(db *gorm.DB) error {
		return db.Where("user_id = ? AND status <> ?", userID, models.PlannerGoalStatusArchived).
			Order("created_at_millis DESC").
			First(&goal).Error
	}

	if r.useShards {
		if err := database.Manager.ExecuteOnUserShard(int(userID), query); err != nil {
			return nil, err
		}
		return &goal, nil
	}

	if err := query(r.db); err != nil {
		return nil, err
	}
	return &goal, nil
}

func (r *plannerRepository) FindGoalsByUserID(userID uint, limit int) ([]models.PlannerGoal, error) {
	if limit <= 0 {
		limit = 20
	}

	var goals []models.PlannerGoal
	query := func(db *gorm.DB) error {
		return db.Where("user_id = ?", userID).
			Order("created_at_millis DESC").
			Limit(limit).
			Find(&goals).Error
	}

	if r.useShards {
		return goals, database.Manager.ExecuteOnUserShard(int(userID), query)
	}

	return goals, query(r.db)
}

func (r *plannerRepository) UpdateGoalStatus(userID uint, goalID string, status models.PlannerGoalStatus) (*models.PlannerGoal, error) {
	goal, err := r.FindGoalByID(goalID)
	if err != nil {
		return nil, err
	}
	if goal.UserID != userID {
		return nil, gorm.ErrRecordNotFound
	}

	goal.Status = status
	if err := r.SaveGoal(goal); err != nil {
		return nil, err
	}
	return goal, nil
}

func (r *plannerRepository) CreateCheckIn(entry *models.PlannerCheckInEntry) error {
	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(entry.UserID), func(db *gorm.DB) error {
			return db.Save(entry).Error
		})
	}

	return r.db.Save(entry).Error
}

func (r *plannerRepository) FindCheckInsByGoalID(userID uint, goalID string, limit int) ([]models.PlannerCheckInEntry, error) {
	if limit <= 0 {
		limit = 80
	}

	var entries []models.PlannerCheckInEntry
	query := func(db *gorm.DB) error {
		return db.Where("user_id = ? AND goal_id = ?", userID, goalID).
			Order("created_at_millis DESC").
			Limit(limit).
			Find(&entries).Error
	}

	if r.useShards {
		return entries, database.Manager.ExecuteOnUserShard(int(userID), query)
	}

	return entries, query(r.db)
}

func (r *plannerRepository) FindLastCheckInsByGoalID(userID uint, goalID string) (map[string]int64, error) {
	entries, err := r.FindCheckInsByGoalID(userID, goalID, 200)
	if err != nil {
		return nil, err
	}

	lastCheckIns := make(map[string]int64)
	for _, entry := range entries {
		if existing, ok := lastCheckIns[entry.Type]; !ok || entry.CreatedAtMillis > existing {
			lastCheckIns[entry.Type] = entry.CreatedAtMillis
		}
	}
	return lastCheckIns, nil
}
