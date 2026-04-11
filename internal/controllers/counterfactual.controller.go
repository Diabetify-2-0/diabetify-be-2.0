package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"diabetify/internal/services"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CounterfactualController struct {
	jobRepo repository.CounterfactualJobRepository
	jobSvc  services.CounterfactualJobService
}

func NewCounterfactualController(
	jobRepo repository.CounterfactualJobRepository,
	jobSvc services.CounterfactualJobService,
) *CounterfactualController {
	return &CounterfactualController{
		jobRepo: jobRepo,
		jobSvc:  jobSvc,
	}
}

func (cc *CounterfactualController) SubmitJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request payload",
			"error":   err.Error(),
		})
		return
	}

	if !hasRequiredCounterfactualPayload(payload) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Missing required fields",
			"error":   "Payload must include 'instance' and 'constraints'",
		})
		return
	}

	jobID := uuid.New().String()
	payload["request_id"] = jobID
	payload["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	requestJSON, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to serialize request payload",
			"error":   err.Error(),
		})
		return
	}
	requestPayload := string(requestJSON)

	job := &models.CounterfactualJob{
		ID:             jobID,
		UserID:         userID.(uint),
		Status:         models.CFJobStatusPending,
		RequestPayload: &requestPayload,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := cc.jobRepo.SaveJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create counterfactual job",
			"error":   err.Error(),
		})
		return
	}

	if err := cc.jobSvc.SubmitJob(jobID, payload); err != nil {
		errMsg := fmt.Sprintf("failed to submit counterfactual request: %v", err)
		_ = cc.jobRepo.UpdateJobTerminalStatus(jobID, models.CFJobStatusFailed, nil, &errMsg, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to submit counterfactual job",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":  "success",
		"message": "Counterfactual job submitted successfully",
		"data": gin.H{
			"job_id":      jobID,
			"status":      models.CFJobStatusPending,
			"submit_time": time.Now(),
			"poll_url":    fmt.Sprintf("/counterfactual/job/%s/status", jobID),
			"result_url":  fmt.Sprintf("/counterfactual/job/%s/result", jobID),
		},
	})
}

func (cc *CounterfactualController) Health(c *gin.Context) {
	status := cc.jobSvc.GetStatus()
	isRunning, _ := status["running"].(bool)
	isRabbitConnected, _ := status["rabbitmq_connected"].(bool)

	httpStatus := http.StatusOK
	message := "Counterfactual service is healthy"
	if !isRunning || !isRabbitConnected {
		httpStatus = http.StatusServiceUnavailable
		message = "Counterfactual service is not healthy"
	}

	c.JSON(httpStatus, gin.H{
		"status":  "success",
		"message": message,
		"data": gin.H{
			"service_status": status,
		},
	})
}

func (cc *CounterfactualController) GetJobStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized access",
		})
		return
	}

	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Job ID is required"})
		return
	}

	job, err := cc.jobRepo.GetJobByID(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Job not found", "error": err.Error()})
		return
	}

	if job.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Job belongs to a different user"})
		return
	}

	resp := gin.H{
		"status":  "success",
		"message": "Counterfactual job status retrieved",
		"data": gin.H{
			"job_id":       job.ID,
			"job_status":   job.Status,
			"reason_code":  job.ReasonCode,
			"created_at":   job.CreatedAt,
			"updated_at":   job.UpdatedAt,
			"completed_at": job.CompletedAt,
		},
	}
	if job.ErrorMessage != nil {
		// Persisted field is named error_message in DB, but response should be semantic:
		// failed -> error, completed/infeasible -> message.
		if job.Status == models.CFJobStatusFailed {
			resp["data"].(gin.H)["error"] = *job.ErrorMessage
		} else {
			resp["data"].(gin.H)["message"] = *job.ErrorMessage
		}
	}

	c.JSON(http.StatusOK, resp)
}

func (cc *CounterfactualController) GetJobResult(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized access"})
		return
	}

	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Job ID is required"})
		return
	}

	job, err := cc.jobRepo.GetJobByID(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Job not found", "error": err.Error()})
		return
	}
	if job.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Job belongs to a different user"})
		return
	}

	if job.Status != models.CFJobStatusCompleted && job.Status != models.CFJobStatusInfeasible && job.Status != models.CFJobStatusFailed {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": fmt.Sprintf("Job is not completed yet. Current status: %s", job.Status),
			"data": gin.H{
				"job_id":     job.ID,
				"job_status": job.Status,
			},
		})
		return
	}

	if job.ResultPayload == nil || *job.ResultPayload == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Job has no result payload",
		})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(*job.ResultPayload), &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to parse result payload",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Counterfactual job result retrieved",
		"data": gin.H{
			"job_id":      job.ID,
			"job_status":  job.Status,
			"reason_code": job.ReasonCode,
			"result":      result,
		},
	})
}

func (cc *CounterfactualController) GetUserJobs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized access"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid limit parameter"})
			return
		}
		limit = parsed
	}

	jobs, err := cc.jobRepo.GetJobsByUserID(userID.(uint), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to fetch counterfactual jobs",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Counterfactual jobs retrieved",
		"data": gin.H{
			"jobs":  jobs,
			"count": len(jobs),
		},
	})
}

func (cc *CounterfactualController) CancelJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized access"})
		return
	}

	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Job ID is required"})
		return
	}

	job, err := cc.jobRepo.GetJobByID(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Job not found", "error": err.Error()})
		return
	}
	if job.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Job belongs to a different user"})
		return
	}
	if job.Status == models.CFJobStatusCompleted || job.Status == models.CFJobStatusInfeasible || job.Status == models.CFJobStatusFailed || job.Status == models.CFJobStatusCancelled {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Job is already terminal and cannot be cancelled",
		})
		return
	}

	if err := cc.jobRepo.CancelJob(jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Failed to cancel job", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Counterfactual job cancelled",
		"data": gin.H{
			"job_id":       jobID,
			"cancelled_at": time.Now(),
		},
	})
}

func hasRequiredCounterfactualPayload(payload map[string]interface{}) bool {
	_, hasInstance := payload["instance"]
	_, hasConstraints := payload["constraints"]
	return hasInstance && hasConstraints
}
