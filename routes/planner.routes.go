package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPlannerRoutes(router *gin.Engine, plannerController *controllers.PlannerController) {
	plannerRoutes := router.Group("/planner")
	plannerRoutes.Use(middleware.AuthMiddleware())
	{
		plannerRoutes.POST("/goals", plannerController.SaveGoal)
		plannerRoutes.GET("/goals/active", plannerController.GetLatestGoal)
		plannerRoutes.GET("/goals", plannerController.GetGoalHistory)
		plannerRoutes.PATCH("/goals/:id/complete", plannerController.CompleteGoal)
		plannerRoutes.PATCH("/goals/:id/archive", plannerController.ArchiveGoal)
		plannerRoutes.POST("/goals/:id/check-ins", plannerController.RecordCheckIn)
		plannerRoutes.GET("/goals/:id/check-ins", plannerController.GetCheckInHistory)
		plannerRoutes.GET("/goals/:id/check-ins/last", plannerController.GetLastCheckIns)
	}
}
