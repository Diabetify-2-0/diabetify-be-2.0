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

		// Model review — approve or reject challenger
		expert.POST("/models/:id/approve", controllers.MLOpsProxy("/models/:id/approve", userRepo))
		expert.POST("/models/:id/reject", controllers.MLOpsProxy("/models/:id/reject", userRepo))

		// Shadow deployment comparison (read-only)
		expert.GET("/shadow/active", controllers.MLOpsProxy("/shadow/active", userRepo))
		expert.GET("/shadow/:deployment_id/stats", controllers.MLOpsProxy("/shadow/:deployment_id/stats", userRepo))
		expert.GET("/shadow/:deployment_id/predictions", controllers.MLOpsProxy("/shadow/:deployment_id/predictions", userRepo))

		// Expert prediction (champion + challenger side-by-side)
		expert.POST("/predict", controllers.MLOpsProxy("/expert/predict", userRepo))
	}
}
