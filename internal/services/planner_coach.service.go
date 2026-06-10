package services

import (
	"bytes"
	"context"
	"diabetify/internal/models"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

type plannerCoachRequest struct {
	UserID            string                                `json:"user_id"`
	Goal              *plannerCoachGoalPayload              `json:"goal"`
	RecentCheckIns    []models.PlannerCheckInEntry          `json:"recent_check_ins"`
	LastCheckIns      map[string]int64                      `json:"last_check_ins"`
	UserProfile       *models.UserProfile                   `json:"user_profile,omitempty"`
	MilestoneProgress *models.PlannerCoachMilestoneProgress `json:"milestone_progress,omitempty"`
}

type plannerCoachGoalPayload struct {
	ID                      string                      `json:"id"`
	UserID                  uint                        `json:"user_id"`
	Title                   string                      `json:"title"`
	Status                  models.PlannerGoalStatus    `json:"status"`
	CurrentRiskPercentage   *float64                    `json:"current_risk_percentage"`
	TargetRiskPercentage    int                         `json:"target_risk_percentage"`
	DurationWeeks           int                         `json:"duration_weeks"`
	ProjectedRiskPercentage *float64                    `json:"projected_risk_percentage"`
	SourceJobID             *string                     `json:"source_job_id,omitempty"`
	CreatedAtMillis         int64                       `json:"created_at_millis"`
	Summary                 *string                     `json:"summary"`
	MonitoringPlan          []string                    `json:"monitoring_plan"`
	Features                []models.PlannerGoalFeature `json:"features"`
}

type plannerCoachLLMResponse struct {
	Headline         string   `json:"headline"`
	Summary          string   `json:"summary"`
	FocusThisWeek    []string `json:"focus_this_week"`
	ActionSteps      []string `json:"action_steps"`
	MonitoringPoints []string `json:"monitoring_points"`
	Warnings         []string `json:"warnings"`
}

func plannerCoachTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CHATBOT_PLANNER_COACH_TIMEOUT_MS"))
	if raw == "" {
		return 12 * time.Second
	}

	ms, err := time.ParseDuration(raw + "ms")
	if err != nil || ms <= 0 {
		return 12 * time.Second
	}
	return ms
}

func plannerCoachPath() string {
	if path := strings.TrimSpace(os.Getenv("CHATBOT_PLANNER_COACH_PATH")); path != "" {
		if strings.HasPrefix(path, "/") {
			return path
		}
		return "/" + path
	}
	return "/api/v1/planner/coach"
}

func (s *plannerService) GetActiveCoach(ctx context.Context, userID uint) (*models.PlannerCoachResponse, error) {
	goal, err := s.GetLatestGoal(userID)
	if err != nil || goal == nil {
		return nil, err
	}

	checkIns, err := s.GetCheckInHistory(userID, goal.ID, 12)
	if err != nil {
		return nil, err
	}

	lastCheckIns, err := s.GetLastCheckIns(userID, goal.ID)
	if err != nil {
		return nil, err
	}

	profile, err := s.profileRepo.FindByUserID(userID)
	if err != nil {
		profile = nil
	}

	milestoneProgress := buildPlannerCoachMilestoneProgress(s, goal, checkIns, profile, time.Now().UTC())
	if response, err := s.generateCoachViaChatbot(ctx, userID, goal, checkIns, lastCheckIns, profile, milestoneProgress); err == nil && response != nil {
		return response, nil
	}

	return s.buildCoachFallback(goal, checkIns, milestoneProgress), nil
}

func (s *plannerService) generateCoachViaChatbot(
	ctx context.Context,
	userID uint,
	goal *models.PlannerGoal,
	checkIns []models.PlannerCheckInEntry,
	lastCheckIns map[string]int64,
	profile *models.UserProfile,
	milestoneProgress *models.PlannerCoachMilestoneProgress,
) (*models.PlannerCoachResponse, error) {
	requestBody := plannerCoachRequest{
		UserID:            fmt.Sprintf("%d", userID),
		Goal:              buildPlannerCoachGoalPayload(goal),
		RecentCheckIns:    checkIns,
		LastCheckIns:      lastCheckIns,
		UserProfile:       profile,
		MilestoneProgress: milestoneProgress,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		chatbotServiceURL()+plannerCoachPath(),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("XAI_INTERNAL_API_KEY"); key != "" {
		req.Header.Set("X-Internal-Key", key)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("planner coach service returned HTTP %d", resp.StatusCode)
	}

	var llm plannerCoachLLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llm); err != nil {
		return nil, err
	}

	return s.normalizeCoachResponse(goal, checkIns, milestoneProgress, llm), nil
}

func (s *plannerService) normalizeCoachResponse(
	goal *models.PlannerGoal,
	checkIns []models.PlannerCheckInEntry,
	milestoneProgress *models.PlannerCoachMilestoneProgress,
	llm plannerCoachLLMResponse,
) *models.PlannerCoachResponse {
	now := time.Now().UTC()
	progressWeek := plannerProgressWeek(goal, now)

	headline := strings.TrimSpace(llm.Headline)
	if headline == "" {
		headline = fallbackHeadline(goal, progressWeek)
	}
	summary := strings.TrimSpace(llm.Summary)
	if summary == "" {
		summary = fallbackSummary(goal, milestoneProgress, progressWeek, len(checkIns))
	}
	focus := dedupeStrings(llm.FocusThisWeek)
	if len(focus) == 0 {
		focus = fallbackFocus(goal, milestoneProgress)
	}
	actionSteps := dedupeStrings(llm.ActionSteps)
	if len(actionSteps) == 0 {
		actionSteps = fallbackActionSteps(goal, milestoneProgress)
	}
	monitoring := dedupeStrings(llm.MonitoringPoints)
	if len(monitoring) == 0 {
		monitoring = fallbackMonitoring(goal, milestoneProgress, len(checkIns))
	}
	warnings := dedupeStrings(llm.Warnings)
	if len(warnings) == 0 {
		warnings = fallbackWarnings(goal)
	}

	return &models.PlannerCoachResponse{
		GoalID:            goal.ID,
		Headline:          headline,
		Summary:           summary,
		FocusThisWeek:     focus,
		ActionSteps:       actionSteps,
		MonitoringPoints:  monitoring,
		Warnings:          warnings,
		MilestoneProgress: milestoneProgress,
		GeneratedBy:       "chatbot_service",
		FallbackUsed:      false,
		GeneratedAt:       now,
		ProgressWeek:      progressWeek,
		DurationWeeks:     goal.DurationWeeks,
		CheckInCount:      len(checkIns),
		SourceJobID:       goal.SourceJobID,
		ProjectedRiskNote: projectedRiskNote(goal),
	}
}

func (s *plannerService) buildCoachFallback(
	goal *models.PlannerGoal,
	checkIns []models.PlannerCheckInEntry,
	milestoneProgress *models.PlannerCoachMilestoneProgress,
) *models.PlannerCoachResponse {
	now := time.Now().UTC()
	progressWeek := plannerProgressWeek(goal, now)
	return &models.PlannerCoachResponse{
		GoalID:            goal.ID,
		Headline:          fallbackHeadline(goal, progressWeek),
		Summary:           fallbackSummary(goal, milestoneProgress, progressWeek, len(checkIns)),
		FocusThisWeek:     fallbackFocus(goal, milestoneProgress),
		ActionSteps:       fallbackActionSteps(goal, milestoneProgress),
		MonitoringPoints:  fallbackMonitoring(goal, milestoneProgress, len(checkIns)),
		Warnings:          fallbackWarnings(goal),
		MilestoneProgress: milestoneProgress,
		GeneratedBy:       "backend_fallback",
		FallbackUsed:      true,
		GeneratedAt:       now,
		ProgressWeek:      progressWeek,
		DurationWeeks:     goal.DurationWeeks,
		CheckInCount:      len(checkIns),
		SourceJobID:       goal.SourceJobID,
		ProjectedRiskNote: projectedRiskNote(goal),
	}
}

func buildPlannerCoachGoalPayload(goal *models.PlannerGoal) *plannerCoachGoalPayload {
	if goal == nil {
		return nil
	}

	return &plannerCoachGoalPayload{
		ID:                      goal.ID,
		UserID:                  goal.UserID,
		Title:                   goal.Title,
		Status:                  goal.Status,
		CurrentRiskPercentage:   goal.CurrentRiskPercentage,
		TargetRiskPercentage:    goal.TargetRiskPercentage,
		DurationWeeks:           goal.DurationWeeks,
		ProjectedRiskPercentage: goal.ProjectedRiskPercentage,
		SourceJobID:             goal.SourceJobID,
		CreatedAtMillis:         goal.CreatedAtMillis,
		Summary:                 goal.Summary,
		MonitoringPlan:          goal.MonitoringPlan,
		Features:                goal.Features,
	}
}

func fallbackHeadline(goal *models.PlannerGoal, progressWeek int) string {
	featureLabel := "target utama"
	if goal != nil && len(goal.Features) > 0 && strings.TrimSpace(goal.Features[0].Label) != "" {
		featureLabel = goal.Features[0].Label
	}
	return fmt.Sprintf("Fokus Minggu %d: %s", progressWeek, featureLabel)
}

func fallbackSummary(goal *models.PlannerGoal, milestoneProgress *models.PlannerCoachMilestoneProgress, progressWeek, checkInCount int) string {
	if goal == nil {
		return ""
	}
	if focus := plannerPrimaryMilestone(milestoneProgress); focus != nil {
		return fmt.Sprintf(
			"Pada minggu %d, fokus utama ada pada %s. Saat ini %s dengan target minggu ini %s untuk menjaga arah menuju target risiko di bawah %d%%.",
			progressWeek,
			strings.ToLower(focus.Label),
			focus.CurrentText,
			focus.ExpectedText,
			goal.TargetRiskPercentage,
		)
	}
	target := goal.TargetRiskPercentage
	duration := goal.DurationWeeks
	if checkInCount == 0 {
		return fmt.Sprintf(
			"Mulai minggu %d dengan fokus pada perubahan yang paling berdampak agar risiko bergerak ke bawah %d%% dalam %d minggu.",
			progressWeek,
			target,
			duration,
		)
	}
	return fmt.Sprintf(
		"Lanjutkan progres minggu %d dengan menjaga perubahan utama tetap konsisten agar risiko tetap bergerak ke bawah %d%% dalam %d minggu.",
		progressWeek,
		target,
		duration,
	)
}

func fallbackFocus(goal *models.PlannerGoal, milestoneProgress *models.PlannerCoachMilestoneProgress) []string {
	if focus := plannerPrimaryMilestone(milestoneProgress); focus != nil {
		return dedupeStrings([]string{
			fmt.Sprintf("Fokuskan minggu ini pada %s: saat ini %s, target minggu ini %s.", focus.Label, focus.CurrentText, focus.ExpectedText),
			plannerStatusGuidance(focus),
		})
	}

	if goal == nil || len(goal.Features) == 0 {
		return []string{
			"Pertahankan satu perubahan utama yang paling realistis untuk dijalankan minggu ini.",
			"Gunakan check-in untuk melihat apakah target mingguan sudah cukup realistis.",
		}
	}

	focus := make([]string, 0, 2)
	for _, feature := range goal.Features {
		if action := strings.TrimSpace(feature.ActionLabel); action != "" {
			focus = append(focus, action)
		}
		if len(focus) == 2 {
			break
		}
	}
	if len(focus) == 0 {
		focus = append(focus, "Pertahankan perubahan utama secara bertahap dan konsisten.")
	}
	return dedupeStrings(focus)
}

func fallbackActionSteps(goal *models.PlannerGoal, milestoneProgress *models.PlannerCoachMilestoneProgress) []string {
	if goal == nil {
		return nil
	}

	steps := dedupeStrings(goal.ActionSteps)
	if len(steps) > 0 {
		if focus := plannerPrimaryMilestone(milestoneProgress); focus != nil {
			steps = append([]string{fmt.Sprintf("Perbarui progres %s agar bergerak dari %s menuju %s.", strings.ToLower(focus.Label), focus.CurrentText, focus.ExpectedText)}, steps...)
		}
		return dedupeStrings(steps)
	}

	steps = append(steps, fallbackFocus(goal, milestoneProgress)...)
	steps = append(steps,
		"Evaluasi hambatan yang muncul sebelum menaikkan intensitas perubahan.",
		"Gunakan check-in rutin untuk memastikan target mingguan tetap realistis.",
	)
	return dedupeStrings(steps)
}

func fallbackMonitoring(goal *models.PlannerGoal, milestoneProgress *models.PlannerCoachMilestoneProgress, checkInCount int) []string {
	if goal == nil {
		return nil
	}

	points := []string{
		fmt.Sprintf("Plan saat ini berada di minggu %d dari %d.", plannerProgressWeek(goal, time.Now().UTC()), goal.DurationWeeks),
		fmt.Sprintf("Total check-in yang sudah tercatat: %d.", checkInCount),
	}
	for _, item := range plannerTopMilestonesForMonitoring(milestoneProgress) {
		points = append(points,
			fmt.Sprintf("%s saat ini %s dari target akhir %s.", item.Label, item.CurrentText, item.TargetText),
			fmt.Sprintf("Progres %s sekitar %d%% dengan status %s.", strings.ToLower(item.Label), item.ProgressPercentage, plannerStatusLabel(item.Status)),
		)
	}
	if goal.CurrentRiskPercentage != nil {
		points = append(points, fmt.Sprintf("Risiko saat goal dibuat: %.0f%%.", *goal.CurrentRiskPercentage))
	}
	if goal.ProjectedRiskPercentage != nil {
		points = append(points, fmt.Sprintf("Risiko proyeksi berdasarkan skenario: %.0f%%.", *goal.ProjectedRiskPercentage))
	}
	points = append(points, "Catat progres dan hambatan pada check-in berikutnya.")
	return dedupeStrings(points)
}

func fallbackWarnings(goal *models.PlannerGoal) []string {
	if goal == nil {
		return nil
	}

	var warnings []string
	for _, feature := range goal.Features {
		switch strings.ToLower(strings.TrimSpace(feature.FeatureName)) {
		case "bmi":
			warnings = append(warnings,
				"Lakukan perubahan berat badan secara bertahap dan hindari target yang terlalu agresif tanpa evaluasi klinis.",
			)
		case "is_hypertension":
			warnings = append(warnings,
				"Perubahan gaya hidup terkait hipertensi sebaiknya mempertimbangkan kontrol tekanan darah dan terapi yang sedang berjalan.",
			)
		case "is_cholesterol":
			warnings = append(warnings,
				"Perubahan terkait kolesterol sebaiknya mempertimbangkan pola makan, hasil lipid, dan terapi yang sedang berjalan.",
			)
		case "smoking_status", "brinkman_index":
			warnings = append(warnings,
				"Perubahan terkait merokok lebih aman dijalankan bertahap dengan dukungan konseling atau pendampingan yang sesuai.",
			)
		case "moderate_physical_activity_frequency":
			warnings = append(warnings,
				"Naikkan aktivitas fisik secara bertahap dan evaluasi gejala bila ada keterbatasan saat beraktivitas.",
			)
		}
	}
	return dedupeStrings(warnings)
}

func projectedRiskNote(goal *models.PlannerGoal) string {
	if goal == nil || goal.ProjectedRiskPercentage == nil {
		return ""
	}

	targetDelta := ""
	if goal.CurrentRiskPercentage != nil {
		delta := *goal.CurrentRiskPercentage - *goal.ProjectedRiskPercentage
		targetDelta = fmt.Sprintf(" Perubahan skenario ini memproyeksikan penurunan sekitar %.0f poin risiko.", math.Max(delta, 0))
	}

	return fmt.Sprintf(
		"Jika plan berjalan baik, risiko dapat bergerak menuju sekitar %.0f%%.%s",
		*goal.ProjectedRiskPercentage,
		targetDelta,
	)
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func plannerPrimaryMilestone(milestoneProgress *models.PlannerCoachMilestoneProgress) *models.PlannerCoachMilestoneItem {
	if milestoneProgress == nil || len(milestoneProgress.Items) == 0 {
		return nil
	}

	for i := range milestoneProgress.Items {
		if milestoneProgress.Items[i].Status == "BEHIND" {
			return &milestoneProgress.Items[i]
		}
	}
	for i := range milestoneProgress.Items {
		if milestoneProgress.Items[i].Status == "ON_TRACK" {
			return &milestoneProgress.Items[i]
		}
	}
	return &milestoneProgress.Items[0]
}

func plannerTopMilestonesForMonitoring(milestoneProgress *models.PlannerCoachMilestoneProgress) []models.PlannerCoachMilestoneItem {
	if milestoneProgress == nil || len(milestoneProgress.Items) == 0 {
		return nil
	}
	if len(milestoneProgress.Items) <= 2 {
		return milestoneProgress.Items
	}
	return milestoneProgress.Items[:2]
}

func plannerStatusGuidance(item *models.PlannerCoachMilestoneItem) string {
	if item == nil {
		return "Gunakan check-in minggu ini untuk memastikan progres tetap realistis."
	}
	switch item.Status {
	case "BEHIND":
		return "Prioritaskan satu perubahan kecil namun konsisten agar target mingguan tidak makin tertinggal."
	case "ON_TRACK":
		return "Pertahankan ritme yang sudah berjalan agar target mingguan tetap tercapai."
	case "ACHIEVED":
		return "Pertahankan hasil yang sudah tercapai sambil menjaga konsistensi check-in."
	default:
		return "Lakukan check-in terkait faktor ini agar planner bisa membaca progres dengan lebih akurat."
	}
}

func plannerStatusLabel(status string) string {
	switch status {
	case "ACHIEVED":
		return "tercapai"
	case "ON_TRACK":
		return "on track"
	case "BEHIND":
		return "tertinggal"
	default:
		return "dipantau"
	}
}
