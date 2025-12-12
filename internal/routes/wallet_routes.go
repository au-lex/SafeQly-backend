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

	// Funding with Paystack
	wallet.Post("/fund", handlers.FundAccount)
	wallet.Get("/paystack/callback", handlers.PaystackCallback) // Paystack callback

	// Bank utilities
	wallet.Get("/banks", handlers.GetBanks) // Get list of banks
	wallet.Get("/resolve-account", handlers.ResolveAccountNumber) // Verify account number

	// Bank accounts
	wallet.Post("/bank-account", handlers.AddBankAccount)
	wallet.Get("/bank-accounts", handlers.GetBankAccounts)
	wallet.Put("/bank-account/:id/set-default", handlers.SetDefaultBankAccount)
	wallet.Delete("/bank-account/:id", handlers.DeleteBankAccount)

	// Withdrawal with Paystack
	wallet.Post("/withdraw", handlers.WithdrawFunds)

	// Transaction history
	wallet.Get("/transactions", handlers.GetTransactionHistory)
	wallet.Get("/transaction/:id", handlers.GetTransactionByID)
}