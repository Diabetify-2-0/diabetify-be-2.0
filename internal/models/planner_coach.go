package models

import "time"

type PlannerCoachResponse struct {
	GoalID            string                         `json:"goal_id"`
	Headline          string                         `json:"headline"`
	Summary           string                         `json:"summary"`
	FocusThisWeek     []string                       `json:"focus_this_week"`
	ActionSteps       []string                       `json:"action_steps"`
	MonitoringPoints  []string                       `json:"monitoring_points"`
	Warnings          []string                       `json:"warnings"`
	MilestoneProgress *PlannerCoachMilestoneProgress `json:"milestone_progress,omitempty"`
	GeneratedBy       string                         `json:"generated_by"`
	FallbackUsed      bool                           `json:"fallback_used"`
	GeneratedAt       time.Time                      `json:"generated_at"`
	ProgressWeek      int                            `json:"progress_week"`
	DurationWeeks     int                            `json:"duration_weeks"`
	CheckInCount      int                            `json:"check_in_count"`
	SourceJobID       *string                        `json:"source_job_id,omitempty"`
	ProjectedRiskNote string                         `json:"projected_risk_note,omitempty"`
}

type PlannerCoachMilestoneProgress struct {
	ProgressWeek  int                         `json:"progress_week"`
	DurationWeeks int                         `json:"duration_weeks"`
	Items         []PlannerCoachMilestoneItem `json:"items"`
}

type PlannerCoachMilestoneItem struct {
	FeatureName        string   `json:"feature_name"`
	Label              string   `json:"label"`
	Status             string   `json:"status"`
	BaselineValue      *float64 `json:"baseline_value,omitempty"`
	TargetValue        *float64 `json:"target_value,omitempty"`
	CurrentValue       *float64 `json:"current_value,omitempty"`
	BaselineText       string   `json:"baseline_text"`
	TargetText         string   `json:"target_text"`
	CurrentText        string   `json:"current_text"`
	ExpectedText       string   `json:"expected_text"`
	ProgressFraction   float64  `json:"progress_fraction"`
	ProgressPercentage int      `json:"progress_percentage"`
	LatestCheckInLabel string   `json:"latest_check_in_label,omitempty"`
	LatestCheckInValue string   `json:"latest_check_in_value,omitempty"`
}
