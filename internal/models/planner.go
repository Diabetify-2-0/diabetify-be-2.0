package models

import (
	"time"

	"gorm.io/gorm"
)

type PlannerGoalStatus string

const (
	PlannerGoalStatusActive PlannerGoalStatus = "ACTIVE"
)

type PlannerGoalFeature struct {
	FeatureName   string   `json:"feature_name"`
	Label         string   `json:"label"`
	BaselineValue *float64 `json:"baseline_value"`
	TargetValue   *float64 `json:"target_value"`
	BaselineText  string   `json:"baseline_text"`
	TargetText    string   `json:"target_text"`
	ActionLabel   string   `json:"action_label"`
}

type PlannerGoal struct {
	ID                      string               `gorm:"primaryKey;size:128" json:"id"`
	CreatedAt               time.Time            `json:"created_at"`
	UpdatedAt               time.Time            `json:"updated_at"`
	DeletedAt               gorm.DeletedAt       `gorm:"index" json:"-" swaggerignore:"true"`
	UserID                  uint                 `gorm:"index;not null" json:"user_id"`
	Title                   string               `gorm:"not null" json:"title"`
	Status                  PlannerGoalStatus    `gorm:"type:varchar(24);index;not null;default:ACTIVE" json:"status"`
	CurrentRiskPercentage   *float64             `json:"current_risk_percentage"`
	TargetRiskPercentage    int                  `gorm:"not null" json:"target_risk_percentage"`
	DurationWeeks           int                  `gorm:"not null;default:12" json:"duration_weeks"`
	ProjectedRiskPercentage *float64             `json:"projected_risk_percentage"`
	SourceJobID             *string              `gorm:"size:128;index" json:"source_job_id"`
	CreatedAtMillis         int64                `gorm:"not null" json:"created_at_millis"`
	Summary                 *string              `json:"summary"`
	ActionSteps             []string             `gorm:"serializer:json" json:"action_steps"`
	MonitoringPlan          []string             `gorm:"serializer:json" json:"monitoring_plan"`
	Features                []PlannerGoalFeature `gorm:"serializer:json" json:"features"`
}

func (g *PlannerGoal) GetShardKey() int {
	return int(g.UserID)
}

func (g *PlannerGoal) TableName() string {
	return "planner_goals"
}

type PlannerCheckInEntry struct {
	ID              string         `gorm:"primaryKey;size:160" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-" swaggerignore:"true"`
	UserID          uint           `gorm:"index;not null" json:"user_id"`
	GoalID          string         `gorm:"size:128;index;not null" json:"goal_id"`
	Type            string         `gorm:"size:64;index;not null" json:"type"`
	Label           string         `gorm:"not null" json:"label"`
	ValueText       string         `gorm:"not null" json:"value_text"`
	Note            string         `json:"note"`
	CreatedAtMillis int64          `gorm:"index;not null" json:"created_at_millis"`
}

func (e *PlannerCheckInEntry) GetShardKey() int {
	return int(e.UserID)
}

func (e *PlannerCheckInEntry) TableName() string {
	return "planner_check_in_entries"
}
