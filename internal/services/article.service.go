package services

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
)

// ArticleService defines the interface for article service
type ArticleService interface {
	CreateArticle(article *models.Article) error
	GetAllArticles() ([]models.Article, error)
	GetArticleByID(id uint) (*models.Article, error)
	GetArticleImage(id uint) ([]byte, string, error)
	UpdateArticle(article *models.Article) error
	DeleteArticle(id uint) error
	DeleteArticleImage(id uint) error
	SaveArticleImage(id uint, imageData []byte, mimeType string) error
}

// articleService implements ArticleService
type articleService struct {
	repo repository.ArticleRepository
}

// NewArticleService creates a new ArticleService instance
func NewArticleService(repo repository.ArticleRepository) ArticleService {
	return &articleService{
		repo: repo,
	}
}

// CreateArticle creates a new article
func (s *articleService) CreateArticle(article *models.Article) error {
	return s.repo.Create(article)
}

// GetAllArticles retrieves all articles
func (s *articleService) GetAllArticles() ([]models.Article, error) {
	return s.repo.FindAll()
}

// GetArticleByID retrieves an article by ID
func (s *articleService) GetArticleByID(id uint) (*models.Article, error) {
	return s.repo.FindByID(id)
}

// GetArticleImage retrieves the image for an article
func (s *articleService) GetArticleImage(id uint) ([]byte, string, error) {
	return s.repo.GetImage(id)
}

// UpdateArticle updates an existing article
func (s *articleService) UpdateArticle(article *models.Article) error {
	return s.repo.Update(article)
}

// DeleteArticle deletes an article
func (s *articleService) DeleteArticle(id uint) error {
	return s.repo.Delete(id)
}

// DeleteArticleImage deletes the image for an article
func (s *articleService) DeleteArticleImage(id uint) error {
	return s.repo.DeleteImage(id)
}

// SaveArticleImage saves an image for an article
func (s *articleService) SaveArticleImage(id uint, imageData []byte, mimeType string) error {
	return s.repo.SaveImage(id, imageData, mimeType)
}
