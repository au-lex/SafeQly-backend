package handlers

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"SafeQly/internal/database"
	"SafeQly/internal/models"
	"SafeQly/internal/services"
)

var paystackService *services.PaystackService


func InitPaystackService() {
	paystackService = services.NewPaystackService()
}

type FundAccountRequest struct {
	Amount          float64 `json:"amount" validate:"required,gt=0"`
	PaymentMethod   string  `json:"payment_method" validate:"required"`
	PaymentProvider string  `json:"payment_provider"`
}

type WithdrawRequest struct {
	Amount        float64 `json:"amount" validate:"required,gt=0"`
	BankAccountID uint    `json:"bank_account_id" validate:"required"`
}

type AddBankAccountRequest struct {
	BankName      string `json:"bank_name" validate:"required"`
	AccountNumber string `json:"account_number" validate:"required"`
	AccountName   string `json:"account_name" validate:"required"`
	BankCode      string `json:"bank_code" validate:"required"`
}

// GetWalletBalance retrieves user's wallet balance
func GetWalletBalance(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve balance",
		})
	}

	return c.JSON(fiber.Map{
		"balance": user.Balance,
		"user": fiber.Map{
			"id":        user.ID,
			"full_name": user.FullName,
			"email":     user.Email,
			"user_tag":  user.UserTag,
		},
	})
}

// FundAccount initiates a deposit/funding transaction with Paystack
func FundAccount(c *fiber.Ctx) error {
	req := new(FundAccountRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := c.Locals("user_id").(uint)

	// Get user details
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user information",
		})
	}

	// Validate minimum amount
	if req.Amount < 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Minimum deposit amount is ₦100",
		})
	}

	// Generate unique reference
	reference := generateTransactionReference("DEP")

	// Create transaction record
	transaction := models.Transaction{
		UserID:          userID,
		Type:            models.TransactionDeposit,
		Amount:          req.Amount,
		Status:          models.TransactionPending,
		Reference:       reference,
		Description:     fmt.Sprintf("Deposit of ₦%.2f", req.Amount),
		PaymentMethod:   req.PaymentMethod,
		PaymentProvider: "paystack",
	}

	if err := database.DB.Create(&transaction).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create transaction",
		})
	}

	// Initialize Paystack payment
	callbackURL := fmt.Sprintf("http://localhost:8080/api/wallet/paystack/callback?reference=%s", reference)
	
	paymentResp, err := paystackService.InitializePayment(
		user.Email,
		req.Amount,
		reference,
		callbackURL,
	)

	if err != nil {
		// Update transaction status to failed
		transaction.Status = models.TransactionFailed
		database.DB.Save(&transaction)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to initialize payment: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Payment initialized. Complete payment to credit your account.",
		"transaction": fiber.Map{
			"id":             transaction.ID,
			"reference":      transaction.Reference,
			"amount":         transaction.Amount,
			"status":         transaction.Status,
			"payment_method": transaction.PaymentMethod,
		},
		"payment_info": fiber.Map{
			"authorization_url": paymentResp.Data.AuthorizationURL,
			"access_code":       paymentResp.Data.AccessCode,
			"reference":         paymentResp.Data.Reference,
		},
	})
}

// PaystackCallback handles Paystack payment callback/webhook
func PaystackCallback(c *fiber.Ctx) error {
	reference := c.Query("reference")
	if reference == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing payment reference",
		})
	}

	// Verify payment with Paystack
	verifyResp, err := paystackService.VerifyPayment(reference)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to verify payment: %v", err),
		})
	}

	// Check if payment was successful
	if verifyResp.Data.Status != "success" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Payment was not successful",
			"status": verifyResp.Data.Status,
		})
	}

	// Find transaction
	var transaction models.Transaction
	if err := database.DB.Where("reference = ?", reference).First(&transaction).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Transaction not found",
		})
	}

	// Check if already completed
	if transaction.Status == models.TransactionCompleted {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Payment already processed",
		})
	}

	// Get user
	var user models.User
	if err := database.DB.First(&user, transaction.UserID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user",
		})
	}

	// Convert amount from kobo to naira
	amountPaid := float64(verifyResp.Data.Amount) / 100

	// Credit user's account
	user.Balance += amountPaid
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update balance",
		})
	}

	// Update transaction status
	now := time.Now()
	transaction.Status = models.TransactionCompleted
	transaction.CompletedAt = &now
	transaction.Amount = amountPaid // Update with actual amount paid

	if err := database.DB.Save(&transaction).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update transaction",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Payment verified and wallet credited successfully",
		"transaction": fiber.Map{
			"id":           transaction.ID,
			"reference":    transaction.Reference,
			"amount":       transaction.Amount,
			"status":       transaction.Status,
			"completed_at": transaction.CompletedAt,
		},
		"new_balance": user.Balance,
	})
}

// GetBanks retrieves list of Nigerian banks
func GetBanks(c *fiber.Ctx) error {
	banks, err := paystackService.GetBanks("nigeria")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to retrieve banks: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"banks": banks.Data,
		"count": len(banks.Data),
	})
}

// ResolveAccountNumber verifies and resolves bank account details
func ResolveAccountNumber(c *fiber.Ctx) error {
	accountNumber := c.Query("account_number")
	bankCode := c.Query("bank_code")

	if accountNumber == "" || bankCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "account_number and bank_code are required",
		})
	}

	resolved, err := paystackService.ResolveAccountNumber(accountNumber, bankCode)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to resolve account: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"account_number": resolved.Data.AccountNumber,
		"account_name":   resolved.Data.AccountName,
	})
}

// AddBankAccount adds a bank account for withdrawals
func AddBankAccount(c *fiber.Ctx) error {
	req := new(AddBankAccountRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := c.Locals("user_id").(uint)

	// Verify account with Paystack first
	resolved, err := paystackService.ResolveAccountNumber(req.AccountNumber, req.BankCode)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to verify bank account. Please check your details.",
		})
	}

	// Check if account already exists
	var existingAccount models.BankAccount
	if err := database.DB.Where("user_id = ? AND account_number = ?", userID, req.AccountNumber).First(&existingAccount).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Bank account already exists",
		})
	}

	// Create transfer recipient in Paystack
	recipientResp, err := paystackService.CreateTransferRecipient(
		resolved.Data.AccountName,
		req.AccountNumber,
		req.BankCode,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create transfer recipient: %v", err),
		})
	}

	// Check if user has any bank accounts
	var count int64
	database.DB.Model(&models.BankAccount{}).Where("user_id = ?", userID).Count(&count)

	// Create bank account with Paystack recipient code
	bankAccount := models.BankAccount{
		UserID:        userID,
		BankName:      req.BankName,
		AccountNumber: req.AccountNumber,
		AccountName:   resolved.Data.AccountName, // Use verified name from Paystack
		BankCode:      req.BankCode,
		RecipientCode: recipientResp.Data.RecipientCode, // Store Paystack recipient code
		IsDefault:     count == 0, // First account becomes default
	}

	if err := database.DB.Create(&bankAccount).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add bank account",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":      "Bank account verified and added successfully",
		"bank_account": bankAccount,
	})
}

// GetBankAccounts retrieves all bank accounts for the user
func GetBankAccounts(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var bankAccounts []models.BankAccount
	if err := database.DB.Where("user_id = ?", userID).Order("is_default DESC, created_at DESC").Find(&bankAccounts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve bank accounts",
		})
	}

	return c.JSON(fiber.Map{
		"bank_accounts": bankAccounts,
		"count":         len(bankAccounts),
	})
}

// SetDefaultBankAccount sets a bank account as default
func SetDefaultBankAccount(c *fiber.Ctx) error {
	accountID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var bankAccount models.BankAccount
	if err := database.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&bankAccount).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Bank account not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Unset all other defaults for this user
	database.DB.Model(&models.BankAccount{}).Where("user_id = ?", userID).Update("is_default", false)

	// Set this as default
	bankAccount.IsDefault = true
	if err := database.DB.Save(&bankAccount).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set default account",
		})
	}

	return c.JSON(fiber.Map{
		"message":      "Default bank account updated",
		"bank_account": bankAccount,
	})
}

// DeleteBankAccount removes a bank account
func DeleteBankAccount(c *fiber.Ctx) error {
	accountID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var bankAccount models.BankAccount
	if err := database.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&bankAccount).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Bank account not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	if err := database.DB.Delete(&bankAccount).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete bank account",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Bank account deleted successfully",
	})
}

// WithdrawFunds initiates a withdrawal with Paystack
func WithdrawFunds(c *fiber.Ctx) error {
	req := new(WithdrawRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := c.Locals("user_id").(uint)

	// Validate minimum withdrawal
	if req.Amount < 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Minimum withdrawal amount is ₦100",
		})
	}

	// Get user
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user information",
		})
	}

	// Check balance
	if user.Balance < req.Amount {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Insufficient balance. You have ₦%.2f", user.Balance),
		})
	}

	// Get bank account
	var bankAccount models.BankAccount
	if err := database.DB.Where("id = ? AND user_id = ?", req.BankAccountID, userID).First(&bankAccount).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Bank account not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Generate reference
	reference := generateTransactionReference("WTH")

	// Create transaction
	transaction := models.Transaction{
		UserID:        userID,
		Type:          models.TransactionWithdrawal,
		Amount:        req.Amount,
		Status:        models.TransactionPending,
		Reference:     reference,
		Description:   fmt.Sprintf("Withdrawal of ₦%.2f to %s", req.Amount, bankAccount.BankName),
		BankName:      bankAccount.BankName,
		AccountNumber: bankAccount.AccountNumber,
		AccountName:   bankAccount.AccountName,
	}

	if err := database.DB.Create(&transaction).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create withdrawal",
		})
	}

	// Deduct from balance first
	user.Balance -= req.Amount
	if err := database.DB.Save(&user).Error; err != nil {
		database.DB.Delete(&transaction)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process withdrawal",
		})
	}

	// Initiate transfer with Paystack
	transferResp, err := paystackService.InitiateTransfer(
		bankAccount.RecipientCode,
		req.Amount,
		fmt.Sprintf("Withdrawal to %s", bankAccount.AccountName),
		reference,
	)

	if err != nil {
		// Rollback: Credit back user's account and mark transaction as failed
		user.Balance += req.Amount
		database.DB.Save(&user)

		transaction.Status = models.TransactionFailed
		database.DB.Save(&transaction)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to initiate transfer: %v", err),
		})
	}

	// Update transaction with transfer details
	if transferResp.Data.Status == "success" {
		now := time.Now()
		transaction.Status = models.TransactionCompleted
		transaction.CompletedAt = &now
	}
	database.DB.Save(&transaction)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Withdrawal initiated successfully. Funds will be transferred shortly.",
		"transaction": fiber.Map{
			"id":             transaction.ID,
			"reference":      transaction.Reference,
			"amount":         transaction.Amount,
			"status":         transaction.Status,
			"bank_name":      transaction.BankName,
			"account_number": transaction.AccountNumber,
		},
		"new_balance": user.Balance,
		"transfer_info": fiber.Map{
			"transfer_code": transferResp.Data.TransferCode,
			"status":        transferResp.Data.Status,
		},
	})
}

// GetTransactionHistory retrieves user's transaction history
func GetTransactionHistory(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	txType := c.Query("type")

	query := database.DB.Where("user_id = ?", userID)

	if txType != "" {
		query = query.Where("type = ?", txType)
	}

	var transactions []models.Transaction
	if err := query.Order("created_at DESC").Find(&transactions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve transactions",
		})
	}

	return c.JSON(fiber.Map{
		"transactions": transactions,
		"count":        len(transactions),
	})
}

// GetTransactionByID retrieves a specific transaction
func GetTransactionByID(c *fiber.Ctx) error {
	txID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var transaction models.Transaction
	if err := database.DB.Where("id = ? AND user_id = ?", txID, userID).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Transaction not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	return c.JSON(fiber.Map{
		"transaction": transaction,
	})
}

// Helper function to generate transaction reference
func generateTransactionReference(prefix string) string {
	rand.Seed(time.Now().UnixNano())
	timestamp := time.Now().Unix()
	random := rand.Intn(999999)
	return fmt.Sprintf("%s-%d-%06d", prefix, timestamp, random)
}