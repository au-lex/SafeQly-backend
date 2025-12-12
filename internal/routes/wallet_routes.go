package routes

import (
	"github.com/gofiber/fiber/v2"
	"SafeQly/internal/handlers"
	"SafeQly/internal/middleware"
)

func SetupWalletRoutes(app *fiber.App) {
	wallet := app.Group("/api/wallet", middleware.Protected())
	// Wallet balance
	wallet.Get("/balance", handlers.GetWalletBalance)

	// Funding
	wallet.Post("/fund", handlers.FundAccount)
	wallet.Post("/complete-deposit/:reference", handlers.CompleteDeposit)

	// Bank accounts
	wallet.Post("/bank-account", handlers.AddBankAccount)
	wallet.Get("/bank-accounts", handlers.GetBankAccounts)
	wallet.Put("/bank-account/:id/set-default", handlers.SetDefaultBankAccount)
	wallet.Delete("/bank-account/:id", handlers.DeleteBankAccount)

	// Withdrawal
	wallet.Post("/withdraw", handlers.WithdrawFunds)

	// Transaction history
	wallet.Get("/transactions", handlers.GetTransactionHistory)
	wallet.Get("/transaction/:id", handlers.GetTransactionByID)
}