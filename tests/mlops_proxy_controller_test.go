package tests

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"diabetify/internal/controllers"
	"diabetify/internal/models"
	"diabetify/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func startMLOpsMock(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Setenv("MLOPS_SERVICE_URL", srv.URL)
	t.Cleanup(srv.Close)
}

func buildMLOpsRouter(method, ginPath, targetPath string, userID uint, repo *mocks.MockUserRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	h := controllers.MLOpsProxy(targetPath, repo)
	switch method {
	case http.MethodGet:
		router.GET(ginPath, h)
	case http.MethodPost:
		router.POST(ginPath, h)
	case http.MethodPut:
		router.PUT(ginPath, h)
	default:
		router.Handle(method, ginPath, h)
	}
	return router
}

func TestMLOpsProxyForwardsStatusAndBody(t *testing.T) {
	startMLOpsMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"ok","job_id":"abc-123"}`))
	})

	repo := new(mocks.MockUserRepository)
	router := buildMLOpsRouter(http.MethodPost, "/trigger", "/training/trigger", 1, repo)

	req := httptest.NewRequest(http.MethodPost, "/trigger", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "job_id")
}

func TestMLOpsProxySetsXUserIDHeader(t *testing.T) {
	var receivedXUserID string
	startMLOpsMock(t, func(w http.ResponseWriter, r *http.Request) {
		receivedXUserID = r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	repo := new(mocks.MockUserRepository)
	router := buildMLOpsRouter(http.MethodPost, "/trigger", "/training/trigger", 7, repo)

	req := httptest.NewRequest(http.MethodPost, "/trigger", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "7", receivedXUserID)
}

func TestMLOpsProxyForwardsQueryParams(t *testing.T) {
	var receivedQuery string
	startMLOpsMock(t, func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	repo := new(mocks.MockUserRepository)
	router := buildMLOpsRouter(http.MethodPost, "/trigger", "/training/trigger", 1, repo)

	req := httptest.NewRequest(http.MethodPost, "/trigger?page=2&size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "page=2&size=10", receivedQuery)
}

func TestMLOpsProxyResolvesGinPathParams(t *testing.T) {
	var receivedPath string
	startMLOpsMock(t, func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	repo := new(mocks.MockUserRepository)
	router := buildMLOpsRouter(http.MethodGet, "/datasets/:id", "/datasets/:id", 1, repo)

	req := httptest.NewRequest(http.MethodGet, "/datasets/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "/datasets/42", receivedPath)
}

func TestMLOpsProxyReturnsGatewayErrorWhenUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	t.Setenv("MLOPS_SERVICE_URL", srv.URL)

	repo := new(mocks.MockUserRepository)
	router := buildMLOpsRouter(http.MethodPost, "/trigger", "/training/trigger", 1, repo)

	req := httptest.NewRequest(http.MethodPost, "/trigger", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "MLOps service unreachable")
}

func TestMLOpsProxyEnrichesDatasetListWithUploaderName(t *testing.T) {
	mlopsBody, _ := json.Marshal(map[string]interface{}{
		"data": []map[string]interface{}{
			{"id": 1, "name": "dataset.csv", "uploaded_by": 10},
		},
	})
	startMLOpsMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(mlopsBody)
	})

	repo := new(mocks.MockUserRepository)
	repo.On("GetUserByID", uint(10)).Return(&models.User{Name: "Alice"}, nil)
	router := buildMLOpsRouter(http.MethodGet, "/datasets", "/datasets", 1, repo)

	req := httptest.NewRequest(http.MethodGet, "/datasets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	row := body["data"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "Alice", row["uploader_name"])
	repo.AssertExpectations(t)
}

func TestMLOpsProxyEnrichesTrainingJobsList(t *testing.T) {
	mlopsBody, _ := json.Marshal(map[string]interface{}{
		"data": []map[string]interface{}{
			{"id": 1, "triggered_by": 2, "trained_by": 3},
		},
	})
	startMLOpsMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(mlopsBody)
	})

	repo := new(mocks.MockUserRepository)
	repo.On("GetUserByID", uint(2)).Return(&models.User{Name: "Bob"}, nil)
	repo.On("GetUserByID", uint(3)).Return(&models.User{Name: "Carol"}, nil)
	router := buildMLOpsRouter(http.MethodGet, "/training/jobs", "/training/jobs", 1, repo)

	req := httptest.NewRequest(http.MethodGet, "/training/jobs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	row := body["data"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "Bob", row["triggered_by_name"])
	assert.Equal(t, "Carol", row["trained_by_name"])
	repo.AssertExpectations(t)
}

func TestMLOpsProxyEnrichesModelsList(t *testing.T) {
	mlopsBody, _ := json.Marshal(map[string]interface{}{
		"data": []map[string]interface{}{
			{"id": 1, "version": "v2", "trained_by": 5},
		},
	})
	startMLOpsMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(mlopsBody)
	})

	repo := new(mocks.MockUserRepository)
	repo.On("GetUserByID", uint(5)).Return(&models.User{Name: "Dave"}, nil)
	router := buildMLOpsRouter(http.MethodGet, "/models", "/models", 1, repo)

	req := httptest.NewRequest(http.MethodGet, "/models", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	row := body["data"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "Dave", row["trained_by_name"])
	repo.AssertExpectations(t)
}

func TestMLOpsProxyFallbackNameForUnknownUser(t *testing.T) {
	mlopsBody, _ := json.Marshal(map[string]interface{}{
		"data": []map[string]interface{}{
			{"id": 1, "uploaded_by": 99},
		},
	})
	startMLOpsMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(mlopsBody)
	})

	repo := new(mocks.MockUserRepository)
	repo.On("GetUserByID", uint(99)).Return(nil, errors.New("not found"))
	router := buildMLOpsRouter(http.MethodGet, "/datasets", "/datasets", 1, repo)

	req := httptest.NewRequest(http.MethodGet, "/datasets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	row := body["data"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "User #99", row["uploader_name"])
	repo.AssertExpectations(t)
}

func TestMLOpsProxyDoesNotEnrichWhenStatusNotOK(t *testing.T) {
	mlopsBody := `{"error":"dataset not found"}`
	startMLOpsMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(mlopsBody))
	})

	repo := new(mocks.MockUserRepository)
	router := buildMLOpsRouter(http.MethodGet, "/datasets", "/datasets", 1, repo)

	req := httptest.NewRequest(http.MethodGet, "/datasets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, mlopsBody, w.Body.String())
	repo.AssertNotCalled(t, "GetUserByID", mock.Anything)
}
