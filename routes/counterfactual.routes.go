package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterCounterfactualRoutes(router *gin.Engine, counterfactualController *controllers.CounterfactualController) {
	router.GET("/counterfactual/health", counterfactualController.Health)

	counterfactualRoutes := router.Group("/counterfactual")
	counterfactualRoutes.Use(middleware.AuthMiddleware())
	{
		counterfactualRoutes.POST("/", counterfactualController.SubmitJob)
		counterfactualRoutes.GET("/job/:job_id/status", counterfactualController.GetJobStatus)
		counterfactualRoutes.GET("/job/:job_id/result", counterfactualController.GetJobResult)
		counterfactualRoutes.POST("/job/:job_id/cancel", counterfactualController.CancelJob)
		counterfactualRoutes.GET("/jobs", counterfactualController.GetUserJobs)
	}
}
