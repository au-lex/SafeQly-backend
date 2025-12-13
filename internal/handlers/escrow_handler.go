package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"SafeQly/internal/database"
	"SafeQly/internal/models"
)

type CreateEscrowRequest struct {
	SellerTag    string  `json:"seller_tag" validate:"required"`
	Items        string  `json:"items" validate:"required"`
	Amount       float64 `json:"amount" validate:"required,gt=0"`
	DeliveryDate string  `json:"delivery_date" validate:"required"`
	AttachedFile string  `json:"attached_file"`
}

type SearchUserRequest struct {
	UserTag string `json:"user_tag" validate:"required"`
}

type RejectEscrowRequest struct {
	Reason string `json:"reason" validate:"required"`
}

// SearchUserByTag searches for a user by their tag
func SearchUserByTag(c *fiber.Ctx) error {
	req := new(SearchUserRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := c.Locals("user_id").(uint)

	var user models.User
	if err := database.DB.Where("user_tag = ?", req.UserTag).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	if user.ID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "You cannot create an escrow with yourself",
		})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":      user.ID,
			"name":    user.FullName,
			"tag":     user.UserTag,
			"avatar":  user.Avatar,
			"email":   user.Email,
		},
	})
}

// CreateEscrow creates a new escrow transaction
// Money is moved from buyer's balance to buyer's escrow_balance
func CreateEscrow(c *fiber.Ctx) error {
	req := new(CreateEscrowRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	buyerID := c.Locals("user_id").(uint)

	var seller models.User
	if err := database.DB.Where("user_tag = ?", req.SellerTag).First(&seller).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Seller not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	if seller.ID == buyerID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "You cannot create an escrow with yourself",
		})
	}

	var buyer models.User
	if err := database.DB.First(&buyer, buyerID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve buyer information",
		})
	}

	if buyer.Balance < req.Amount {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Insufficient balance. You have ₦%.2f but need ₦%.2f", buyer.Balance, req.Amount),
		})
	}

	// Use database transaction for atomicity
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Create escrow record
		escrow := models.Escrow{
			BuyerID:      buyerID,
			SellerID:     seller.ID,
			Items:        req.Items,
			Amount:       req.Amount,
			DeliveryDate: req.DeliveryDate,
			AttachedFile: req.AttachedFile,
			Status:       models.EscrowPending,
		}

		if err := tx.Create(&escrow).Error; err != nil {
			return err
		}

		// Move funds from buyer's balance to escrow_balance
		buyer.Balance -= req.Amount
		buyer.EscrowBalance += req.Amount
		
		if err := tx.Save(&buyer).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create escrow",
		})
	}

	// Reload buyer to get updated balances
	database.DB.First(&buyer, buyerID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Escrow created successfully. Waiting for seller to accept.",
		"escrow": fiber.Map{
			"items":         req.Items,
			"amount":        req.Amount,
			"delivery_date": req.DeliveryDate,
			"status":        models.EscrowPending,
			"seller": fiber.Map{
				"id":     seller.ID,
				"name":   seller.FullName,
				"tag":    seller.UserTag,
				"avatar": seller.Avatar,
			},
		},
		"available_balance": buyer.Balance,
		"escrow_balance":    buyer.EscrowBalance,
	})
}

// AcceptEscrow - Seller accepts the escrow
func AcceptEscrow(c *fiber.Ctx) error {
	escrowID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var escrow models.Escrow
	if err := database.DB.First(&escrow, escrowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Escrow not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	if escrow.SellerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only the seller can accept this escrow",
		})
	}

	if escrow.Status != models.EscrowPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Cannot accept escrow with status: %s", escrow.Status),
		})
	}

	now := time.Now()
	escrow.Status = models.EscrowAccepted
	escrow.AcceptedAt = &now
	
	if err := database.DB.Save(&escrow).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to accept escrow",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Escrow accepted successfully. You can now proceed with the transaction.",
		"escrow": fiber.Map{
			"id":          escrow.ID,
			"status":      escrow.Status,
			"accepted_at": escrow.AcceptedAt,
		},
	})
}

// RejectEscrow - Seller rejects the escrow
// Money is returned from buyer's escrow_balance to balance
func RejectEscrow(c *fiber.Ctx) error {
	escrowID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	req := new(RejectEscrowRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var escrow models.Escrow
	if err := database.DB.First(&escrow, escrowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Escrow not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	if escrow.SellerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only the seller can reject this escrow",
		})
	}

	if escrow.Status != models.EscrowPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Cannot reject escrow with status: %s", escrow.Status),
		})
	}

	// Use database transaction
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var buyer models.User
		if err := tx.First(&buyer, escrow.BuyerID).Error; err != nil {
			return err
		}

		// Return funds from escrow_balance to balance
		buyer.EscrowBalance -= escrow.Amount
		buyer.Balance += escrow.Amount
		
		if err := tx.Save(&buyer).Error; err != nil {
			return err
		}

		// Update escrow status
		escrow.Status = models.EscrowRejected
		escrow.RejectionReason = req.Reason
		
		if err := tx.Save(&escrow).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reject escrow",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Escrow rejected. Amount has been refunded to buyer.",
		"escrow": fiber.Map{
			"id":     escrow.ID,
			"status": escrow.Status,
			"reason": escrow.RejectionReason,
		},
	})
}

// CompleteEscrow - Seller marks the delivery as completed
func CompleteEscrow(c *fiber.Ctx) error {
	escrowID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var escrow models.Escrow
	if err := database.DB.First(&escrow, escrowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Escrow not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	if escrow.SellerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only the seller can mark this escrow as completed",
		})
	}

	if escrow.Status != models.EscrowAccepted {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Cannot complete escrow with status: %s", escrow.Status),
		})
	}

	now := time.Now()
	escrow.Status = models.EscrowCompleted
	escrow.CompletedAt = &now
	
	if err := database.DB.Save(&escrow).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to complete escrow",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Delivery marked as completed. Waiting for buyer to release funds.",
		"escrow": fiber.Map{
			"id":           escrow.ID,
			"status":       escrow.Status,
			"completed_at": escrow.CompletedAt,
		},
	})
}

// ReleaseEscrow - Buyer releases funds to seller
// Money moves from buyer's escrow_balance to seller's balance
func ReleaseEscrow(c *fiber.Ctx) error {
	escrowID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var escrow models.Escrow
	if err := database.DB.First(&escrow, escrowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Escrow not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	if escrow.BuyerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only the buyer can release funds",
		})
	}

	if escrow.Status != models.EscrowCompleted {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Cannot release funds. Escrow status: %s. Seller must mark as completed first.", escrow.Status),
		})
	}

	// Use database transaction
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Get buyer and seller
		var buyer, seller models.User
		if err := tx.First(&buyer, escrow.BuyerID).Error; err != nil {
			return err
		}
		if err := tx.First(&seller, escrow.SellerID).Error; err != nil {
			return err
		}

		// Remove from buyer's escrow balance
		buyer.EscrowBalance -= escrow.Amount
		if err := tx.Save(&buyer).Error; err != nil {
			return err
		}

		// Add to seller's available balance
		seller.Balance += escrow.Amount
		if err := tx.Save(&seller).Error; err != nil {
			return err
		}

		// Update escrow status
		now := time.Now()
		escrow.Status = models.EscrowReleased
		escrow.ReleasedAt = &now
		
		if err := tx.Save(&escrow).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to release funds",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Funds released successfully to seller",
		"escrow": fiber.Map{
			"id":          escrow.ID,
			"status":      escrow.Status,
			"amount":      escrow.Amount,
			"released_at": escrow.ReleasedAt,
		},
	})
}

// GetMyEscrows retrieves all escrows for the authenticated user
func GetMyEscrows(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	role := c.Query("role")

	query := database.DB.Preload("Buyer").Preload("Seller")

	switch role {
	case "buyer":
		query = query.Where("buyer_id = ?", userID)
	case "seller":
		query = query.Where("seller_id = ?", userID)
	default:
		query = query.Where("buyer_id = ? OR seller_id = ?", userID, userID)
	}

	var escrows []models.Escrow
	if err := query.Order("created_at DESC").Find(&escrows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve escrows",
		})
	}

	return c.JSON(fiber.Map{
		"escrows": escrows,
		"count":   len(escrows),
	})
}

// GetEscrowByID retrieves a specific escrow
func GetEscrowByID(c *fiber.Ctx) error {
	escrowID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var escrow models.Escrow
	if err := database.DB.
		Preload("Buyer").
		Preload("Seller").
		First(&escrow, escrowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Escrow not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	if escrow.BuyerID != userID && escrow.SellerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have access to this escrow",
		})
	}

	return c.JSON(fiber.Map{
		"escrow": escrow,
	})
}