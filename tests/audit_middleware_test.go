package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"diabetify/internal/middleware"
	"diabetify/internal/models"
	"diabetify/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

func buildAuditRouter(repo *mocks.MockAuditLogRepository, method, path string, userID uint, handlerStatus int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if userID > 0 {
			c.Set("user_id", userID)
		}
		c.Next()
	})
	router.Use(middleware.AuditMiddleware(repo))

	handler := func(c *gin.Context) { c.Status(handlerStatus) }
	switch method {
	case http.MethodPost:
		router.POST(path, handler)
	case http.MethodPut:
		router.PUT(path, handler)
	case http.MethodPatch:
		router.PATCH(path, handler)
	case http.MethodDelete:
		router.DELETE(path, handler)
	default:
		router.GET(path, handler)
	}
	return router
}

func TestAuditMiddlewareRecordsSuccessfulAction(t *testing.T) {
	repo := new(mocks.MockAuditLogRepository)
	repo.On("Create", mock.MatchedBy(func(l *models.AuditLog) bool {
		return l.Action == "UPLOAD_DATASET" && l.Status == "success" && l.UserID == 7
	})).Return(nil)

	router := buildAuditRouter(repo, http.MethodPost, "/data/datasets", 7, http.StatusCreated)

	req := httptest.NewRequest(http.MethodPost, "/data/datasets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	time.Sleep(20 * time.Millisecond)
	repo.AssertExpectations(t)
}

func TestAuditMiddlewareRecordsFailedAction(t *testing.T) {
	repo := new(mocks.MockAuditLogRepository)
	repo.On("Create", mock.MatchedBy(func(l *models.AuditLog) bool {
		return l.Action == "TRIGGER_TRAINING" && l.Status == "failed"
	})).Return(nil)

	router := buildAuditRouter(repo, http.MethodPost, "/data/training/trigger", 3, http.StatusBadRequest)

	req := httptest.NewRequest(http.MethodPost, "/data/training/trigger", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	time.Sleep(20 * time.Millisecond)
	repo.AssertExpectations(t)
}

func TestAuditMiddlewareSkipsNonAuditablePaths(t *testing.T) {
	repo := new(mocks.MockAuditLogRepository)

	router := buildAuditRouter(repo, http.MethodGet, "/health", 5, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	time.Sleep(20 * time.Millisecond)
	repo.AssertNotCalled(t, "Create", mock.Anything)
}

func TestAuditMiddlewareSkipsWhenNoUserID(t *testing.T) {
	repo := new(mocks.MockAuditLogRepository)

	router := buildAuditRouter(repo, http.MethodPost, "/data/datasets", 0, http.StatusCreated)

	req := httptest.NewRequest(http.MethodPost, "/data/datasets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	time.Sleep(20 * time.Millisecond)
	repo.AssertNotCalled(t, "Create", mock.Anything)
}

func TestAuditMiddlewareIncludesParamDetails(t *testing.T) {
	repo := new(mocks.MockAuditLogRepository)
	repo.On("Create", mock.MatchedBy(func(l *models.AuditLog) bool {
		return l.Action == "APPROVE_MODEL" && l.Details == "model #42"
	})).Return(nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user_id", uint(1)); c.Next() })
	router.Use(middleware.AuditMiddleware(repo))
	router.POST("/expert/models/:id/approve", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/expert/models/42/approve", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	time.Sleep(20 * time.Millisecond)
	repo.AssertExpectations(t)
}

func TestAuditMiddlewareDriftTrigger(t *testing.T) {
	repo := new(mocks.MockAuditLogRepository)
	repo.On("Create", mock.MatchedBy(func(l *models.AuditLog) bool {
		return l.Action == "TRIGGER_DRIFT" && l.Status == "success"
	})).Return(nil)

	router := buildAuditRouter(repo, http.MethodPost, "/data/drift/trigger", 2, http.StatusOK)

	req := httptest.NewRequest(http.MethodPost, "/data/drift/trigger", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	time.Sleep(20 * time.Millisecond)
	repo.AssertExpectations(t)
}
