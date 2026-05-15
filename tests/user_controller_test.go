package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"diabetify/internal/controllers"
	"diabetify/internal/models"
	"diabetify/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test helper functions
func setupUserTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupUserControllerWithMocks() (*controllers.UserController, *mocks.MockUserService) {
	mockUserService := new(mocks.MockUserService)
	controller := controllers.NewUserController(mockUserService)
	return controller, mockUserService
}

func addUserAuthMiddleware(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

// Helper function to create a test password hash
func createTestPasswordHash(password string) string {
	// Use a fixed salt for testing
	salt := make([]byte, 8)
	// Fill with known values for consistency
	for i := range salt {
		salt[i] = byte(i)
	}

	// SHA256
	h := sha256.New()
	h.Write([]byte(password))
	h.Write(salt)
	hash := h.Sum(nil)

	return hex.EncodeToString(salt) + hex.EncodeToString(hash)
}

func TestLoginUser(t *testing.T) {
	os.Setenv("JWT_SECRET_KEY", "test-secret-key")
	defer os.Unsetenv("JWT_SECRET_KEY")

	// Create a proper test password hash
	testPassword := "password123"
	testPasswordHash := createTestPasswordHash(testPassword)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockUserService)
		expectedStatus int
		expectedMsg    string
		checkToken     bool
	}{
		{
			name: "successful login",
			requestBody: map[string]interface{}{
				"email":    "john@example.com",
				"password": testPassword,
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				user := &models.User{
					ID:       1,
					Email:    "john@example.com",
					Password: testPasswordHash,
				}
				mockService.On("GetUserByEmail", "john@example.com").Return(user, nil)
				mockService.On("VerifyPassword", testPasswordHash, testPassword).Return(true)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "User logged in successfully",
			checkToken:     true,
		},
		{
			name: "user not found",
			requestBody: map[string]interface{}{
				"email":    "nonexistent@example.com",
				"password": "password123",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("GetUserByEmail", "nonexistent@example.com").Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
			checkToken:     false,
		},
		{
			name: "incorrect password",
			requestBody: map[string]interface{}{
				"email":    "john@example.com",
				"password": "wrongpassword",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				user := &models.User{
					ID:       1,
					Email:    "john@example.com",
					Password: testPasswordHash,
				}
				mockService.On("GetUserByEmail", "john@example.com").Return(user, nil)
				mockService.On("VerifyPassword", testPasswordHash, "wrongpassword").Return(false)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "Unauthorized",
			checkToken:     false,
		},
		{
			name: "invalid request data",
			requestBody: map[string]interface{}{
				"email": "john@example.com",
				// Missing password
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
			checkToken:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupUserControllerWithMocks()
			tt.setupMocks(mockService)

			router := setupUserTestRouter()
			router.POST("/users/login", controller.LoginUser)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/users/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			if tt.checkToken {
				assert.NotNil(t, response["data"])
				assert.IsType(t, "", response["data"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestRegisterUserRejectsPrivilegedRole(t *testing.T) {
	controller, mockService := setupUserControllerWithMocks()

	router := setupUserTestRouter()
	router.POST("/users", controller.RegisterUser)

	body, _ := json.Marshal(map[string]interface{}{
		"email":    "admin@example.com",
		"password": "password123",
		"name":     "Admin Candidate",
		"role":     "ADMIN",
	})

	req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Public registration cannot assign privileged roles")

	mockService.AssertExpectations(t)
}

func TestUpdateUserRejectsPayloadWithoutAllowedFields(t *testing.T) {
	controller, mockService := setupUserControllerWithMocks()

	mockService.On("GetUserByID", uint(1)).Return(&models.User{
		ID:    1,
		Email: "john@example.com",
		Name:  "John",
		Role:  "USER",
	}, nil)

	router := setupUserTestRouter()
	router.Use(addUserAuthMiddleware(1))
	router.PUT("/users/me", controller.UpdateUser)

	body, _ := json.Marshal(map[string]interface{}{
		"role": "ADMIN",
	})

	req := httptest.NewRequest("PUT", "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "No updatable fields were provided")

	mockService.AssertExpectations(t)
}

func TestUpdateUserDoesNotAllowRoleEscalation(t *testing.T) {
	controller, mockService := setupUserControllerWithMocks()

	mockService.On("GetUserByID", uint(1)).Return(&models.User{
		ID:       1,
		Email:    "john@example.com",
		Password: "existing-hash",
		Name:     "John",
		Role:     "USER",
		Verified: false,
	}, nil)
	mockService.
		On("UpdateUser", mock.MatchedBy(func(user *models.User) bool {
			return user.ID == 1 &&
				user.Name == "Jane Doe" &&
				user.Role == "USER" &&
				user.Verified == false
		})).
		Return(nil)

	router := setupUserTestRouter()
	router.Use(addUserAuthMiddleware(1))
	router.PUT("/users/me", controller.UpdateUser)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "Jane Doe",
		"role": "ADMIN",
	})

	req := httptest.NewRequest("PUT", "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "User updated successfully")

	mockService.AssertExpectations(t)
}

func TestPatchUserRejectsRestrictedFields(t *testing.T) {
	controller, mockService := setupUserControllerWithMocks()

	mockService.On("GetUserByID", uint(1)).Return(&models.User{
		ID:    1,
		Email: "john@example.com",
		Name:  "John",
		Role:  "USER",
	}, nil)

	router := setupUserTestRouter()
	router.Use(addUserAuthMiddleware(1))
	router.PATCH("/users/me", controller.PatchUser)

	body, _ := json.Marshal(map[string]interface{}{
		"role": "ADMIN",
	})

	req := httptest.NewRequest("PATCH", "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "field role cannot be updated")

	mockService.AssertExpectations(t)
}

func TestGetCurrentUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		setupMocks     func(*mocks.MockUserService)
		hasAuth        bool
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful get current user",
			userID: 1,
			setupMocks: func(mockService *mocks.MockUserService) {
				user := &models.User{
					ID:    1,
					Name:  "John Doe",
					Email: "john@example.com",
				}
				mockService.On("GetUserByID", uint(1)).Return(user, nil)
			},
			hasAuth:        true,
			expectedStatus: http.StatusOK,
			expectedMsg:    "User information retrieved successfully",
		},
		{
			name:   "user not found",
			userID: 999,
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("GetUserByID", uint(999)).Return(nil, errors.New("user not found"))
			},
			hasAuth:        true,
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
		},
		{
			name:           "unauthorized - no user_id in context",
			userID:         0,
			setupMocks:     func(mockService *mocks.MockUserService) {},
			hasAuth:        false,
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupUserControllerWithMocks()
			tt.setupMocks(mockService)

			router := setupUserTestRouter()
			if tt.hasAuth {
				router.Use(addUserAuthMiddleware(tt.userID))
			}
			router.GET("/users/me", controller.GetCurrentUser)

			req := httptest.NewRequest("GET", "/users/me", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			mockService.AssertExpectations(t)
		})
	}
}

func TestForgotPassword(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockUserService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "successful forgot password",
			requestBody: map[string]interface{}{
				"email": "user@example.com",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("ForgotPassword", "user@example.com").Return("123456", nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Code sent successfully",
		},
		{
			name: "email does not exist",
			requestBody: map[string]interface{}{
				"email": "nonexistent@example.com",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("ForgotPassword", "nonexistent@example.com").Return("", errors.New("user not found"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Failed to process forgot password request",
		},
		{
			name:        "invalid request data",
			requestBody: map[string]interface{}{
				// Missing email
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name: "database error when creating reset password",
			requestBody: map[string]interface{}{
				"email": "user@example.com",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("ForgotPassword", "user@example.com").Return("", errors.New("database error"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Failed to process forgot password request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupUserControllerWithMocks()
			tt.setupMocks(mockService)

			router := setupUserTestRouter()
			router.POST("/users/forgot-password", controller.ForgotPassword)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/users/forgot-password", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			mockService.AssertExpectations(t)
		})
	}
}

func TestResetPassword(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockUserService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "successful password reset",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "123456",
				"new_password": "newpassword123",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("ResetPassword", "user@example.com", "123456", "newpassword123").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Password has been reset successfully",
		},
		{
			name: "invalid or expired code",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "wrong123",
				"new_password": "newpassword123",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("ResetPassword", "user@example.com", "wrong123", "newpassword123").Return(errors.New("invalid or expired verification code"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Failed to reset password",
		},
		{
			name: "expired code",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "123456",
				"new_password": "newpassword123",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("ResetPassword", "user@example.com", "123456", "newpassword123").Return(errors.New("code has expired"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Failed to reset password",
		},
		{
			name: "password too short",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "123456",
				"new_password": "short",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("ResetPassword", "user@example.com", "123456", "short").Return(errors.New("password must be at least 8 characters"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Failed to reset password",
		},
		{
			name: "user not found",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "123456",
				"new_password": "newpassword123",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("ResetPassword", "user@example.com", "123456", "newpassword123").Return(errors.New("user not found"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Failed to reset password",
		},
		{
			name: "database error when updating password",
			requestBody: map[string]interface{}{
				"email":        "user@example.com",
				"code":         "123456",
				"new_password": "newpassword123",
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				mockService.On("ResetPassword", "user@example.com", "123456", "newpassword123").Return(errors.New("failed to update password"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Failed to reset password",
		},
		{
			name: "invalid request data",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
				"code":  "123456",
				// Missing new_password
			},
			setupMocks: func(mockService *mocks.MockUserService) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupUserControllerWithMocks()
			tt.setupMocks(mockService)

			router := setupUserTestRouter()
			router.POST("/users/reset-password", controller.ResetPassword)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/users/reset-password", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], tt.expectedMsg)

			mockService.AssertExpectations(t)
		})
	}
}
