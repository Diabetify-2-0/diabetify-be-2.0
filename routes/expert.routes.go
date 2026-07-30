package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"
	"diabetify/internal/repository"

	"github.com/gin-gonic/gin"
)

func RegisterExpertRoutes(router *gin.Engine, userRepo repository.UserRepository, auditMiddleware gin.HandlerFunc) {
	expert := router.Group("/expert")
	expert.Use(middleware.AuthMiddleware())
	expert.Use(middleware.RoleMiddleware("MEDICAL_EXPERT", "ADMIN"))
	expert.Use(auditMiddleware)
	{
		// Browse model versions
		expert.GET("/models", controllers.MLOpsProxy("/models", userRepo))
		expert.GET("/models/:id", controllers.MLOpsProxy("/models/:id", userRepo))

		// Model review — approve, reject challenger, or rollback deployed
		expert.POST("/models/:id/approve", controllers.MLOpsProxy("/models/:id/approve", userRepo))
		expert.POST("/models/:id/reject", controllers.MLOpsProxy("/models/:id/reject", userRepo))
		expert.POST("/models/rollback", controllers.MLOpsProxy("/models/rollback", userRepo))

		// Shadow deployment comparison (read-only)
		expert.GET("/shadow/active", controllers.MLOpsProxy("/shadow/active", userRepo))
		expert.GET("/shadow/:deployment_id/stats", controllers.MLOpsProxy("/shadow/:deployment_id/stats", userRepo))
		expert.GET("/shadow/:deployment_id/predictions", controllers.MLOpsProxy("/shadow/:deployment_id/predictions", userRepo))

		// Expert prediction (champion + challenger side-by-side)
		expert.POST("/predict", controllers.MLOpsProxy("/expert/predict", userRepo))
	}
}
