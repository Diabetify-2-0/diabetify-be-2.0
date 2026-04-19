package controllers

import (
	"diabetify/internal/models"
	"diabetify/internal/services"
	"diabetify/internal/utils"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type UserController struct {
	service services.UserService
}

func NewUserController(service services.UserService) *UserController {
	return &UserController{
		service: service,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type RegisterUserRequest struct {
	Email    string  `json:"email" binding:"required,email"`
	Password string  `json:"password" binding:"required,min=8"`
	Name     string  `json:"name" binding:"required,min=2"`
	Gender   *string `json:"gender"`
	DOB      *string `json:"dob"`
	Role     string  `json:"role" binding:"omitempty,oneof=ADMIN USER DATA_SCIENTIST MEDICAL_EXPERT"` // Default: USER if not provided
}

// RegisterUser godoc
// @Summary Register a new user
// @Description Register a new user. Role is optional and defaults to USER. Valid roles: ADMIN, USER, DATA_SCIENTIST, MEDICAL_EXPERT
// @Tags users
// @Accept json
// @Produce json
// @Param request body RegisterUserRequest true "User registration data (role optional, defaults to USER)"
// @Success 201 {object} map[string]interface{} "User registered successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data or invalid role"
// @Failure 500 {object} map[string]interface{} "Failed to create user"
// @Router /users [post]
func (uc *UserController) RegisterUser(c *gin.Context) {
	var req RegisterUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Set default role to USER if not provided
	if req.Role == "" {
		req.Role = "USER"
	}

	// Call service to create user (validation, password hashing, email duplicate check)
	user, err := uc.service.CreateUser(req.Email, req.Password, req.Name, req.Role, req.Gender, req.DOB)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Failed to create user",
			"error":   err.Error(),
		})
		return
	}

	var message string
	switch req.Role {
	case "ADMIN":
		message = "Admin registered successfully."
	case "DATA_SCIENTIST":
		message = "Data Scientist registered successfully."
	case "MEDICAL_EXPERT":
		message = "Medical Expert registered successfully."
	default:
		message = "User registered successfully. Please verify your email."
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": message,
		"data": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

// GetUserByID godoc
// @Summary Get a user by ID
// @Description Retrieve user information by user ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Router /users/{id} [get]
func (uc *UserController) GetUserByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid user ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	user, err := uc.service.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user exists with the provided ID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User retrieved successfully",
		"data":    user,
	})
}

// UpdateUser godoc
// @Summary Update current user
// @Description Update the authenticated user's information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body models.User true "User data"
// @Success 200 {object} map[string]interface{} "User updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Failed to update user"
// @Router /users/me [put]
func (uc *UserController) UpdateUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return
	}

	existingUser, err := uc.service.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user exists with the provided ID",
		})
		return
	}

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Set the user ID from the JWT token
	user.ID = userID.(uint)

	// Prevent changing email to one that already exists
	if user.Email != "" && user.Email != existingUser.Email {
		_, err := uc.service.GetUserByEmail(user.Email)
		if err == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Email already in use",
				"error":   "Email address is already registered",
			})
			return
		}
	}

	// Service will handle password hashing and validation
	if err := uc.service.UpdateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update user",
			"error":   err.Error(),
		})
		return
	}

	user.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User updated successfully",
		"data":    user,
	})
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Delete user by ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 500 {object} map[string]interface{} "Failed to delete user"
// @Router /users/{id} [delete]
func (uc *UserController) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid user ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	if err := uc.service.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete user",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User deleted successfully",
		"data":    nil,
	})
}

// LoginUser godoc
// @Summary Login a user
// @Description Authenticate user credentials
// @Tags users
// @Accept json
// @Produce json
// @Param login body LoginRequest true "Email and Password"
// @Success 200 {object} map[string]interface{} "User logged in successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Router /users/login [post]
func (uc *UserController) LoginUser(c *gin.Context) {
	var loginRequest LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	user, err := uc.service.GetUserByEmail(loginRequest.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user associated with this email",
		})
		return
	}

	// Verify password using service
	if !uc.service.VerifyPassword(user.Password, loginRequest.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "Invalid email or password",
		})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})
	jwtSecret := []byte(os.Getenv("JWT_SECRET_KEY"))
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not generate token",
			"error":   err.Error(),
		})
		return
	}
	// Authentication is successful
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User logged in successfully",
		"data":    tokenString,
	})
}

// ForgotPassword godoc
// @Summary Request password reset code
// @Description Send a verification code to user's email for password reset
// @Tags users
// @Accept json
// @Produce json
// @Param forgotPassword body ForgotPasswordRequest true "User Email"
// @Success 200 {object} map[string]interface{} "Code sent successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data or email does not exist"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/forgot-password [post]
func (uc *UserController) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Call service to generate reset code
	code, err := uc.service.ForgotPassword(req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Failed to process forgot password request",
			"error":   err.Error(),
		})
		return
	}

	mailConfig := utils.LoadMailConfig()

	// Send email asynchronously
	go func() {
		if err := utils.SendEmail(mailConfig, req.Email, "Reset Password", "Use this code to reset your password :  "+code); err != nil {
			log.Printf("Failed to send email to %s: %v", req.Email, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Code sent successfully",
		"data":    nil,
	})
}

// ResetPassword godoc
// @Summary Reset user password
// @Description Reset user password using verification code
// @Tags users
// @Accept json
// @Produce json
// @Param resetPassword body ResetPasswordRequest true "Email, Code, and New Password"
// @Success 200 {object} map[string]interface{} "Password has been reset successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data, code expired, or invalid password"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/reset-password [post]
func (uc *UserController) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Call service to reset password
	if err := uc.service.ResetPassword(req.Email, req.Code, req.NewPassword); err != nil {
		// Determine if it's a validation error or not found error
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Failed to reset password",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Password has been reset successfully",
		"data":    nil,
	})
}

// PatchUser godoc
// @Summary Patch current user
// @Description Update specific fields of the authenticated user's information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userData body map[string]interface{} true "User data to update"
// @Success 200 {object} map[string]interface{} "User patched successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Failed to update user"
// @Router /users/me [patch]
func (uc *UserController) PatchUser(c *gin.Context) {
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

	// Check if the user exists
	existingUser, err := uc.service.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user exists with the provided ID",
		})
		return
	}

	// Parse the patch data
	var patchData map[string]interface{}
	if err := c.ShouldBindJSON(&patchData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Prevent changing email to one that already exists
	if email, hasEmail := patchData["email"].(string); hasEmail && email != existingUser.Email {
		// Check if email already exists
		_, err := uc.service.GetUserByEmail(email)
		if err == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Email already in use",
				"error":   "Email address is already registered",
			})
			return
		}
	}

	// Apply the patch to the user through service (handles password hashing)
	if err := uc.service.PatchUser(userID.(uint), patchData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update user",
			"error":   err.Error(),
		})
		return
	}

	// Get the updated user
	updatedUser, err := uc.service.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "User updated but failed to retrieve updated data",
			"error":   err.Error(),
		})
		return
	}

	// Hide sensitive information
	updatedUser.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User patched successfully",
		"data":    updatedUser,
	})
}

// GetCurrentUser godoc
// @Summary Get current user information
// @Description Retrieve information about the currently authenticated user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User information retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Router /users/me [get]
func (uc *UserController) GetCurrentUser(c *gin.Context) {
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

	// Retrieve the user
	user, err := uc.service.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user exists with this ID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User information retrieved successfully",
		"data":    user,
	})
}

func (uc *UserController) requireAdmin(c *gin.Context) bool {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Unauthorized",
			"error":   "User ID not found in token",
		})
		return false
	}

	user, err := uc.service.GetUserByID(userID.(uint))
	if err != nil || user.Role != "ADMIN" {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Forbidden",
			"error":   "Admin access required",
		})
		return false
	}
	return true
}

// ListAllUsers godoc
// @Summary List all non-patient users (admin only)
// @Description Returns all users with roles other than USER. Requires ADMIN role.
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Users retrieved successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve users"
// @Router /admin/users [get]
func (uc *UserController) ListAllUsers(c *gin.Context) {
	if !uc.requireAdmin(c) {
		return
	}

	users, err := uc.service.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve users",
			"error":   err.Error(),
		})
		return
	}

	for _, u := range users {
		u.Password = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Users retrieved successfully",
		"data":    users,
	})
}

// AdminPatchUser godoc
// @Summary Update any user by ID (admin only)
// @Description Admin can update name, email, role, or verified status for any user.
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param userData body map[string]interface{} true "Fields to update"
// @Success 200 {object} map[string]interface{} "User updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Failed to update user"
// @Router /admin/users/{id} [patch]
func (uc *UserController) AdminPatchUser(c *gin.Context) {
	if !uc.requireAdmin(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid user ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	_, err = uc.service.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "User not found",
			"error":   "No user exists with the provided ID",
		})
		return
	}

	var patchData map[string]interface{}
	if err := c.ShouldBindJSON(&patchData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	if err := uc.service.PatchUser(uint(id), patchData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update user",
			"error":   err.Error(),
		})
		return
	}

	updatedUser, err := uc.service.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "User updated but failed to retrieve updated data",
			"error":   err.Error(),
		})
		return
	}
	updatedUser.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User updated successfully",
		"data":    updatedUser,
	})
}

// AdminDeleteUser godoc
// @Summary Delete any user by ID (admin only)
// @Description Admin can delete any user account.
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Failed to delete user"
// @Router /admin/users/{id} [delete]
func (uc *UserController) AdminDeleteUser(c *gin.Context) {
	if !uc.requireAdmin(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid user ID",
			"error":   "ID must be a valid positive integer",
		})
		return
	}

	if err := uc.service.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to delete user",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User deleted successfully",
		"data":    nil,
	})
}
