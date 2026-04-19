package mocks

import (
	"context"
	"diabetify/internal/models"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockUserService is a mock implementation of the user service
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(email, password, name, role string, gender, dob *string) (*models.User, error) {
	args := m.Called(email, password, name, role, gender, dob)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(id uint) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetAllUsers() ([]*models.User, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) VerifyPassword(hashedPassword, password string) bool {
	args := m.Called(hashedPassword, password)
	return args.Bool(0)
}

func (m *MockUserService) UpdateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserService) DeleteUser(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserService) SetUserVerified(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockUserService) PatchUser(id uint, data map[string]interface{}) error {
	args := m.Called(id, data)
	return args.Error(0)
}

func (m *MockUserService) ForgotPassword(email string) (string, error) {
	args := m.Called(email)
	return args.String(0), args.Error(1)
}

func (m *MockUserService) ResetPassword(email, code, newPassword string) error {
	args := m.Called(email, code, newPassword)
	return args.Error(0)
}

// MockMLClient is a mock implementation of the ML client interface
type MockMLClient struct {
	mock.Mock
}

func (m *MockMLClient) Predict(ctx context.Context, features []float64) (*models.PredictionResponse, error) {
	args := m.Called(ctx, features)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PredictionResponse), args.Error(1)
}

func (m *MockMLClient) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMLClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMLClient) PredictAsync(ctx context.Context, jobID string, features []float64) error {
	args := m.Called(ctx, jobID, features)
	return args.Error(0)
}

func (m *MockMLClient) HealthCheckAsync(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockPredictionJobWorker is a mock implementation of the prediction job worker
type MockPredictionJobWorker struct {
	mock.Mock
}

func (m *MockPredictionJobWorker) Start() {
	m.Called()
}

func (m *MockPredictionJobWorker) Stop() {
	m.Called()
}

func (m *MockPredictionJobWorker) SubmitJob(jobRequest models.PredictionJobRequest) error {
	args := m.Called(jobRequest)
	return args.Error(0)
}

func (m *MockPredictionJobWorker) GetWhatIfResult(jobID string) (map[string]interface{}, bool, error) {
	args := m.Called(jobID)
	return args.Get(0).(map[string]interface{}), args.Bool(1), args.Error(2)
}

func (m *MockPredictionJobWorker) GetStatus() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// MockCounterfactualJobService is a mock implementation of the counterfactual job service
type MockCounterfactualJobService struct {
	mock.Mock
}

func (m *MockCounterfactualJobService) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCounterfactualJobService) Stop() {
	m.Called()
}

func (m *MockCounterfactualJobService) SubmitJob(jobID string, payload map[string]interface{}) error {
	args := m.Called(jobID, payload)
	return args.Error(0)
}

func (m *MockCounterfactualJobService) GetStatus() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// MockUserProfileService is a mock implementation of the user profile service
type MockUserProfileService struct {
	mock.Mock
}

func (m *MockUserProfileService) GetUserProfile(userID uint) (*models.UserProfile, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

func (m *MockUserProfileService) CreateUserProfile(profile *models.UserProfile) (*models.UserProfile, error) {
	args := m.Called(profile)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

func (m *MockUserProfileService) UpdateUserProfile(profile *models.UserProfile) (*models.UserProfile, error) {
	args := m.Called(profile)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

func (m *MockUserProfileService) DeleteUserProfile(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserProfileService) PatchUserProfile(userID uint, data map[string]interface{}) error {
	args := m.Called(userID, data)
	return args.Error(0)
}

// MockVerificationService is a mock implementation of the verification service
type MockVerificationService struct {
	mock.Mock
}

func (m *MockVerificationService) SendVerificationCode(email string) (string, error) {
	args := m.Called(email)
	return args.String(0), args.Error(1)
}

func (m *MockVerificationService) VerifyCode(email, code string) error {
	args := m.Called(email, code)
	return args.Error(0)
}

func (m *MockVerificationService) ResendVerificationCode(email string) (string, error) {
	args := m.Called(email)
	return args.String(0), args.Error(1)
}

// MockActivityService is a mock implementation of the activity service
type MockActivityService struct {
	mock.Mock
}

func (m *MockActivityService) CreateActivity(activity *models.Activity) error {
	args := m.Called(activity)
	return args.Error(0)
}

func (m *MockActivityService) GetActivityByID(id uint) (*models.Activity, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Activity), args.Error(1)
}

func (m *MockActivityService) GetCurrentUserActivities(userID uint, limit int) ([]models.Activity, error) {
	args := m.Called(userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Activity), args.Error(1)
}

func (m *MockActivityService) GetActivitiesByDateRange(userID uint, startDate, endDate time.Time) ([]models.Activity, error) {
	args := m.Called(userID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Activity), args.Error(1)
}

func (m *MockActivityService) UpdateActivity(activity *models.Activity) error {
	args := m.Called(activity)
	return args.Error(0)
}

func (m *MockActivityService) DeleteActivity(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockActivityService) CountUserActivities(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

// MockArticleService is a mock implementation of the article service
type MockArticleService struct {
	mock.Mock
}

func (m *MockArticleService) CreateArticle(article *models.Article) error {
	args := m.Called(article)
	return args.Error(0)
}

func (m *MockArticleService) GetAllArticles() ([]models.Article, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Article), args.Error(1)
}

func (m *MockArticleService) GetArticleByID(id uint) (*models.Article, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Article), args.Error(1)
}

func (m *MockArticleService) GetArticleImage(id uint) ([]byte, string, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).([]byte), args.String(1), args.Error(2)
}

func (m *MockArticleService) UpdateArticle(article *models.Article) error {
	args := m.Called(article)
	return args.Error(0)
}

func (m *MockArticleService) DeleteArticle(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockArticleService) DeleteArticleImage(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockArticleService) SaveArticleImage(id uint, imageData []byte, mimeType string) error {
	args := m.Called(id, imageData, mimeType)
	return args.Error(0)
}

// MockOAuthService is a mock implementation of the oauth service
type MockOAuthService struct {
	mock.Mock
}

func (m *MockOAuthService) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockOAuthService) CreateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}
