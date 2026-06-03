package dto

import (
	"diabetify/internal/models"
	"time"
)

type UserResponse struct {
	ID               uint       `json:"id"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Name             string     `json:"name"`
	Email            string     `json:"email"`
	Gender           *string    `json:"gender"`
	DOB              *string    `json:"dob"`
	Role             string     `json:"role"`
	Verified         bool       `json:"verified"`
	LastPredictionAt *time.Time `json:"last_prediction_at,omitempty"`
}

func NewUserResponse(user *models.User) UserResponse {
	if user == nil {
		return UserResponse{}
	}

	return UserResponse{
		ID:               user.ID,
		CreatedAt:        user.CreatedAt,
		UpdatedAt:        user.UpdatedAt,
		Name:             user.Name,
		Email:            user.Email,
		Gender:           user.Gender,
		DOB:              user.DOB,
		Role:             user.Role,
		Verified:         user.Verified,
		LastPredictionAt: user.LastPredictionAt,
	}
}

func NewUserResponses(users []*models.User) []UserResponse {
	responses := make([]UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, NewUserResponse(user))
	}
	return responses
}

type UserProfileResponse struct {
	ID                        uint      `json:"id"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
	UserID                    uint      `json:"user_id"`
	Hypertension              *bool     `json:"hypertension"`
	Cholesterol               *bool     `json:"cholesterol"`
	Bloodline                 *bool     `json:"bloodline"`
	Weight                    *int      `json:"weight"`
	Height                    *int      `json:"height"`
	BMI                       *float64  `json:"bmi"`
	Smoking                   *int      `json:"smoking"`
	AgeOfSmoking              *int      `json:"age_of_smoking"`
	AgeOfStopSmoking          *int      `json:"age_of_stop_smoking"`
	MacrosomicBaby            *int      `json:"macrosomic_baby"`
	PhysicalActivityFrequency *int      `json:"physical_activity_frequency"`
	SmokeCount                *int      `json:"smoke_count"`
}

func NewUserProfileResponse(profile *models.UserProfile) UserProfileResponse {
	if profile == nil {
		return UserProfileResponse{}
	}

	return UserProfileResponse{
		ID:                        profile.ID,
		CreatedAt:                 profile.CreatedAt,
		UpdatedAt:                 profile.UpdatedAt,
		UserID:                    profile.UserID,
		Hypertension:              profile.Hypertension,
		Cholesterol:               profile.Cholesterol,
		Bloodline:                 profile.Bloodline,
		Weight:                    profile.Weight,
		Height:                    profile.Height,
		BMI:                       profile.BMI,
		Smoking:                   profile.Smoking,
		AgeOfSmoking:              profile.AgeOfSmoking,
		AgeOfStopSmoking:          profile.AgeOfStopSmoking,
		MacrosomicBaby:            profile.MacrosomicBaby,
		PhysicalActivityFrequency: profile.PhysicalActivityFrequency,
		SmokeCount:                profile.SmokeCount,
	}
}

type PredictionJobResponse struct {
	ID           string     `json:"id"`
	UserID       uint       `json:"user_id"`
	Status       string     `json:"status"`
	PredictionID *uint      `json:"prediction_id,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

func NewPredictionJobResponse(job *models.PredictionJob) PredictionJobResponse {
	if job == nil {
		return PredictionJobResponse{}
	}

	return PredictionJobResponse{
		ID:           job.ID,
		UserID:       job.UserID,
		Status:       job.Status,
		PredictionID: job.PredictionID,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt,
		UpdatedAt:    job.UpdatedAt,
		CompletedAt:  job.CompletedAt,
	}
}

func NewPredictionJobResponses(jobs []*models.PredictionJob) []PredictionJobResponse {
	responses := make([]PredictionJobResponse, 0, len(jobs))
	for _, job := range jobs {
		responses = append(responses, NewPredictionJobResponse(job))
	}
	return responses
}

type CounterfactualJobResponse struct {
	ID           string     `json:"id"`
	UserID       uint       `json:"user_id"`
	Status       string     `json:"status"`
	ReasonCode   *string    `json:"reason_code,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

func NewCounterfactualJobResponse(job *models.CounterfactualJob) CounterfactualJobResponse {
	if job == nil {
		return CounterfactualJobResponse{}
	}

	return CounterfactualJobResponse{
		ID:           job.ID,
		UserID:       job.UserID,
		Status:       job.Status,
		ReasonCode:   job.ReasonCode,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt,
		UpdatedAt:    job.UpdatedAt,
		CompletedAt:  job.CompletedAt,
	}
}

func NewCounterfactualJobResponses(jobs []*models.CounterfactualJob) []CounterfactualJobResponse {
	responses := make([]CounterfactualJobResponse, 0, len(jobs))
	for _, job := range jobs {
		responses = append(responses, NewCounterfactualJobResponse(job))
	}
	return responses
}
