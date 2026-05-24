package routes

import (
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(router *gin.Engine, userController *controllers.UserController, auditMiddleware gin.HandlerFunc) {
	userRoutesPublic := router.Group("/users")
	{
		userRoutesPublic.POST("/", userController.RegisterUser)
		userRoutesPublic.POST("/login", userController.LoginUser)
		userRoutesPublic.POST("/forgot-password", userController.ForgotPassword)
		userRoutesPublic.POST("/reset-password", userController.ResetPassword)
	}

	userRoutesPrivate := router.Group("/users")
	userRoutesPrivate.Use(middleware.AuthMiddleware())
	{
		userRoutesPrivate.GET("/me", userController.GetCurrentUser)
		userRoutesPrivate.PUT("/me", userController.UpdateUser)
		userRoutesPrivate.PATCH("/me", userController.PatchUser)
	}

	adminRoutes := router.Group("/admin")
	adminRoutes.Use(middleware.AuthMiddleware())
	adminRoutes.Use(auditMiddleware)
	{
		adminRoutes.GET("/users", userController.ListAllUsers)
		adminRoutes.PATCH("/users/:id", userController.AdminPatchUser)
		adminRoutes.DELETE("/users/:id", userController.AdminDeleteUser)
	}
}
