package models

import (
	"time"

	"gorm.io/gorm"
)

type CounterfactualJob struct {
	ID             string         `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID         uint           `gorm:"not null;index" json:"user_id"`
	Status         string         `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	ReasonCode     *string        `gorm:"type:varchar(100)" json:"reason_code,omitempty"`
	ErrorMessage   *string        `gorm:"type:text" json:"error_message,omitempty"`
	RequestPayload *string        `gorm:"type:text" json:"request_payload,omitempty"`
	ResultPayload  *string        `gorm:"type:text" json:"result_payload,omitempty"`
	CreatedAt      time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

const (
	CFJobStatusPending    = "pending"
	CFJobStatusSubmitted  = "submitted"
	CFJobStatusCompleted  = "completed"
	CFJobStatusInfeasible = "infeasible"
	CFJobStatusFailed     = "failed"
	CFJobStatusCancelled  = "cancelled"
)

func (cj *CounterfactualJob) TableName() string {
	return "counterfactual_jobs"
}
