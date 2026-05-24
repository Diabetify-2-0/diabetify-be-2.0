package repository

import (
	"diabetify/internal/models"

	"gorm.io/gorm"
)

type AuditLogRepository interface {
	Create(log *models.AuditLog) error
	List(offset, limit int) ([]*models.AuditLog, int64, error)
	GetByID(id uint) (*models.AuditLog, error)
}

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(log *models.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *auditLogRepository) List(offset, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	if err := r.db.Model(&models.AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

func (r *auditLogRepository) GetByID(id uint) (*models.AuditLog, error) {
	var log models.AuditLog
	err := r.db.First(&log, id).Error
	return &log, err
}
