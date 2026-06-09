package services

import (
	"diabetify/internal/models"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	plannerWeekMillis = 7 * 24 * time.Hour
)

var plannerNumberPattern = regexp.MustCompile(`-?\d+(?:\.\d+)?`)

func buildPlannerCoachMilestoneProgress(
	service *plannerService,
	goal *models.PlannerGoal,
	checkIns []models.PlannerCheckInEntry,
	profile *models.UserProfile,
	now time.Time,
) *models.PlannerCoachMilestoneProgress {
	if goal == nil {
		return nil
	}

	progressWeek := plannerProgressWeek(goal, now)
	items := make([]models.PlannerCoachMilestoneItem, 0, len(goal.Features))
	for _, feature := range goal.Features {
		item, ok := buildPlannerCoachMilestoneItem(service, goal.UserID, feature, progressWeek, goal.DurationWeeks, profile, checkIns, now)
		if !ok {
			continue
		}
		items = append(items, item)
	}

	return &models.PlannerCoachMilestoneProgress{
		ProgressWeek:  progressWeek,
		DurationWeeks: goal.DurationWeeks,
		Items:         items,
	}
}

func buildPlannerCoachMilestoneItem(
	service *plannerService,
	userID uint,
	feature models.PlannerGoalFeature,
	currentWeek int,
	totalWeeks int,
	profile *models.UserProfile,
	history []models.PlannerCheckInEntry,
	now time.Time,
) (models.PlannerCoachMilestoneItem, bool) {
	baseline := feature.BaselineValue
	target := feature.TargetValue
	if baseline == nil || target == nil {
		return models.PlannerCoachMilestoneItem{}, false
	}

	label := plannerDisplayFeatureLabel(feature)
	currentValue := plannerCurrentFeatureValue(service, userID, feature.FeatureName, profile, now)
	currentText := plannerFormatFeatureValue(feature.FeatureName, currentValue, profileHeightCm(profile))
	targetText := plannerFormatFeatureValue(feature.FeatureName, target, profileHeightCm(profile))
	baselineText := plannerFormatFeatureValue(feature.FeatureName, baseline, profileHeightCm(profile))
	if baselineText == "-" && strings.TrimSpace(feature.BaselineText) != "" {
		baselineText = feature.BaselineText
	}
	if targetText == "-" && strings.TrimSpace(feature.TargetText) != "" {
		targetText = feature.TargetText
	}
	latestCheckIn := plannerLatestRelevantCheckIn(feature.FeatureName, history)

	item := models.PlannerCoachMilestoneItem{
		FeatureName:      feature.FeatureName,
		Label:            label,
		BaselineValue:    baseline,
		TargetValue:      target,
		CurrentValue:     currentValue,
		BaselineText:     baselineText,
		TargetText:       targetText,
		CurrentText:      currentText,
		ExpectedText:     targetText,
		ProgressFraction: 0,
	}
	if latestCheckIn != nil {
		item.LatestCheckInLabel = latestCheckIn.Label
		item.LatestCheckInValue = latestCheckIn.ValueText
	}

	if plannerIsCategoricalFeature(feature.FeatureName) {
		reached := plannerIsTargetReached(feature.FeatureName, *baseline, *target, currentValue)
		hasUpdate := latestCheckIn != nil
		switch {
		case reached:
			item.Status = "ACHIEVED"
			item.ExpectedText = plannerCategoricalMilestoneFocusText(feature.FeatureName, true)
			item.ProgressFraction = 1
		case hasUpdate:
			item.Status = "MONITOR"
			item.ExpectedText = plannerCategoricalMilestoneFocusText(feature.FeatureName, false)
			item.ProgressFraction = 0.5
		default:
			item.Status = "MONITOR"
			item.ExpectedText = plannerCategoricalMilestoneFocusText(feature.FeatureName, false)
			item.ProgressFraction = 0
		}
		item.ProgressPercentage = int(math.Round(item.ProgressFraction * 100))
		return item, true
	}

	expectedFraction := float64(currentWeek) / float64(maxInt(totalWeeks, 1))
	expectedValue := *baseline + ((*target - *baseline) * expectedFraction)
	item.ExpectedText = plannerFormatFeatureValue(feature.FeatureName, &expectedValue, profileHeightCm(profile))
	progressFraction := plannerCalculateProgressFraction(*baseline, *target, currentValue)
	weeklyTargetReached := plannerHasReachedDisplayedNumericTarget(
		feature.FeatureName,
		baselineText,
		currentText,
		item.ExpectedText,
	)
	reached := plannerHasReachedDisplayedNumericTarget(feature.FeatureName, baselineText, currentText, targetText) ||
		plannerIsTargetReached(feature.FeatureName, *baseline, *target, currentValue)

	resolvedFraction := progressFraction
	if displayedFraction, ok := plannerDisplayedNumericProgressFraction(feature.FeatureName, baselineText, currentText, targetText); ok {
		resolvedFraction = displayedFraction
	}

	switch {
	case latestCheckIn == nil:
		item.Status = "MONITOR"
		item.ProgressFraction = 0
	case reached:
		item.Status = "ACHIEVED"
		item.ProgressFraction = 1
	case weeklyTargetReached:
		item.Status = "ACHIEVED"
		item.ProgressFraction = resolvedFraction
	default:
		item.Status = "BEHIND"
		item.ProgressFraction = resolvedFraction
	}
	item.ProgressFraction = clampFloat(item.ProgressFraction, 0, 1)
	item.ProgressPercentage = int(math.Round(item.ProgressFraction * 100))
	return item, true
}

func plannerProgressWeek(goal *models.PlannerGoal, now time.Time) int {
	if goal == nil || goal.DurationWeeks <= 0 {
		return 1
	}
	start := time.UnixMilli(goal.CreatedAtMillis)
	if start.IsZero() || now.Before(start) {
		return 1
	}

	elapsed := now.Sub(start)
	weeks := int(math.Ceil(float64(elapsed) / float64(plannerWeekMillis)))
	if weeks < 1 {
		weeks = 1
	}
	if weeks > goal.DurationWeeks {
		weeks = goal.DurationWeeks
	}
	return weeks
}

func plannerCurrentFeatureValue(
	service *plannerService,
	userID uint,
	featureName string,
	profile *models.UserProfile,
	now time.Time,
) *float64 {
	if profile == nil {
		return nil
	}

	switch featureName {
	case "BMI":
		return profile.BMI
	case "moderate_physical_activity_frequency":
		value, ok := plannerResolvedPhysicalActivityFrequency(service, userID, profile, now)
		if !ok {
			return nil
		}
		return &value
	case "smoking_status":
		if profile.Smoking == nil {
			return nil
		}
		value := float64(*profile.Smoking)
		return &value
	case "is_hypertension":
		if profile.Hypertension == nil {
			return nil
		}
		value := 0.0
		if *profile.Hypertension {
			value = 1.0
		}
		return &value
	case "is_cholesterol":
		if profile.Cholesterol == nil {
			return nil
		}
		value := 0.0
		if *profile.Cholesterol {
			value = 1.0
		}
		return &value
	case "is_bloodline":
		if profile.Bloodline == nil {
			return nil
		}
		value := 0.0
		if *profile.Bloodline {
			value = 1.0
		}
		return &value
	case "is_macrosomic_baby":
		if profile.MacrosomicBaby == nil {
			return nil
		}
		value := float64(*profile.MacrosomicBaby)
		return &value
	default:
		return nil
	}
}

func plannerResolvedPhysicalActivityFrequency(
	service *plannerService,
	userID uint,
	profile *models.UserProfile,
	now time.Time,
) (float64, bool) {
	if service == nil || profile == nil {
		return 0, false
	}

	endDate := now
	startDate := endDate.AddDate(0, 0, -7)
	if profile.CreatedAt.Before(startDate) && service.activityRepo != nil {
		activities, err := service.activityRepo.GetActivitiesByUserIDAndTypeAndDateRange(userID, "workout", startDate, endDate)
		if err == nil {
			totalFrequency := 0
			for _, activity := range activities {
				totalFrequency += activity.Value
			}
			return float64(totalFrequency), true
		}
	}

	if profile.PhysicalActivityFrequency != nil {
		return float64(*profile.PhysicalActivityFrequency), true
	}
	return 0, false
}

func plannerLatestRelevantCheckIn(featureName string, history []models.PlannerCheckInEntry) *models.PlannerCheckInEntry {
	checkInType := ""
	switch featureName {
	case "BMI":
		checkInType = "weight"
	case "moderate_physical_activity_frequency":
		checkInType = "activity"
	case "is_hypertension":
		checkInType = "hypertension"
	case "is_cholesterol":
		checkInType = "cholesterol"
	case "smoking_status":
		checkInType = "smoking"
	default:
		return nil
	}

	var latest *models.PlannerCheckInEntry
	for i := range history {
		entry := history[i]
		if entry.Type != checkInType {
			continue
		}
		if latest == nil || entry.CreatedAtMillis > latest.CreatedAtMillis {
			copied := entry
			latest = &copied
		}
	}
	return latest
}

func plannerDisplayFeatureLabel(feature models.PlannerGoalFeature) string {
	if feature.FeatureName == "BMI" {
		return "Berat badan"
	}
	return sanitizePlannerText(feature.Label)
}

func plannerFormatFeatureValue(name string, value *float64, heightCm int) string {
	if value == nil {
		return "-"
	}

	switch name {
	case "BMI":
		if weight := plannerBMIToWeight(*value, heightCm); weight != nil {
			return fmt.Sprintf("%.0f kg", *weight)
		}
		return "-"
	case "age":
		return fmt.Sprintf("%d tahun", int(*value))
	case "moderate_physical_activity_frequency":
		return fmt.Sprintf("%d hari/minggu", int(*value))
	case "smoking_status":
		switch int(*value) {
		case 0:
			return "Tidak pernah"
		case 1:
			return "Sudah berhenti"
		case 2:
			return "Masih aktif"
		default:
			return fmt.Sprintf("%d", int(*value))
		}
	case "brinkman_index":
		switch int(*value) {
		case 0:
			return "Sangat rendah"
		case 1:
			return "Ringan"
		case 2:
			return "Sedang"
		case 3:
			return "Tinggi"
		default:
			return fmt.Sprintf("%d", int(*value))
		}
	case "is_hypertension":
		if int(*value) == 1 {
			return "Belum terkontrol"
		}
		return "Terkontrol"
	case "is_cholesterol":
		if int(*value) == 1 {
			return "Belum terkontrol"
		}
		return "Terkontrol"
	case "is_bloodline":
		if int(*value) == 1 {
			return "Ya"
		}
		return "Tidak"
	case "is_macrosomic_baby":
		switch int(*value) {
		case 0:
			return "Tidak"
		case 1:
			return "Ya"
		case 2:
			return "Tidak relevan"
		default:
			return fmt.Sprintf("%d", int(*value))
		}
	default:
		return fmt.Sprintf("%.2f", *value)
	}
}

func plannerBMIToWeight(bmi float64, heightCm int) *float64 {
	heightMeters := float64(heightCm) / 100.0
	if heightMeters <= 0 {
		return nil
	}
	weight := bmi * heightMeters * heightMeters
	return &weight
}

func plannerIsTargetReached(featureName string, baseline, target float64, current *float64) bool {
	if current == nil {
		return false
	}

	switch featureName {
	case "smoking_status", "is_hypertension", "is_cholesterol", "is_bloodline", "is_macrosomic_baby":
		return int(math.Round(*current)) == int(math.Round(target))
	default:
		tolerance := 0.05
		switch {
		case math.Abs(*current-target) <= tolerance:
			return true
		case target > baseline:
			return *current >= target
		case target < baseline:
			return *current <= target
		default:
			return false
		}
	}
}

func plannerCalculateProgressFraction(baseline, target float64, current *float64) float64 {
	if current == nil {
		return 0
	}
	distance := target - baseline
	if distance == 0 {
		if *current == target {
			return 1
		}
		return 0
	}
	return clampFloat((*current-baseline)/distance, 0, 1)
}

func plannerIsCategoricalFeature(featureName string) bool {
	switch featureName {
	case "smoking_status", "brinkman_index", "is_hypertension", "is_cholesterol", "is_bloodline", "is_macrosomic_baby":
		return true
	default:
		return false
	}
}

func plannerCategoricalMilestoneFocusText(featureName string, reached bool) string {
	switch featureName {
	case "is_hypertension":
		if reached {
			return "Pertahankan tetap terkontrol"
		}
		return "Pantau tekanan darah"
	case "is_cholesterol":
		if reached {
			return "Pertahankan tetap terkontrol"
		}
		return "Pantau status kolesterol"
	case "smoking_status":
		if reached {
			return "Pertahankan berhenti merokok"
		}
		return "Usahakan berhenti merokok"
	default:
		if reached {
			return "Pertahankan status target"
		}
		return "Pantau perubahan status"
	}
}

func plannerHasReachedDisplayedNumericTarget(featureName, baselineText, currentText, targetText string) bool {
	status, ok := plannerDisplayedNumericTargetStatus(featureName, baselineText, currentText, targetText)
	return ok && status != "BELOW"
}

func plannerDisplayedNumericProgressFraction(featureName, baselineText, currentText, targetText string) (float64, bool) {
	baseline, ok := plannerDisplayedNumericComparableValue(featureName, baselineText)
	if !ok {
		return 0, false
	}
	current, ok := plannerDisplayedNumericComparableValue(featureName, currentText)
	if !ok {
		return 0, false
	}
	target, ok := plannerDisplayedNumericComparableValue(featureName, targetText)
	if !ok {
		return 0, false
	}

	value := plannerCalculateProgressFraction(baseline, target, &current)
	return value, true
}

func plannerDisplayedNumericTargetStatus(featureName, baselineText, currentText, targetText string) (string, bool) {
	baseline, ok := plannerDisplayedNumericComparableValue(featureName, baselineText)
	if !ok {
		return "", false
	}
	current, ok := plannerDisplayedNumericComparableValue(featureName, currentText)
	if !ok {
		return "", false
	}
	target, ok := plannerDisplayedNumericComparableValue(featureName, targetText)
	if !ok {
		return "", false
	}

	switch {
	case target < baseline && current < target:
		return "OVER", true
	case target < baseline && current == target:
		return "EXACT", true
	case target < baseline:
		return "BELOW", true
	case target > baseline && current > target:
		return "OVER", true
	case target > baseline && current == target:
		return "EXACT", true
	case target > baseline:
		return "BELOW", true
	case current == target:
		return "EXACT", true
	default:
		return "BELOW", true
	}
}

func plannerDisplayedNumericComparableValue(featureName, text string) (float64, bool) {
	switch featureName {
	case "BMI", "moderate_physical_activity_frequency":
		match := plannerNumberPattern.FindString(text)
		if match == "" {
			return 0, false
		}
		value, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return 0, false
		}
		return value, true
	default:
		return 0, false
	}
}

func sanitizePlannerText(text string) string {
	replacer := strings.NewReplacer("BMI", "berat badan", "bmi", "berat badan", "IMT", "berat badan", "imt", "berat badan")
	return replacer.Replace(text)
}

func profileHeightCm(profile *models.UserProfile) int {
	if profile == nil || profile.Height == nil {
		return 0
	}
	return *profile.Height
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
