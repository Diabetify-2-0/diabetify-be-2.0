package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"diabetify/internal/controllers"
	"diabetify/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Test helper functions
func setupVerificationTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupVerificationControllerWithMocks() (*controllers.VerificationController, *mocks.MockVerificationService) {
	mockVerificationService := new(mocks.MockVerificationService)
	controller := controllers.NewVerificationController(mockVerificationService)
	return controller, mockVerificationService
}

func TestNewVerificationController(t *testing.T) {
	mockService := new(mocks.MockVerificationService)
	controller := controllers.NewVerificationController(mockService)
	assert.NotNil(t, controller)
}

func TestSendVerificationCode(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockVerificationService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "successful verification code send",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				mockService.On("SendVerificationCode", "test@example.com").Return("123456", nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Verification code sent successfully",
		},
		{
			name: "invalid email format",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name: "user not found",
			requestBody: map[string]interface{}{
				"email": "nonexistent@example.com",
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				mockService.On("SendVerificationCode", "nonexistent@example.com").Return("", errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
		},
		{
			name: "database error when creating verification",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				mockService.On("SendVerificationCode", "test@example.com").Return("", errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Failed to create verification code",
		},
		{
			name:        "missing email field",
			requestBody: map[string]interface{}{},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupVerificationControllerWithMocks()
			tt.setupMocks(mockService)

			router := setupVerificationTestRouter()
			router.POST("/verify/send", controller.SendVerificationCode)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/verify/send", bytes.NewBuffer(body))
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

func TestVerifyCode(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockVerificationService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "successful verification",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
				"code":  "123456",
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				mockService.On("VerifyCode", "test@example.com", "123456").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Verification successful",
		},
		{
			name: "invalid verification code",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
				"code":  "wrong123",
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				mockService.On("VerifyCode", "test@example.com", "wrong123").Return(errors.New("invalid or expired verification code"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "Invalid or expired verification code",
		},
		{
			name: "database error when setting user verified",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
				"code":  "123456",
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				mockService.On("VerifyCode", "test@example.com", "123456").Return(errors.New("failed to verify user"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "Invalid or expired verification code",
		},
		{
			name: "invalid request format",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
				// Missing code field
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
		{
			name: "invalid email format",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
				"code":  "123456",
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				// No mocks needed as validation will fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupVerificationControllerWithMocks()
			tt.setupMocks(mockService)

			router := setupVerificationTestRouter()
			router.POST("/verify", controller.VerifyCode)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/verify", bytes.NewBuffer(body))
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

func TestResendVerificationCode(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMocks     func(*mocks.MockVerificationService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "successful resend verification code",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				// ResendVerificationCode internally calls SendVerificationCode
				mockService.On("SendVerificationCode", "test@example.com").Return("123456", nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Verification code sent successfully",
		},
		{
			name: "user not found for resend",
			requestBody: map[string]interface{}{
				"email": "nonexistent@example.com",
			},
			setupMocks: func(mockService *mocks.MockVerificationService) {
				// ResendVerificationCode internally calls SendVerificationCode
				mockService.On("SendVerificationCode", "nonexistent@example.com").Return("", errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService := setupVerificationControllerWithMocks()
			tt.setupMocks(mockService)

			router := setupVerificationTestRouter()
			router.POST("/verify/resend", controller.ResendVerificationCode)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/verify/resend", bytes.NewBuffer(body))
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

// Benchmark tests
func BenchmarkSendVerificationCode(b *testing.B) {
	controller, mockService := setupVerificationControllerWithMocks()

	mockService.On("SendVerificationCode", "test@example.com").Return("123456", nil)

	router := setupVerificationTestRouter()
	router.POST("/verify/send", controller.SendVerificationCode)

	requestBody := map[string]interface{}{
		"email": "test@example.com",
	}
	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/verify/send", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
