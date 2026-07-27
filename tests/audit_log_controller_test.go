package tests

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"diabetify/internal/controllers"
	"diabetify/internal/models"
	"diabetify/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupAuditLogController() (*controllers.AuditLogController, *mocks.MockAuditLogRepository, *mocks.MockUserRepository) {
	auditRepo := new(mocks.MockAuditLogRepository)
	userRepo := new(mocks.MockUserRepository)
	ctrl := controllers.NewAuditLogController(auditRepo, userRepo)
	return ctrl, auditRepo, userRepo
}

func TestListAuditLogs(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		query          string
		setupMock      func(*mocks.MockAuditLogRepository, *mocks.MockUserRepository)
		expectedStatus int
		checkBody      func(*testing.T, map[string]interface{})
	}{
		{
			name:  "returns paginated logs with user info",
			query: "?page=1&page_size=2",
			setupMock: func(ar *mocks.MockAuditLogRepository, ur *mocks.MockUserRepository) {
				logs := []*models.AuditLog{
					{ID: 1, UserID: 10, Action: "UPLOAD_DATASET", Details: "", Status: "success", CreatedAt: now},
					{ID: 2, UserID: 10, Action: "TRIGGER_TRAINING", Details: "", Status: "failed", CreatedAt: now},
				}
				ar.On("List", 0, 2, "").Return(logs, int64(2), nil)
				ur.On("GetUserByID", uint(10)).Return(&models.User{Name: "Alice", Role: "DATA_SCIENTIST"}, nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(2), body["total"])
				assert.Equal(t, float64(1), body["page"])
				data := body["data"].([]interface{})
				assert.Len(t, data, 2)
				first := data[0].(map[string]interface{})
				assert.Equal(t, "Alice", first["user_name"])
				assert.Equal(t, "DATA_SCIENTIST", first["role"])
				assert.Equal(t, "UPLOAD_DATASET", first["action"])
			},
		},
		{
			name:  "uses default page and page_size when invalid",
			query: "?page=0&page_size=0",
			setupMock: func(ar *mocks.MockAuditLogRepository, ur *mocks.MockUserRepository) {
				ar.On("List", 0, 20, "").Return([]*models.AuditLog{}, int64(0), nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(1), body["page"])
			},
		},
		{
			name:  "returns 500 when repo fails",
			query: "",
			setupMock: func(ar *mocks.MockAuditLogRepository, ur *mocks.MockUserRepository) {
				ar.On("List", 0, 20, "").Return(nil, int64(0), errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkBody:      nil,
		},
		{
			name:  "falls back to placeholder when user not found",
			query: "",
			setupMock: func(ar *mocks.MockAuditLogRepository, ur *mocks.MockUserRepository) {
				logs := []*models.AuditLog{
					{ID: 1, UserID: 99, Action: "DELETE_USER", Status: "success", CreatedAt: now},
				}
				ar.On("List", 0, 20, "").Return(logs, int64(1), nil)
				ur.On("GetUserByID", uint(99)).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].([]interface{})
				first := data[0].(map[string]interface{})
				assert.Equal(t, "User #99", first["user_name"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, auditRepo, userRepo := setupAuditLogController()
			tt.setupMock(auditRepo, userRepo)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/admin/audit-logs", ctrl.ListAuditLogs)

			req := httptest.NewRequest(http.MethodGet, "/admin/audit-logs"+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkBody != nil {
				var body map[string]interface{}
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				tt.checkBody(t, body)
			}
			auditRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}

func TestGetAuditLog(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		id             string
		setupMock      func(*mocks.MockAuditLogRepository, *mocks.MockUserRepository)
		expectedStatus int
		checkBody      func(*testing.T, map[string]interface{})
	}{
		{
			name: "returns log with user info",
			id:   "5",
			setupMock: func(ar *mocks.MockAuditLogRepository, ur *mocks.MockUserRepository) {
				log := &models.AuditLog{ID: 5, UserID: 10, Action: "APPROVE_MODEL", Details: "model #3", Status: "success", CreatedAt: now}
				ar.On("GetByID", uint(5)).Return(log, nil)
				ur.On("GetUserByID", uint(10)).Return(&models.User{Name: "Bob", Role: "MEDICAL_EXPERT"}, nil)
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(5), body["id"])
				assert.Equal(t, "APPROVE_MODEL", body["action"])
				assert.Equal(t, "model #3", body["details"])
				assert.Equal(t, "Bob", body["user_name"])
				assert.Equal(t, "MEDICAL_EXPERT", body["role"])
			},
		},
		{
			name:           "returns 400 for non-numeric id",
			id:             "abc",
			setupMock:      func(ar *mocks.MockAuditLogRepository, ur *mocks.MockUserRepository) {},
			expectedStatus: http.StatusBadRequest,
			checkBody:      nil,
		},
		{
			name: "returns 404 when not found",
			id:   "999",
			setupMock: func(ar *mocks.MockAuditLogRepository, ur *mocks.MockUserRepository) {
				ar.On("GetByID", uint(999)).Return(nil, errors.New("record not found"))
			},
			expectedStatus: http.StatusNotFound,
			checkBody:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, auditRepo, userRepo := setupAuditLogController()
			tt.setupMock(auditRepo, userRepo)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/admin/audit-logs/:id", ctrl.GetAuditLog)

			req := httptest.NewRequest(http.MethodGet, "/admin/audit-logs/"+tt.id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkBody != nil {
				var body map[string]interface{}
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				tt.checkBody(t, body)
			}
			auditRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}
