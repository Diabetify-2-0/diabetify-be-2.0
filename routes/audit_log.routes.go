package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterAuditLogRoutes(router *gin.Engine, ctrl *controllers.AuditLogController) {
	admin := router.Group("/audit-logs")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.RoleMiddleware("ADMIN"))
	{
		admin.GET("", ctrl.ListAuditLogs)
		admin.GET("/:id", ctrl.GetAuditLog)
	}
}
