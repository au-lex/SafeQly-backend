package routes

import (
	"github.com/gofiber/fiber/v2"
	"SafeQly/internal/handlers"
	"SafeQly/internal/middleware"
)

func SetupEscrowRoutes(app *fiber.App) {
	escrow := app.Group("/api/escrow", middleware.Protected())
	

		// Get recent escrow users
		escrow.Get("/recent-users", handlers.GetRecentEscrowUsers)

	// Search user by tag
	escrow.Post("/search-user", handlers.SearchUserByTag)
	
	// Create new escrow (buyer)
	escrow.Post("/create", handlers.CreateEscrow)
	
	// Accept escrow (seller)
	escrow.Post("/:id/accept", handlers.AcceptEscrow)
	
	// Reject escrow (seller)
	escrow.Post("/:id/reject", handlers.RejectEscrow)
	
	// Complete escrow (seller marks delivery as done)
	escrow.Post("/:id/complete", handlers.CompleteEscrow)
	
	// Release funds (buyer confirms and releases payment)
	escrow.Post("/:id/release", handlers.ReleaseEscrow)
	
	// Get all my escrows
	escrow.Get("/my-escrows", handlers.GetMyEscrows)
	
	// Get specific escrow
	escrow.Get("/:id", handlers.GetEscrowByID)
	

}
