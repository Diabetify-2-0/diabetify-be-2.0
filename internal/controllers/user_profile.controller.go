package controllers

import (
	"diabetify/internal/dto"
	"diabetify/internal/models"
	"diabetify/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserProfileController struct {
	service services.UserProfileService
}

func NewUserProfileController(service services.UserProfileService) *UserProfileController {
	return &UserProfileController{service: service}
}

// GetUserProfile godoc
// @Summary Get user profile
// @Description Retrieve the authenticated user's profile
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User profile retrieved successfully"
// @Failure 404 {object} map[string]interface{} "Profile not found"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /profile [get]
func (pc *UserProfileController) GetUserProfile(c *gin.Context) {
	// Get user ID from the JWT token (set by middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	// Call service to get profile
	profile, err := pc.service.GetUserProfile(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Profile not found",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User profile retrieved successfully",
		"data":    dto.NewUserProfileResponse(profile),
	})
}

// CreateUserProfile godoc
// @Summary Create user profile
// @Description Create a profile for the authenticated user
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body models.UserProfile true "Profile data"
// @Success 201 {object} map[string]interface{} "Profile created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to create profile"
// @Router /profile [post]
func (pc *UserProfileController) CreateUserProfile(c *gin.Context) {
	var profile models.UserProfile

	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Get user ID from the JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	profile.UserID = userID.(uint)

	// Call service to create profile (service handles BMI calculation)
	createdProfile, err := pc.service.CreateUserProfile(&profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to create profile",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Profile created successfully",
		"data":    dto.NewUserProfileResponse(createdProfile),
	})
}

// UpdateUserProfile godoc
// @Summary Update user profile
// @Description Update the authenticated user's profile
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body models.UserProfile true "Profile data"
// @Success 200 {object} map[string]interface{} "Profile updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Profile not found"
// @Failure 500 {object} map[string]interface{} "Failed to update profile"
// @Router /profile [put]
func (pc *UserProfileController) UpdateUserProfile(c *gin.Context) {
	var updatedProfile models.UserProfile
	if err := c.ShouldBindJSON(&updatedProfile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Get user ID from the JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	updatedProfile.UserID = userID.(uint)

	// Call service to update profile (service handles BMI recalculation and validation)
	profile, err := pc.service.UpdateUserProfile(&updatedProfile)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Profile not found",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profile updated successfully",
		"data":    dto.NewUserProfileResponse(profile),
	})
}

// DeleteUserProfile godoc
// @Summary Delete user profile
// @Description Delete the authenticated user's profile
// @Tags profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Profile deleted successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to delete profile"
// @Router /profile [delete]
func (pc *UserProfileController) DeleteUserProfile(c *gin.Context) {
	// Get user ID from the JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	// Call service to delete profile
	if err := pc.service.DeleteUserProfile(userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete profile",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profile deleted successfully",
		"data":    nil,
	})
}

// PatchUserProfile godoc
// @Summary Patch user profile
// @Description Update specific fields of the authenticated user's profile
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body map[string]interface{} true "Profile data to update"
// @Success 200 {object} map[string]interface{} "Profile patched successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Profile not found"
// @Failure 500 {object} map[string]interface{} "Failed to update profile"
// @Router /profile [patch]
func (pc *UserProfileController) PatchUserProfile(c *gin.Context) {
	var patchData map[string]interface{}
	if err := c.ShouldBindJSON(&patchData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	// Call service to patch profile (service handles BMI recalculation if needed)
	if err := pc.service.PatchUserProfile(userID.(uint), patchData); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Failed to patch profile",
			"error":   err.Error(),
		})
		return
	}

	updatedProfile, err := pc.service.GetUserProfile(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve updated profile",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profile patched successfully",
		"data":    dto.NewUserProfileResponse(updatedProfile),
	})
}
