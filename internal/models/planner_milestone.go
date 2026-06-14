package models

type PlannerMilestoneProgress struct {
	ProgressWeek  int                    `json:"progress_week"`
	DurationWeeks int                    `json:"duration_weeks"`
	Items         []PlannerMilestoneItem `json:"items"`
}

type PlannerMilestoneItem struct {
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
