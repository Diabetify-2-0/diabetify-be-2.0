package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"diabetify/internal/models"
	"diabetify/internal/repository"
)

func TestPlannerCoachRequestOmitsCounterfactualAndIncludesMilestoneProgress(t *testing.T) {
	goal := samplePlannerGoal()
	profile := sampleUserProfile()
	checkIns := sampleCheckIns()
	service := &plannerService{}
	progress := buildPlannerCoachMilestoneProgress(service, goal, checkIns, profile, time.Now().UTC())

	payload := plannerCoachRequest{
		UserID:            "123",
		Goal:              buildPlannerCoachGoalPayload(goal),
		RecentCheckIns:    checkIns,
		LastCheckIns:      map[string]int64{"weight": checkIns[0].CreatedAtMillis},
		UserProfile:       profile,
		MilestoneProgress: progress,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	body := string(raw)
	if strings.Contains(body, "counterfactual_result") {
		t.Fatalf("unexpected counterfactual_result in payload: %s", body)
	}
	if !strings.Contains(body, "milestone_progress") {
		t.Fatalf("expected milestone_progress in payload: %s", body)
	}
}

func TestBuildPlannerCoachMilestoneProgressMirrorsBMIProgress(t *testing.T) {
	goal := samplePlannerGoal()
	profile := sampleUserProfile()
	checkIns := sampleCheckIns()

	service := &plannerService{}
	progress := buildPlannerCoachMilestoneProgress(service, goal, checkIns, profile, time.UnixMilli(goal.CreatedAtMillis).Add(14*24*time.Hour))
	if progress == nil {
		t.Fatal("expected milestone progress")
	}
	if progress.ProgressWeek != 2 {
		t.Fatalf("expected progress week 2, got %d", progress.ProgressWeek)
	}
	if len(progress.Items) != 1 {
		t.Fatalf("expected 1 milestone item, got %d", len(progress.Items))
	}

	item := progress.Items[0]
	if item.FeatureName != "BMI" {
		t.Fatalf("expected BMI feature, got %s", item.FeatureName)
	}
	if item.CurrentText == "-" || item.TargetText == "-" || item.ExpectedText == "-" {
		t.Fatalf("expected formatted texts, got current=%q target=%q expected=%q", item.CurrentText, item.TargetText, item.ExpectedText)
	}
	if item.Status == "" {
		t.Fatal("expected milestone status")
	}
	if item.ProgressPercentage <= 0 {
		t.Fatalf("expected positive progress percentage, got %d", item.ProgressPercentage)
	}
	if item.LatestCheckInLabel == "" || item.LatestCheckInValue == "" {
		t.Fatalf("expected latest check-in context, got label=%q value=%q", item.LatestCheckInLabel, item.LatestCheckInValue)
	}
}

func TestBuildPlannerCoachMilestoneProgressUsesActivityHistoryForPhysicalActivity(t *testing.T) {
	baseline := 1.0
	target := 5.0
	goal := &models.PlannerGoal{
		ID:                   "goal-activity",
		UserID:               123,
		Status:               models.PlannerGoalStatusActive,
		TargetRiskPercentage: 30,
		DurationWeeks:        12,
		CreatedAtMillis:      time.Now().UTC().AddDate(0, 0, -13).UnixMilli(),
		Features: []models.PlannerGoalFeature{
			{
				FeatureName:   "moderate_physical_activity_frequency",
				Label:         "Aktivitas Fisik",
				BaselineValue: &baseline,
				TargetValue:   &target,
				BaselineText:  "1 hari/minggu",
				TargetText:    "5 hari/minggu",
			},
		},
	}
	profile := sampleUserProfile()
	profile.CreatedAt = time.Now().UTC().AddDate(0, 0, -20)
	profile.PhysicalActivityFrequency = intPtr(1)

	service := &plannerService{
		activityRepo: fakeActivityRepository{
			activitiesByType: []models.Activity{
				{Value: 1},
				{Value: 1},
				{Value: 1},
			},
		},
	}
	progress := buildPlannerCoachMilestoneProgress(service, goal, nil, profile, time.Now().UTC())
	if progress == nil || len(progress.Items) != 1 {
		t.Fatalf("expected one activity milestone, got %+v", progress)
	}
	item := progress.Items[0]
	if item.CurrentValue == nil || *item.CurrentValue != 3 {
		t.Fatalf("expected activity current value from activity history = 3, got %+v", item.CurrentValue)
	}
	if item.CurrentText != "3 hari/minggu" {
		t.Fatalf("expected user-facing activity text, got %q", item.CurrentText)
	}
}

func TestBuildPlannerCoachMilestoneProgressMarksNumericMilestoneBehindUntilWeeklyTargetReached(t *testing.T) {
	baselineBMI := bmiFromWeightValue(83, 172)
	targetBMI := bmiFromWeightValue(73, 172)
	goal := &models.PlannerGoal{
		ID:                   "goal-weekly-bmi",
		UserID:               123,
		Status:               models.PlannerGoalStatusActive,
		TargetRiskPercentage: 30,
		DurationWeeks:        4,
		CreatedAtMillis:      time.Now().UTC().AddDate(0, 0, -14).UnixMilli(),
		Features: []models.PlannerGoalFeature{
			{
				FeatureName:   "BMI",
				Label:         "BMI",
				BaselineValue: &baselineBMI,
				TargetValue:   &targetBMI,
				BaselineText:  "83 kg",
				TargetText:    "73 kg",
				ActionLabel:   "Turunkan berat secara bertahap",
			},
		},
	}
	service := &plannerService{}
	scenarioNow := time.UnixMilli(goal.CreatedAtMillis).Add(8 * 24 * time.Hour)

	cases := []struct {
		weightKg       int
		expectedStatus string
		expectedText   string
	}{
		{weightKg: 82, expectedStatus: "BEHIND", expectedText: "82 kg"},
		{weightKg: 81, expectedStatus: "BEHIND", expectedText: "81 kg"},
		{weightKg: 80, expectedStatus: "BEHIND", expectedText: "80 kg"},
		{weightKg: 79, expectedStatus: "BEHIND", expectedText: "79 kg"},
		{weightKg: 78, expectedStatus: "ACHIEVED", expectedText: "78 kg"},
		{weightKg: 77, expectedStatus: "ACHIEVED", expectedText: "77 kg"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%dkg", tc.weightKg), func(t *testing.T) {
			profile := sampleUserProfile()
			profile.Weight = intPtr(tc.weightKg)
			profile.BMI = bmiFromWeight(tc.weightKg, 172)
			checkIns := []models.PlannerCheckInEntry{
				{
					ID:              fmt.Sprintf("checkin-%d", tc.weightKg),
					UserID:          123,
					GoalID:          goal.ID,
					Type:            "weight",
					Label:           "Berat Mingguan",
					ValueText:       fmt.Sprintf("%d kg", tc.weightKg),
					CreatedAtMillis: scenarioNow.UnixMilli(),
				},
			}

			progress := buildPlannerCoachMilestoneProgress(service, goal, checkIns, profile, scenarioNow)
			if progress == nil || len(progress.Items) != 1 {
				t.Fatalf("expected one milestone item, got %+v", progress)
			}

			item := progress.Items[0]
			if item.ExpectedText != "78 kg" {
				t.Fatalf("expected weekly target 78 kg, got %q", item.ExpectedText)
			}
			if item.CurrentText != tc.expectedText {
				t.Fatalf("expected current text %q, got %q", tc.expectedText, item.CurrentText)
			}
			if item.Status != tc.expectedStatus {
				t.Fatalf("expected status %s at %d kg, got %s", tc.expectedStatus, tc.weightKg, item.Status)
			}
		})
	}
}

func samplePlannerGoal() *models.PlannerGoal {
	baselineBMI := 28.06
	targetBMI := 22.86
	currentRisk := 60.3
	projectedRisk := 47.4
	return &models.PlannerGoal{
		ID:                      "goal-1",
		UserID:                  123,
		Title:                   "Turunkan risiko ke bawah 30%",
		Status:                  models.PlannerGoalStatusActive,
		CurrentRiskPercentage:   &currentRisk,
		TargetRiskPercentage:    30,
		DurationWeeks:           12,
		ProjectedRiskPercentage: &projectedRisk,
		CreatedAtMillis:         time.Now().UTC().Add(-8 * 24 * time.Hour).UnixMilli(),
		Features: []models.PlannerGoalFeature{
			{
				FeatureName:   "BMI",
				Label:         "BMI",
				BaselineValue: &baselineBMI,
				TargetValue:   &targetBMI,
				BaselineText:  "BMI 28.1",
				TargetText:    "BMI 22.9",
				ActionLabel:   "Turunkan berat sekitar 15.9 kg",
			},
		},
		ActionSteps: []string{"Catat berat badan mingguan."},
	}
}

func sampleUserProfile() *models.UserProfile {
	weight := 81
	height := 172
	bmi := 27.38
	activity := 2
	smoking := 2
	hypertension := true
	cholesterol := true
	return &models.UserProfile{
		UserID:                    123,
		Weight:                    &weight,
		Height:                    &height,
		BMI:                       &bmi,
		PhysicalActivityFrequency: &activity,
		Smoking:                   &smoking,
		Hypertension:              &hypertension,
		Cholesterol:               &cholesterol,
	}
}

func intPtr(v int) *int {
	return &v
}

func bmiFromWeight(weightKg int, heightCm int) *float64 {
	heightMeters := float64(heightCm) / 100.0
	if heightMeters <= 0 {
		return nil
	}
	bmi := float64(weightKg) / (heightMeters * heightMeters)
	return &bmi
}

func bmiFromWeightValue(weightKg int, heightCm int) float64 {
	heightMeters := float64(heightCm) / 100.0
	return float64(weightKg) / (heightMeters * heightMeters)
}

func sampleCheckIns() []models.PlannerCheckInEntry {
	return []models.PlannerCheckInEntry{
		{
			ID:              "checkin-1",
			UserID:          123,
			GoalID:          "goal-1",
			Type:            "weight",
			Label:           "Berat Mingguan",
			ValueText:       "84 kg",
			Note:            "Berat turun sedikit",
			CreatedAtMillis: time.Now().UTC().Add(-24 * time.Hour).UnixMilli(),
		},
	}
}

type fakeActivityRepository struct {
	repository.ActivityRepository
	activitiesByType []models.Activity
}

func (f fakeActivityRepository) GetActivitiesByUserIDAndTypeAndDateRange(userID uint, activityType string, startDate, endDate time.Time) ([]models.Activity, error) {
	return f.activitiesByType, nil
}
