package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"
	"diabetify/internal/repository"

	"github.com/gin-gonic/gin"
)

// RegisterExpertRoutes registers MLOps proxy routes for Medical Experts.
// All routes require authentication and the MEDICAL_EXPERT (or ADMIN) role.
func RegisterExpertRoutes(router *gin.Engine, userRepo repository.UserRepository) {
	expert := router.Group("/expert")
	expert.Use(middleware.AuthMiddleware())
	expert.Use(middleware.RoleMiddleware("MEDICAL_EXPERT", "ADMIN"))
	{
		// Browse model versions
		expert.GET("/models", controllers.MLOpsProxy("/models", userRepo))
		expert.GET("/models/:id", controllers.MLOpsProxy("/models/:id", userRepo))

		// Approval workflow
		expert.GET("/approvals", controllers.MLOpsProxy("/approvals", userRepo))
		expert.POST("/approvals/:id/review", controllers.MLOpsProxy("/approvals/:id/review", userRepo))

		// Promote an approved model to live
		expert.POST("/models/:id/promote", controllers.MLOpsProxy("/models/:id/promote", userRepo))
	}
}
