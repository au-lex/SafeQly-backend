package routes

import (
	"github.com/gofiber/fiber/v2"
	"SafeQly/internal/handlers"
	"SafeQly/internal/middleware"
)

// SetupUserRoutes sets up all user profile related routes
func SetupUserRoutes(app *fiber.App) {

	user := app.Group("/api/user", middleware.Protected())

	
	// Get user profile
	user.Get("/profile", handlers.GetUserProfile)
	
	// Update user profile
	user.Put("/profile", handlers.UpdateUserProfile)
	
	// Change password
	user.Post("/change-password", handlers.ChangePassword)
	
	// Avatar management
	user.Post("/avatar", handlers.UploadAvatar)
	user.Delete("/avatar", handlers.DeleteAvatar)
}