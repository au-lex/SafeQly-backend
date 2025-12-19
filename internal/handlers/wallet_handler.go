package handlers

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	// "io"
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

// Request structs
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

// Webhook payload structs
type PaystackWebhookPayload struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type PaystackChargeData struct {
	ID              int64  `json:"id"`
	Domain          string `json:"domain"`
	Status          string `json:"status"`
	Reference       string `json:"reference"`
	Amount          int64  `json:"amount"`
	Message         string `json:"message"`
	GatewayResponse string `json:"gateway_response"`
	PaidAt          string `json:"paid_at"`
	CreatedAt       string `json:"created_at"`
	Channel         string `json:"channel"`
	Currency        string `json:"currency"`
	Customer        struct {
		ID           int64  `json:"id"`
		Email        string `json:"email"`
		CustomerCode string `json:"customer_code"`
	} `json:"customer"`
}

type PaystackTransferData struct {
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Domain        string `json:"domain"`
	Failures      string `json:"failures"`
	ID            int64  `json:"id"`
	Reason        string `json:"reason"`
	Reference     string `json:"reference"`
	Source        string `json:"source"`
	SourceDetails string `json:"source_details"`
	Status        string `json:"status"`
	TransferCode  string `json:"transfer_code"`
	TransferredAt string `json:"transferred_at"`
	Recipient     struct {
		Domain   string `json:"domain"`
		Type     string `json:"type"`
		Currency string `json:"currency"`
		Name     string `json:"name"`
		Details  struct {
			AccountNumber string `json:"account_number"`
			AccountName   string `json:"account_name"`
			BankCode      string `json:"bank_code"`
			BankName      string `json:"bank_name"`
		} `json:"details"`
		RecipientCode string `json:"recipient_code"`
	} `json:"recipient"`
}

// ============================================================================
// WALLET BALANCE
// ============================================================================
// Update the GetWalletBalance function in wallet.go

func GetWalletBalance(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve balance",
		})
	}

	return c.JSON(fiber.Map{
		"available_balance": user.Balance,        // Money available for use
		"escrow_balance":    user.EscrowBalance,  // Money locked in escrow
		"total_balance":     user.Balance + user.EscrowBalance, // Total funds
		"user": fiber.Map{
			"id":        user.ID,
			"full_name": user.FullName,
			"email":     user.Email,
			"user_tag":  user.UserTag,
		},
	})
}

// ============================================================================
// FUNDING / DEPOSITS
// ============================================================================



func FundAccount(c *fiber.Ctx) error {
	req := new(FundAccountRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := c.Locals("user_id").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user information",
		})
	}

	if req.Amount < 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Minimum deposit amount is ₦100",
		})
	}

	reference := generateTransactionReference("DEP")

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

	// FIXED: Changed to frontend callback URL
	callbackURL := fmt.Sprintf("https://safeqly.vercel.app/payment-callback?reference=%s", reference)

	paymentResp, err := paystackService.InitializePayment(
		user.Email,
		req.Amount,
		reference,
		callbackURL,
	)

	if err != nil {
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

// Keep PaystackCallback for manual verification if needed
func PaystackCallback(c *fiber.Ctx) error {
	reference := c.Query("reference")
	if reference == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing payment reference",
		})
	}

	// Just verify payment status with Paystack
	verifyResp, err := paystackService.VerifyPayment(reference)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to verify payment",
			"message": "Please contact support if money was deducted",
		})
	}

	// Return payment status - webhook will handle crediting
	if verifyResp.Data.Status == "success" {
		return c.JSON(fiber.Map{
			"message":   "Payment successful! Your wallet will be credited shortly.",
			"reference": reference,
			"status":    verifyResp.Data.Status,
			"amount":    float64(verifyResp.Data.Amount) / 100,
		})
	}

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":     "Payment was not successful",
		"status":    verifyResp.Data.Status,
		"reference": reference,
	})
}


// ============================================================================
// WEBHOOK HANDLERS
// ============================================================================

func PaystackWebhook(c *fiber.Ctx) error {
	// Verify webhook signature
	signature := c.Get("x-paystack-signature")
	if signature == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing signature",
		})
	}

	body := c.Body()

	// Verify signature
	if !verifyPaystackSignature(body, signature, paystackService.SecretKey) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid signature",
		})
	}

	// Parse webhook payload
	var payload PaystackWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid payload",
		})
	}

	// Handle different event types
	switch payload.Event {
	case "charge.success":
		return handleChargeSuccess(c, payload.Data)
	case "transfer.success":
		return handleTransferSuccess(c, payload.Data)
	case "transfer.failed":
		return handleTransferFailed(c, payload.Data)
	case "transfer.reversed":
		return handleTransferFailed(c, payload.Data)
	default:
		fmt.Printf("Unhandled webhook event: %s\n", payload.Event)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Event received",
		})
	}
}

func verifyPaystackSignature(payload []byte, signature, secretKey string) bool {
	mac := hmac.New(sha512.New, []byte(secretKey))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func handleChargeSuccess(c *fiber.Ctx, data json.RawMessage) error {
	var chargeData PaystackChargeData
	if err := json.Unmarshal(data, &chargeData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid charge data",
		})
	}

	if chargeData.Status != "success" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Charge not successful",
		})
	}

	// Find transaction by reference
	var transaction models.Transaction
	if err := database.DB.Where("reference = ?", chargeData.Reference).First(&transaction).Error; err != nil {
		fmt.Printf("Transaction not found: %s\n", chargeData.Reference)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Transaction not found",
		})
	}

	// Check if already processed (prevent double credit)
	if transaction.Status == models.TransactionCompleted {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Already processed",
		})
	}

	// Get user
	var user models.User
	if err := database.DB.First(&user, transaction.UserID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Convert amount from kobo to naira
	amountPaid := float64(chargeData.Amount) / 100

	// Use database transaction for atomicity
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Credit user's account
		user.Balance += amountPaid
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		// Update transaction status
		now := time.Now()
		transaction.Status = models.TransactionCompleted
		transaction.CompletedAt = &now
		transaction.Amount = amountPaid

		if err := tx.Save(&transaction).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Failed to process payment: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process payment",
		})
	}

	fmt.Printf("✅ Payment successful: %s - ₦%.2f credited to user %d\n",
		chargeData.Reference, amountPaid, user.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Payment processed",
	})
}

func handleTransferSuccess(c *fiber.Ctx, data json.RawMessage) error {
	var transferData PaystackTransferData
	if err := json.Unmarshal(data, &transferData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid transfer data",
		})
	}

	var transaction models.Transaction
	if err := database.DB.Where("reference = ?", transferData.Reference).First(&transaction).Error; err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Transaction not found",
		})
	}

	now := time.Now()
	transaction.Status = models.TransactionCompleted
	transaction.CompletedAt = &now
	database.DB.Save(&transaction)

	fmt.Printf("✅ Transfer successful: %s\n", transferData.Reference)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Transfer completed",
	})
}

func handleTransferFailed(c *fiber.Ctx, data json.RawMessage) error {
	var transferData PaystackTransferData
	if err := json.Unmarshal(data, &transferData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid transfer data",
		})
	}

	var transaction models.Transaction
	if err := database.DB.Where("reference = ?", transferData.Reference).First(&transaction).Error; err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Transaction not found",
		})
	}

	var user models.User
	if err := database.DB.First(&user, transaction.UserID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Refund user (use database transaction)
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		user.Balance += transaction.Amount
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		transaction.Status = models.TransactionFailed
		if err := tx.Save(&transaction).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Failed to refund: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process refund",
		})
	}

	fmt.Printf("⚠️ Transfer failed: %s - ₦%.2f refunded to user %d\n",
		transferData.Reference, transaction.Amount, user.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Transfer failed, refunded",
	})
}

// ============================================================================
// BANKS
// ============================================================================

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

// ============================================================================
// BANK ACCOUNTS
// ============================================================================

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
		AccountName:   resolved.Data.AccountName,
		BankCode:      req.BankCode,
		RecipientCode: recipientResp.Data.RecipientCode,
		IsDefault:     count == 0,
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

// ============================================================================
// WITHDRAWALS
// ============================================================================

func WithdrawFunds(c *fiber.Ctx) error {
	req := new(WithdrawRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := c.Locals("user_id").(uint)

	if req.Amount < 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Minimum withdrawal amount is ₦100",
		})
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user information",
		})
	}

	if user.Balance < req.Amount {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Insufficient balance. You have ₦%.2f", user.Balance),
		})
	}

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

	reference := generateTransactionReference("WTH")

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

// ============================================================================
// TRANSACTIONS
// ============================================================================

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

func GetTransactionByReference(c *fiber.Ctx) error {
	reference := c.Query("reference")
	if reference == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Reference is required",
		})
	}

	userID := c.Locals("user_id").(uint)

	var transaction models.Transaction
	if err := database.DB.Where("reference = ? AND user_id = ?", reference, userID).First(&transaction).Error; err != nil {
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
		"status":      transaction.Status,
		"completed":   transaction.Status == models.TransactionCompleted,
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func generateTransactionReference(prefix string) string {
	rand.Seed(time.Now().UnixNano())
	timestamp := time.Now().Unix()
	random := rand.Intn(999999)
	return fmt.Sprintf("%s-%d-%06d", prefix, timestamp, random)
}