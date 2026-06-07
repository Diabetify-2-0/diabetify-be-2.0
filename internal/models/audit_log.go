package models

import "time"

type AuditLog struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"not null;index"`
	Action      string    `gorm:"not null;size:100"`
	Details     string    `gorm:"size:500"`
	Status      string    `gorm:"not null;size:20"`
	ErrorDetail string    `gorm:"size:1000"`
	CreatedAt   time.Time `gorm:"autoCreateTime;index"`
}
