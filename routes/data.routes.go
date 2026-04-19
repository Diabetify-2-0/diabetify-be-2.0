package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterDataRoutes registers MLOps proxy routes for Data Scientists.
// All routes require authentication and the DATA_SCIENTIST (or ADMIN) role.
func RegisterDataRoutes(router *gin.Engine) {
	data := router.Group("/data")
	data.Use(middleware.AuthMiddleware())
	data.Use(middleware.RoleMiddleware("DATA_SCIENTIST", "ADMIN"))
	{
		// Dataset management
		data.POST("/datasets", controllers.MLOpsProxy("/datasets"))
		data.GET("/datasets", controllers.MLOpsProxy("/datasets"))
		data.GET("/datasets/:id", controllers.MLOpsProxy("/datasets/:id"))
		data.POST("/datasets/:id/preprocess", controllers.MLOpsProxy("/datasets/:id/preprocess"))

		// Training jobs
		data.POST("/training/trigger", controllers.MLOpsProxy("/training/trigger"))
		data.GET("/training/jobs", controllers.MLOpsProxy("/training/jobs"))
		data.GET("/training/jobs/:id", controllers.MLOpsProxy("/training/jobs/:id"))
		data.POST("/training/jobs/:id/cancel", controllers.MLOpsProxy("/training/jobs/:id/cancel"))

		// Model versions
		data.GET("/models", controllers.MLOpsProxy("/models"))
		data.GET("/models/:id", controllers.MLOpsProxy("/models/:id"))

		// Request approval for a model (body must include model_id)
		data.POST("/approvals", controllers.MLOpsProxy("/approvals"))
	}
}
