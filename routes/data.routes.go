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
		data.GET("/datasets/:id/download", controllers.MLOpsProxy("/datasets/:id/download", userRepo))
		data.POST("/datasets/:id/preprocess", controllers.MLOpsProxy("/datasets/:id/preprocess", userRepo))

		// Training jobs
		data.POST("/training/trigger", controllers.MLOpsProxy("/training/trigger", userRepo))
		data.GET("/training/jobs", controllers.MLOpsProxy("/training/jobs", userRepo))
		data.GET("/training/jobs/:id", controllers.MLOpsProxy("/training/jobs/:id", userRepo))
		data.POST("/training/jobs/:id/cancel", controllers.MLOpsProxy("/training/jobs/:id/cancel", userRepo))

		// Model versions
		data.GET("/models", controllers.MLOpsProxy("/models", userRepo))
		data.GET("/models/:id", controllers.MLOpsProxy("/models/:id", userRepo))

		// Shadow deployment
		data.POST("/shadow/activate", controllers.MLOpsProxy("/shadow/activate", userRepo))
		data.POST("/shadow/deactivate/:deployment_id", controllers.MLOpsProxy("/shadow/deactivate/:deployment_id", userRepo))
		data.GET("/shadow/active", controllers.MLOpsProxy("/shadow/active", userRepo))

		// Drift detection
		data.POST("/drift/trigger", controllers.MLOpsProxy("/drift/trigger", userRepo))
		data.GET("/drift/alerts", controllers.MLOpsProxy("/drift/alerts", userRepo))
		data.POST("/drift/alerts/:id/acknowledge", controllers.MLOpsProxy("/drift/alerts/:id/acknowledge", userRepo))
		data.GET("/drift/analytics", controllers.MLOpsProxy("/drift/analytics", userRepo))
		data.GET("/drift/config", controllers.MLOpsProxy("/drift/config", userRepo))
		data.PUT("/drift/config", controllers.MLOpsProxy("/drift/config", userRepo))
		data.GET("/drift/log/count", controllers.MLOpsProxy("/drift/log/count", userRepo))
	}
}
