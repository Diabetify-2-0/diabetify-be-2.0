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
	DeleteGoal(userID uint, goalID string) error
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
	save := func(db *gorm.DB) error {
		return db.Transaction(func(tx *gorm.DB) error {
			var oldGoalIDs []string
			oldActiveGoals := tx.Model(&models.PlannerGoal{}).Where(
				"user_id = ? AND status = ? AND id <> ?",
				goal.UserID,
				models.PlannerGoalStatusActive,
				goal.ID,
			)
			if err := oldActiveGoals.Pluck("id", &oldGoalIDs).Error; err != nil {
				return err
			}
			if len(oldGoalIDs) > 0 {
				if err := tx.Unscoped().
					Where("user_id = ? AND goal_id IN ?", goal.UserID, oldGoalIDs).
					Delete(&models.PlannerCheckInEntry{}).Error; err != nil {
					return err
				}
				if err := tx.Unscoped().
					Where("user_id = ? AND id IN ?", goal.UserID, oldGoalIDs).
					Delete(&models.PlannerGoal{}).Error; err != nil {
					return err
				}
			}

			return tx.Save(goal).Error
		})
	}

	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(goal.UserID), func(db *gorm.DB) error {
			return save(db)
		})
	}

	return save(r.db)
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
		return db.Where("user_id = ? AND status = ?", userID, models.PlannerGoalStatusActive).
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

func (r *plannerRepository) DeleteGoal(userID uint, goalID string) error {
	deleteGoal := func(db *gorm.DB) error {
		return db.Transaction(func(tx *gorm.DB) error {
			result := tx.Unscoped().
				Where("user_id = ? AND id = ?", userID, goalID).
				Delete(&models.PlannerGoal{})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
			return tx.Unscoped().
				Where("user_id = ? AND goal_id = ?", userID, goalID).
				Delete(&models.PlannerCheckInEntry{}).Error
		})
	}

	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(userID), deleteGoal)
	}

	return deleteGoal(r.db)
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
