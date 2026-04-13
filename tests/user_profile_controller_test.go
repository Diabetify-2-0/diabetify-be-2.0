package tests

import (
	"bytes"
	"diabetify/internal/controllers"
	"diabetify/internal/models"
	"diabetify/tests/mocks"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test helper functions
func setupProfileTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupProfileControllerWithMock() (*controllers.UserProfileController, *mocks.MockUserProfileService) {
	mockService := new(mocks.MockUserProfileService)
	controller := controllers.NewUserProfileController(mockService)
	return controller, mockService
}

func addProfileAuthMiddleware(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func TestNewUserProfileController(t *testing.T) {
	mockService := new(mocks.MockUserProfileService)
	controller := controllers.NewUserProfileController(mockService)

	assert.NotNil(t, controller)
}

func TestGetUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		setupMock      func(*mocks.MockUserProfileService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful retrieval",
			userID: 1,
			setupMock: func(m *mocks.MockUserProfileService) {
				weight := 70
				height := 170
				bmi := 24.2
				profile := &models.UserProfile{
					ID:     1,
					UserID: 1,
					Weight: &weight,
					Height: &height,
					BMI:    &bmi,
				}
				m.On("GetUserProfile", uint(1)).Return(profile, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "User profile retrieved successfully",
		},
		{
			name:   "profile not found",
			userID: 1,
			setupMock: func(m *mocks.MockUserProfileService) {
				m.On("GetUserProfile", uint(1)).Return(nil, errors.New("profile not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Profile not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupProfileControllerWithMock()
			tt.setupMock(mockService)

			router := setupProfileTestRouter()
			router.Use(addProfileAuthMiddleware(tt.userID))
			router.GET("/profile", controller.GetUserProfile)

			req := httptest.NewRequest("GET", "/profile", nil)
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

func TestGetUserProfileUnauthorized(t *testing.T) {
	controller, _ := setupProfileControllerWithMock()
	router := setupProfileTestRouter()
	router.GET("/profile", controller.GetUserProfile)

	req := httptest.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["message"])
}

func TestCreateUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		requestBody    map[string]interface{}
		setupMock      func(*mocks.MockUserProfileService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful creation with BMI calculation",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 70.0,
				"height": 170.0,
			},
			setupMock: func(m *mocks.MockUserProfileService) {
				weight := 70
				height := 170
				bmi := 24.2
				profile := &models.UserProfile{
					ID:     1,
					UserID: 1,
					Weight: &weight,
					Height: &height,
					BMI:    &bmi,
				}
				m.On("CreateUserProfile", mock.MatchedBy(func(p *models.UserProfile) bool {
					return p.UserID == 1
				})).Return(profile, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedMsg:    "Profile created successfully",
		},
		{
			name:   "successful creation without BMI calculation",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 70.0,
			},
			setupMock: func(m *mocks.MockUserProfileService) {
				weight := 70
				profile := &models.UserProfile{
					ID:     1,
					UserID: 1,
					Weight: &weight,
					BMI:    nil,
				}
				m.On("CreateUserProfile", mock.MatchedBy(func(p *models.UserProfile) bool {
					return p.UserID == 1
				})).Return(profile, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedMsg:    "Profile created successfully",
		},
		{
			name:           "invalid JSON",
			userID:         1,
			requestBody:    nil,
			setupMock:      func(m *mocks.MockUserProfileService) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name:   "service error",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 70.0,
				"height": 170.0,
			},
			setupMock: func(m *mocks.MockUserProfileService) {
				m.On("CreateUserProfile", mock.AnythingOfType("*models.UserProfile")).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to create profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupProfileControllerWithMock()
			tt.setupMock(mockService)

			router := setupProfileTestRouter()
			router.Use(addProfileAuthMiddleware(tt.userID))
			router.POST("/profile", controller.CreateUserProfile)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest("POST", "/profile", bytes.NewBuffer(body))
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

func TestCreateUserProfileUnauthorized(t *testing.T) {
	controller, _ := setupProfileControllerWithMock()
	router := setupProfileTestRouter()
	router.POST("/profile", controller.CreateUserProfile)

	requestBody := map[string]interface{}{
		"weight": 70.0,
		"height": 170.0,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["message"])
}

func TestUpdateUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		requestBody    map[string]interface{}
		setupMock      func(*mocks.MockUserProfileService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful update with BMI recalculation",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 75.0,
				"height": 175.0,
			},
			setupMock: func(m *mocks.MockUserProfileService) {
				weight := 75
				height := 175
				bmi := 24.5
				profile := &models.UserProfile{
					ID:     1,
					UserID: 1,
					Weight: &weight,
					Height: &height,
					BMI:    &bmi,
				}
				m.On("UpdateUserProfile", mock.MatchedBy(func(p *models.UserProfile) bool {
					return p.UserID == 1
				})).Return(profile, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Profile updated successfully",
		},
		{
			name:           "invalid JSON",
			userID:         1,
			requestBody:    nil,
			setupMock:      func(m *mocks.MockUserProfileService) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name:   "profile not found",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 75.0,
			},
			setupMock: func(m *mocks.MockUserProfileService) {
				m.On("UpdateUserProfile", mock.AnythingOfType("*models.UserProfile")).Return(nil, errors.New("profile not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Profile not found",
		},
		{
			name:   "service update error",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 75.0,
			},
			setupMock: func(m *mocks.MockUserProfileService) {
				m.On("UpdateUserProfile", mock.AnythingOfType("*models.UserProfile")).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Profile not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupProfileControllerWithMock()
			tt.setupMock(mockService)

			router := setupProfileTestRouter()
			router.Use(addProfileAuthMiddleware(tt.userID))
			router.PUT("/profile", controller.UpdateUserProfile)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest("PUT", "/profile", bytes.NewBuffer(body))
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

func TestDeleteUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		setupMock      func(*mocks.MockUserProfileService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful deletion",
			userID: 1,
			setupMock: func(m *mocks.MockUserProfileService) {
				m.On("DeleteUserProfile", uint(1)).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Profile deleted successfully",
		},
		{
			name:   "service error",
			userID: 1,
			setupMock: func(m *mocks.MockUserProfileService) {
				m.On("DeleteUserProfile", uint(1)).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to delete profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupProfileControllerWithMock()
			tt.setupMock(mockService)

			router := setupProfileTestRouter()
			router.Use(addProfileAuthMiddleware(tt.userID))
			router.DELETE("/profile", controller.DeleteUserProfile)

			req := httptest.NewRequest("DELETE", "/profile", nil)
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

func TestDeleteUserProfileUnauthorized(t *testing.T) {
	controller, _ := setupProfileControllerWithMock()
	router := setupProfileTestRouter()
	router.DELETE("/profile", controller.DeleteUserProfile)

	req := httptest.NewRequest("DELETE", "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["message"])
}

func TestPatchUserProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		requestBody    map[string]interface{}
		setupMock      func(*mocks.MockUserProfileService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:   "successful patch with weight update",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 80.0,
			},
			setupMock: func(m *mocks.MockUserProfileService) {
				m.On("PatchUserProfile", uint(1), mock.AnythingOfType("map[string]interface {}")).Return(nil)
				weight := 80
				height := 175
				bmi := 26.12
				m.On("GetUserProfile", uint(1)).Return(&models.UserProfile{
					ID:     1,
					UserID: 1,
					Weight: &weight,
					Height: &height,
					BMI:    &bmi,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Profile patched successfully",
		},
		{
			name:   "successful patch without BMI recalculation",
			userID: 1,
			requestBody: map[string]interface{}{
				"bloodline": true,
			},
			setupMock: func(m *mocks.MockUserProfileService) {
				m.On("PatchUserProfile", uint(1), mock.AnythingOfType("map[string]interface {}")).Return(nil)
				bloodline := true
				weight := 75
				height := 175
				bmi := 24.49
				m.On("GetUserProfile", uint(1)).Return(&models.UserProfile{
					ID:        1,
					UserID:    1,
					Bloodline: &bloodline,
					Weight:    &weight,
					Height:    &height,
					BMI:       &bmi,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Profile patched successfully",
		},
		{
			name:           "invalid JSON",
			userID:         1,
			requestBody:    nil,
			setupMock:      func(m *mocks.MockUserProfileService) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name:   "profile not found",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 80.0,
			},
			setupMock: func(m *mocks.MockUserProfileService) {
				m.On("PatchUserProfile", uint(1), mock.AnythingOfType("map[string]interface {}")).Return(errors.New("profile not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Failed to patch profile",
		},
		{
			name:   "service patch error",
			userID: 1,
			requestBody: map[string]interface{}{
				"weight": 80.0,
			},
			setupMock: func(m *mocks.MockUserProfileService) {
				m.On("PatchUserProfile", uint(1), mock.AnythingOfType("map[string]interface {}")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Failed to patch profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupProfileControllerWithMock()
			tt.setupMock(mockService)

			router := setupProfileTestRouter()
			router.Use(addProfileAuthMiddleware(tt.userID))
			router.PATCH("/profile", controller.PatchUserProfile)

			var body []byte
			if tt.requestBody != nil {
				body, _ = json.Marshal(tt.requestBody)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest("PATCH", "/profile", bytes.NewBuffer(body))
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

func TestPatchUserProfileUnauthorized(t *testing.T) {
	controller, _ := setupProfileControllerWithMock()
	router := setupProfileTestRouter()
	router.PATCH("/profile", controller.PatchUserProfile)

	requestBody := map[string]interface{}{
		"weight": 80.0,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("PATCH", "/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["message"])
}
