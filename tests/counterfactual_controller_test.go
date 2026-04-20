package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"diabetify/internal/controllers"
	"diabetify/internal/models"
	"diabetify/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupCounterfactualTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func setupCounterfactualControllerWithMocks() (*controllers.CounterfactualController, *mocks.MockCounterfactualJobRepository, *mocks.MockCounterfactualJobService) {
	mockRepo := new(mocks.MockCounterfactualJobRepository)
	mockSvc := new(mocks.MockCounterfactualJobService)

	controller := controllers.NewCounterfactualController(mockRepo, mockSvc)
	return controller, mockRepo, mockSvc
}

func addCounterfactualAuthMiddleware(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func TestCounterfactualHealth(t *testing.T) {
	tests := []struct {
		name          string
		serviceStatus map[string]interface{}
		expectedHTTP  int
		expectedMsg   string
	}{
		{
			name: "healthy service",
			serviceStatus: map[string]interface{}{
				"running":            true,
				"rabbitmq_connected": true,
				"request_queue":      "ml.cf.request",
				"response_queue":     "ml.cf.response",
			},
			expectedHTTP: http.StatusOK,
			expectedMsg:  "Counterfactual service is healthy",
		},
		{
			name: "unhealthy service",
			serviceStatus: map[string]interface{}{
				"running":            false,
				"rabbitmq_connected": false,
			},
			expectedHTTP: http.StatusServiceUnavailable,
			expectedMsg:  "Counterfactual service is not healthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, _, mockSvc := setupCounterfactualControllerWithMocks()
			mockSvc.On("GetStatus").Return(tt.serviceStatus)

			router := setupCounterfactualTestRouter()
			router.GET("/counterfactual/health", controller.Health)

			req := httptest.NewRequest("GET", "/counterfactual/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedHTTP, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedMsg, response["message"])

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestCounterfactualGetJobStatusFieldMapping(t *testing.T) {
	reasonCode := "OK"
	infoMessage := "Actionable counterfactual generated for controller testing."

	tests := []struct {
		name             string
		jobStatus        string
		errorMessage     *string
		expectMessageKey bool
		expectErrorKey   bool
	}{
		{
			name:             "completed uses message field",
			jobStatus:        models.CFJobStatusCompleted,
			errorMessage:     &infoMessage,
			expectMessageKey: true,
			expectErrorKey:   false,
		},
		{
			name:             "infeasible uses message field",
			jobStatus:        models.CFJobStatusInfeasible,
			errorMessage:     &infoMessage,
			expectMessageKey: true,
			expectErrorKey:   false,
		},
		{
			name:             "failed uses error field",
			jobStatus:        models.CFJobStatusFailed,
			errorMessage:     &infoMessage,
			expectMessageKey: false,
			expectErrorKey:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockRepo, _ := setupCounterfactualControllerWithMocks()
			job := &models.CounterfactualJob{
				ID:           "job-1",
				UserID:       1,
				Status:       tt.jobStatus,
				ReasonCode:   &reasonCode,
				ErrorMessage: tt.errorMessage,
			}
			mockRepo.On("GetJobByID", "job-1").Return(job, nil)

			router := setupCounterfactualTestRouter()
			router.Use(addCounterfactualAuthMiddleware(1))
			router.GET("/counterfactual/job/:job_id/status", controller.GetJobStatus)

			req := httptest.NewRequest("GET", "/counterfactual/job/job-1/status", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			data := response["data"].(map[string]interface{})
			_, hasMessage := data["message"]
			_, hasError := data["error"]

			assert.Equal(t, tt.expectMessageKey, hasMessage)
			assert.Equal(t, tt.expectErrorKey, hasError)

			mockRepo.AssertExpectations(t)
		})
	}
}
