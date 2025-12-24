package routes

import (
    "github.com/gofiber/fiber/v2"
    "SafeQly/internal/handlers"
    "SafeQly/internal/middleware"
)

func SetupAdminRoutes(app *fiber.App) {
    adminHandler := handlers.NewAdminHandler()

    
    adminAuth := app.Group("/api/admin/auth")
    

    adminAuth.Post("/login", adminHandler.AdminLogin)
    

    adminAuth.Post("/initialize", adminHandler.InitializeFirstAdmin)

    // Protected admin routes
    admin := app.Group("/api/admin", middleware.Protected(), middleware.AdminOnly())

    // Admin profile
    admin.Get("/profile", adminHandler.GetAdminProfile)
    
		// Admin creation
    admin.Post("/create", adminHandler.CreateAdmin)

    // Dashboard
    admin.Get("/dashboard", adminHandler.GetDashboardStats)

    // User Management
    admin.Get("/users", adminHandler.GetAllUsers)
    admin.Get("/users/:id", adminHandler.GetUserByID)
    admin.Put("/users/:id", adminHandler.UpdateUser)
    admin.Post("/users/:id/suspend", adminHandler.SuspendUser)
    admin.Post("/users/:id/unsuspend", adminHandler.UnsuspendUser)
    admin.Delete("/users/:id", adminHandler.DeleteUser)

    // Transaction Management
    admin.Get("/transactions", adminHandler.GetAllTransactions)

    // Dispute Management
    admin.Get("/disputes", adminHandler.GetAllDisputes)
    admin.Get("/disputes/:id", adminHandler.GetDisputeByID)
    admin.Post("/disputes/:id/resolve", adminHandler.ResolveDispute)


    // Withdrawal management (NEW)
	admin.Get("/withdrawals/pending", adminHandler.GetPendingWithdrawals)
	admin.Get("/withdrawals/stats", adminHandler.GetWithdrawalStats)
	admin.Get("/withdrawals/:id", adminHandler.GetWithdrawalByID)
	admin.Post("/withdrawals/:id/complete", adminHandler.CompleteManualWithdrawal)
	admin.Post("/withdrawals/:id/fail", adminHandler.FailManualWithdrawal)
}

