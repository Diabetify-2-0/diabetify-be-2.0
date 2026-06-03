package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PlannerController struct {
	service services.PlannerService
}

func NewPlannerController(service services.PlannerService) *PlannerController {
	return &PlannerController{service: service}
}

func (pc *PlannerController) SaveGoal(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	var goal models.PlannerGoal
	if err := c.ShouldBindJSON(&goal); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid planner goal data",
			"error":   err.Error(),
		})
		return
	}

	if goal.ID == "" || goal.Title == "" || goal.TargetRiskPercentage <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Planner goal id, title, and target risk are required",
		})
		return
	}

	goal.UserID = userID
	if err := pc.service.SaveGoal(&goal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to save planner goal",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Planner goal saved successfully",
		"data":    goal,
	})
}

func (pc *PlannerController) GetLatestGoal(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	goal, err := pc.service.GetLatestGoal(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve planner goal",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Planner goal retrieved successfully",
		"data":    goal,
	})
}

func (pc *PlannerController) RecordCheckIn(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	goalID := c.Param("id")
	var entry models.PlannerCheckInEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid check-in data",
			"error":   err.Error(),
		})
		return
	}

	if entry.ID == "" || entry.Type == "" || entry.Label == "" || entry.ValueText == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Check-in id, type, label, and value are required",
		})
		return
	}

	entry.UserID = userID
	entry.GoalID = goalID
	if err := pc.service.RecordCheckIn(&entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to record planner check-in",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Planner check-in recorded successfully",
		"data":    entry,
	})
}

func (pc *PlannerController) GetCheckInHistory(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "80"))
	if err != nil || limit <= 0 {
		limit = 80
	}

	entries, err := pc.service.GetCheckInHistory(userID, c.Param("id"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve planner check-ins",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Planner check-ins retrieved successfully",
		"data":    entries,
	})
}

func (pc *PlannerController) GetLastCheckIns(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	lastCheckIns, err := pc.service.GetLastCheckIns(userID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve planner check-in state",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Planner check-in state retrieved successfully",
		"data":    lastCheckIns,
	})
}

func (pc *PlannerController) DeleteGoal(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	if err := pc.service.DeleteGoal(userID, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete planner goal",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Planner goal deleted successfully",
		"data":    nil,
	})
}

func authenticatedUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return 0, false
	}
	return userID.(uint), true
}
