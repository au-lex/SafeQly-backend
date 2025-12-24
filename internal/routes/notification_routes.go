

package routes

import (
	"github.com/gofiber/fiber/v2"
	"SafeQly/internal/handlers"
	"SafeQly/internal/middleware"
)

func SetupNotificationRoutes(app *fiber.App) {
	// Initialize notification service
	handlers.InitNotificationService()

	// Notification routes (all require authentication)
	notifications := app.Group("/api/notifications",  middleware.Protected())
	
	// Get all notifications
	notifications.Get("/", handlers.GetNotifications)
	
	// Get unread count
	notifications.Get("/unread-count", handlers.GetUnreadCount)
	
	// Mark specific notification as read
	notifications.Put("/:id/read", handlers.MarkAsRead)
	
	// Mark all notifications as read
	notifications.Put("/read-all", handlers.MarkAllAsRead)
	
	// Delete specific notification
	notifications.Delete("/:id", handlers.DeleteNotification)
	
	// Delete all read notifications
	notifications.Delete("/read-all", handlers.DeleteAllRead)
}

