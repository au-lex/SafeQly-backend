package routes

import (
	"github.com/gofiber/fiber/v2"
	"SafeQly/internal/handlers"
	"SafeQly/internal/middleware"
)

func SetupDisputeRoutes(app *fiber.App) {
	dispute := app.Group("/api/dispute", middleware.Protected())

	// Raise a dispute
	dispute.Post("/raise", handlers.RaiseDispute)
	
	// Upload evidence file
	dispute.Post("/upload-evidence", handlers.UploadDisputeEvidence)
	
	// Get all my disputes
	dispute.Get("/my-disputes", handlers.GetMyDisputes)
	
	// Get specific dispute
	dispute.Get("/:id", handlers.GetDisputeByID)
	
	// Resolve dispute (admin only - you can add admin middleware later)
	dispute.Post("/:id/resolve", handlers.ResolveDispute)
}