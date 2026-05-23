package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"diabetify/internal/models"
	"diabetify/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGoogleAuthCreatesUserAndReturnsToken(t *testing.T) {
	originalVerifier := googleTokenVerifier
	googleTokenVerifier = func(ctx context.Context, token string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"email":          "new.google@example.com",
			"name":           "Google User",
			"email_verified": true,
			"aud":            "test-google-client-id",
			"iss":            "https://accounts.google.com",
		}, nil
	}
	defer func() {
		googleTokenVerifier = originalVerifier
	}()

	os.Setenv("JWT_SECRET_KEY", "test-secret-key")
	os.Setenv("GOOGLE_KEY", "test-google-client-id")
	defer os.Unsetenv("JWT_SECRET_KEY")
	defer os.Unsetenv("GOOGLE_KEY")

	mockOAuthService := new(mocks.MockOAuthService)
	mockOAuthService.On("GetUserByEmail", "new.google@example.com").Return(nil, errors.New("user not found"))
	mockOAuthService.On("CreateGoogleUser", "new.google@example.com", "Google User").Return(&models.User{
		ID:       42,
		Email:    "new.google@example.com",
		Name:     "Google User",
		Role:     "USER",
		Verified: true,
	}, nil)

	controller := NewOauthController(mockOAuthService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/oauth/auth/google", controller.GoogleAuth)

	body, _ := json.Marshal(map[string]string{"token": "fake-token"})
	req := httptest.NewRequest(http.MethodPost, "/oauth/auth/google", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Google authentication successful")
	assert.NotEmpty(t, response["data"])

	mockOAuthService.AssertExpectations(t)
}

func TestGoogleAuthRejectsUnexpectedAudience(t *testing.T) {
	originalVerifier := googleTokenVerifier
	googleTokenVerifier = func(ctx context.Context, token string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"email":          "new.google@example.com",
			"name":           "Google User",
			"email_verified": true,
			"aud":            "unexpected-client-id",
			"iss":            "https://accounts.google.com",
		}, nil
	}
	defer func() {
		googleTokenVerifier = originalVerifier
	}()

	os.Setenv("JWT_SECRET_KEY", "test-secret-key")
	os.Setenv("GOOGLE_KEY", "expected-client-id")
	defer os.Unsetenv("JWT_SECRET_KEY")
	defer os.Unsetenv("GOOGLE_KEY")

	mockOAuthService := new(mocks.MockOAuthService)
	controller := NewOauthController(mockOAuthService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/oauth/auth/google", controller.GoogleAuth)

	body, _ := json.Marshal(map[string]string{"token": "fake-token"})
	req := httptest.NewRequest(http.MethodPost, "/oauth/auth/google", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "google token audience does not match configured client")

	mockOAuthService.AssertExpectations(t)
}
