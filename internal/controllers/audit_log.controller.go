package controllers

import (
	"diabetify/internal/repository"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type auditLogEntry struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	UserName  string    `json:"user_name"`
	Role      string    `json:"role"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditLogController struct {
	auditRepo repository.AuditLogRepository
	userRepo  repository.UserRepository
}

func NewAuditLogController(auditRepo repository.AuditLogRepository, userRepo repository.UserRepository) *AuditLogController {
	return &AuditLogController{auditRepo: auditRepo, userRepo: userRepo}
}

func (ctrl *AuditLogController) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.DefaultQuery("search", "")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	logs, total, err := ctrl.auditRepo.List(offset, pageSize, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	// Collect unique user IDs
	uniqueIDs := make(map[uint]bool)
	for _, l := range logs {
		uniqueIDs[l.UserID] = true
	}
	userNames := make(map[uint]string, len(uniqueIDs))
	userRoles := make(map[uint]string, len(uniqueIDs))
	for id := range uniqueIDs {
		if u, err := ctrl.userRepo.GetUserByID(id); err == nil && u != nil {
			userNames[id] = u.Name
			userRoles[id] = u.Role
		} else {
			userNames[id] = fmt.Sprintf("User #%d", id)
			userRoles[id] = ""
		}
	}

	entries := make([]auditLogEntry, len(logs))
	for i, l := range logs {
		entries[i] = auditLogEntry{
			ID:        l.ID,
			UserID:    l.UserID,
			UserName:  userNames[l.UserID],
			Role:      userRoles[l.UserID],
			Action:    l.Action,
			Details:   l.Details,
			Status:    l.Status,
			CreatedAt: l.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  entries,
		"total": total,
		"page":  page,
	})
}

func (ctrl *AuditLogController) GetAuditLog(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid ID"})
		return
	}

	log, err := ctrl.auditRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Audit log not found"})
		return
	}

	entry := auditLogEntry{
		ID:        log.ID,
		UserID:    log.UserID,
		Action:    log.Action,
		Details:   log.Details,
		Status:    log.Status,
		CreatedAt: log.CreatedAt,
	}
	if u, err := ctrl.userRepo.GetUserByID(log.UserID); err == nil && u != nil {
		entry.UserName = u.Name
		entry.Role = u.Role
	}

	c.JSON(http.StatusOK, entry)
}
