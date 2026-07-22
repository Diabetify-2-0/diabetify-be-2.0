package repository

import (
	"diabetify/internal/models"

	"gorm.io/gorm"
)

type AuditLogRepository interface {
	Create(log *models.AuditLog) error
	List(offset, limit int, search string) ([]*models.AuditLog, int64, error)
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

func (r *auditLogRepository) List(offset, limit int, search string) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	q := r.db.Model(&models.AuditLog{})
	if search != "" {
		q = q.Joins("LEFT JOIN users ON users.id = audit_logs.user_id").
			Where("audit_logs.details ILIKE ? OR audit_logs.action ILIKE ? OR users.name ILIKE ?",
				"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := q.Order("audit_logs.created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

func (r *auditLogRepository) GetByID(id uint) (*models.AuditLog, error) {
	var log models.AuditLog
	err := r.db.First(&log, id).Error
	return &log, err
}
