package routes

import (
	"github.com/gofiber/fiber/v2"
	"SafeQly/internal/handlers"
	"SafeQly/internal/middleware"
)

func SetupWalletRoutes(app *fiber.App) {
	wallet := app.Group("/api/wallet")
	

	
	// Paystack webhook 
	wallet.Post("/paystack/webhook", handlers.PaystackWebhook)
	
	// Paystack callback 
	wallet.Get("/paystack/callback", handlers.PaystackCallback)
	

	// PROTECTED ENDPOINTS 
	
	protected := wallet.Group("", middleware.Protected())
	
	// Wallet Balance
	protected.Get("/balance", handlers.GetWalletBalance)
	
	// Funding
	protected.Post("/fund", handlers.FundAccount)
	
	// Bank Utilities
	protected.Get("/banks", handlers.GetBanks)
	protected.Get("/resolve-account", handlers.ResolveAccountNumber)
	
	// Bank Accounts
	protected.Post("/bank-account", handlers.AddBankAccount)
	protected.Get("/bank-account", handlers.GetBankAccounts)
	protected.Put("/bank-account/:id/set-default", handlers.SetDefaultBankAccount)
	protected.Delete("/bank-account/:id", handlers.DeleteBankAccount)
	
	// Withdrawals
	protected.Post("/withdraw", handlers.WithdrawFunds)
	
	// Transactions
	protected.Get("/transactions", handlers.GetTransactionHistory)
	protected.Get("/transaction/:id", handlers.GetTransactionByID)
	protected.Get("/transaction-status", handlers.GetTransactionByReference)
}