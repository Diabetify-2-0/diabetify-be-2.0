package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"
	"diabetify/internal/repository"

	"github.com/gin-gonic/gin"
)

// RegisterDataRoutes registers MLOps proxy routes for Data Scientists.
// All routes require authentication and the DATA_SCIENTIST (or ADMIN) role.
func RegisterDataRoutes(router *gin.Engine, userRepo repository.UserRepository) {
	data := router.Group("/data")
	data.Use(middleware.AuthMiddleware())
	data.Use(middleware.RoleMiddleware("DATA_SCIENTIST", "ADMIN"))
	{
		// Dataset management
		data.POST("/datasets", controllers.MLOpsProxy("/datasets", userRepo))
		data.GET("/datasets", controllers.MLOpsProxy("/datasets", userRepo))
		data.GET("/datasets/:id", controllers.MLOpsProxy("/datasets/:id", userRepo))
		data.POST("/datasets/:id/preprocess", controllers.MLOpsProxy("/datasets/:id/preprocess", userRepo))

		// Training jobs
		data.POST("/training/trigger", controllers.MLOpsProxy("/training/trigger", userRepo))
		data.GET("/training/jobs", controllers.MLOpsProxy("/training/jobs", userRepo))
		data.GET("/training/jobs/:id", controllers.MLOpsProxy("/training/jobs/:id", userRepo))
		data.POST("/training/jobs/:id/cancel", controllers.MLOpsProxy("/training/jobs/:id/cancel", userRepo))

		// Model versions
		data.GET("/models", controllers.MLOpsProxy("/models", userRepo))
		data.GET("/models/:id", controllers.MLOpsProxy("/models/:id", userRepo))

		// Request approval for a model (body must include model_id)
		data.POST("/approvals", controllers.MLOpsProxy("/approvals", userRepo))
	}
}
