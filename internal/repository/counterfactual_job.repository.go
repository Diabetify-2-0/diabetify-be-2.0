package repository

import (
	"diabetify/database"
	"diabetify/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type CounterfactualJobRepository interface {
	SaveJob(job *models.CounterfactualJob) error
	GetJobByID(id string) (*models.CounterfactualJob, error)
	GetJobsByUserID(userID uint, limit int) ([]*models.CounterfactualJob, error)
	UpdateJobSubmissionStatus(jobID string) error
	UpdateJobTerminalStatus(jobID, status string, reasonCode, errorMessage, resultPayload *string) error
	CancelJob(jobID string) error
}

type counterfactualJobRepository struct {
	db        *gorm.DB
	useShards bool
}

func NewCounterfactualJobRepository(db *gorm.DB) CounterfactualJobRepository {
	return &counterfactualJobRepository{
		db:        db,
		useShards: db == nil,
	}
}

func (r *counterfactualJobRepository) SaveJob(job *models.CounterfactualJob) error {
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	job.UpdatedAt = time.Now()

	if r.useShards {
		return database.Manager.ExecuteOnUserShard(int(job.UserID), func(db *gorm.DB) error {
			return db.Create(job).Error
		})
	}

	return r.db.Create(job).Error
}

func (r *counterfactualJobRepository) GetJobByID(id string) (*models.CounterfactualJob, error) {
	if r.useShards {
		var foundJob *models.CounterfactualJob

		shards := database.Manager.GetAllShards()
		for _, db := range shards {
			var job models.CounterfactualJob
			err := db.Where("id = ?", id).First(&job).Error
			if err == nil {
				foundJob = &job
				break
			}
			if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		}

		if foundJob == nil {
			return nil, gorm.ErrRecordNotFound
		}

		return foundJob, nil
	}

	var job models.CounterfactualJob
	if err := r.db.Where("id = ?", id).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *counterfactualJobRepository) GetJobsByUserID(userID uint, limit int) ([]*models.CounterfactualJob, error) {
	if limit <= 0 {
		limit = 10
	}

	if r.useShards {
		var jobs []*models.CounterfactualJob
		err := database.Manager.ExecuteOnUserShard(int(userID), func(db *gorm.DB) error {
			return db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&jobs).Error
		})
		return jobs, err
	}

	var jobs []*models.CounterfactualJob
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&jobs).Error
	return jobs, err
}

func (r *counterfactualJobRepository) UpdateJobSubmissionStatus(jobID string) error {
	if r.useShards {
		job, err := r.GetJobByID(jobID)
		if err != nil {
			return err
		}

		return database.Manager.ExecuteOnUserShard(int(job.UserID), func(db *gorm.DB) error {
			return updateJobFields(db, jobID, map[string]interface{}{
				"status":       models.CFJobStatusSubmitted,
				"updated_at":   time.Now(),
				"completed_at": nil,
			})
		})
	}

	return updateJobFields(r.db, jobID, map[string]interface{}{
		"status":       models.CFJobStatusSubmitted,
		"updated_at":   time.Now(),
		"completed_at": nil,
	})
}

func (r *counterfactualJobRepository) UpdateJobTerminalStatus(
	jobID, status string,
	reasonCode, errorMessage, resultPayload *string,
) error {
	if status != models.CFJobStatusCompleted && status != models.CFJobStatusInfeasible && status != models.CFJobStatusFailed && status != models.CFJobStatusCancelled {
		return fmt.Errorf("invalid terminal status: %s", status)
	}

	updates := map[string]interface{}{
		"status":       status,
		"updated_at":   time.Now(),
		"completed_at": time.Now(),
	}
	if reasonCode != nil {
		updates["reason_code"] = *reasonCode
	}
	if errorMessage != nil {
		updates["error_message"] = *errorMessage
	}
	if resultPayload != nil {
		updates["result_payload"] = *resultPayload
	}

	if r.useShards {
		job, err := r.GetJobByID(jobID)
		if err != nil {
			return err
		}
		return database.Manager.ExecuteOnUserShard(int(job.UserID), func(db *gorm.DB) error {
			return updateJobFields(db, jobID, updates)
		})
	}

	return updateJobFields(r.db, jobID, updates)
}

func (r *counterfactualJobRepository) CancelJob(jobID string) error {
	if r.useShards {
		job, err := r.GetJobByID(jobID)
		if err != nil {
			return err
		}
		return database.Manager.ExecuteOnUserShard(int(job.UserID), func(db *gorm.DB) error {
			return cancelJobByID(db, jobID)
		})
	}
	return cancelJobByID(r.db, jobID)
}

func cancelJobByID(db *gorm.DB, jobID string) error {
	return updateJobFields(db, jobID, map[string]interface{}{
		"status":       models.CFJobStatusCancelled,
		"updated_at":   time.Now(),
		"completed_at": time.Now(),
	})
}

func updateJobFields(db *gorm.DB, jobID string, updates map[string]interface{}) error {
	result := db.Model(&models.CounterfactualJob{}).Where("id = ?", jobID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("job with ID %s not found", jobID)
	}
	return nil
}
