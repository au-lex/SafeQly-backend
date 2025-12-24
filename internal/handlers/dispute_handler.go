package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"SafeQly/internal/database"
	"SafeQly/internal/models"
)

type RaiseDisputeRequest struct {
	EscrowID    uint   `json:"escrow_id" validate:"required"`
	Reason      string `json:"reason" validate:"required"`
	Description string `json:"description" validate:"required"`
}

type ResolveDisputeRequest struct {
	Resolution string `json:"resolution" validate:"required"`
	Winner     string `json:"winner" validate:"required,oneof=buyer seller"`
}

// RaiseDispute allows buyer or seller to raise a dispute 
func RaiseDispute(c *fiber.Ctx) error {
	// Parse form data
	escrowIDStr := c.FormValue("escrow_id")
	reason := c.FormValue("reason")
	description := c.FormValue("description")

	// Validate required fields
	if escrowIDStr == "" || reason == "" || description == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "escrow_id, reason, and description are required",
		})
	}

	// Parse escrow_id
	escrowID, err := strconv.ParseUint(escrowIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid escrow_id",
		})
	}

	userID := c.Locals("user_id").(uint)

	// Find escrow
	var escrow models.Escrow
	if err := database.DB.First(&escrow, uint(escrowID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Escrow not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Check if user is part of this escrow
	if escrow.BuyerID != userID && escrow.SellerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have access to this escrow",
		})
	}

	// Check if escrow can be disputed
	if escrow.Status == models.EscrowReleased || escrow.Status == models.EscrowRejected || escrow.Status == models.EscrowCancelled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Cannot dispute escrow with status: %s", escrow.Status),
		})
	}

	// Check if dispute already exists
	var existingDispute models.Dispute
	if err := database.DB.Where("escrow_id = ? AND status IN ?", uint(escrowID), []string{"open", "in_progress"}).First(&existingDispute).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "A dispute already exists for this escrow",
		})
	}

	// Handle evidence file upload 
	var evidenceURL, evidencePublicID, evidenceFileName string
	file, err := c.FormFile("evidence")
	if err == nil && file != nil {
		// Validate file size (10MB max)
		maxSize := int64(10 * 1024 * 1024)
		if file.Size > maxSize {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "File too large. Maximum size is 10MB",
			})
		}

		// Upload to Cloudinary
		result, err := cloudinaryService.UploadFile(file, "safeqly/dispute-evidence")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to upload evidence file: %v", err),
			})
		}

		evidenceURL = result.SecureURL
		evidencePublicID = result.PublicID
		evidenceFileName = file.Filename
	}

	// Create dispute
	dispute := models.Dispute{
		EscrowID:          uint(escrowID),
		RaisedBy:          userID,
		Reason:            models.DisputeReason(reason),
		Description:       description,
		Evidence:          evidenceURL,
		EvidencePublicID:  evidencePublicID,
		EvidenceFileName:  evidenceFileName,
		Status:            models.DisputeOpen,
	}

	if err := database.DB.Create(&dispute).Error; err != nil {
		// If dispute creation failed and file was uploaded, delete it
		if evidencePublicID != "" {
			cloudinaryService.DeleteFile(evidencePublicID)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create dispute",
		})
	}

	// Update escrow status
	escrow.Status = models.EscrowDisputed
	if err := database.DB.Save(&escrow).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update escrow status",
		})
	}

	// Load relationships
	database.DB.Preload("Escrow").Preload("User").First(&dispute, dispute.ID)

	// Get the user who raised the dispute
	var raisedBy models.User
	database.DB.First(&raisedBy, userID)

	// Determine who to notify (the other party)
	var notifyUserID uint
	if escrow.BuyerID == userID {
		notifyUserID = escrow.SellerID
	} else {
		notifyUserID = escrow.BuyerID
	}

	// 🔔 SEND NOTIFICATION TO THE OTHER PARTY
	if err := notificationService.NotifyDisputeRaised(notifyUserID, raisedBy.FullName, reason, uint(escrowID), dispute.ID); err != nil {
		fmt.Printf("Failed to send notification: %v\n", err)
	}

	response := fiber.Map{
		"message": "Dispute raised successfully. Our team will review it shortly.",
		"dispute": fiber.Map{
			"id":          dispute.ID,
			"escrow_id":   dispute.EscrowID,
			"reason":      dispute.Reason,
			"description": dispute.Description,
			"status":      dispute.Status,
			"created_at":  dispute.CreatedAt,
		},
	}

	// Add evidence info if uploaded
	if evidenceURL != "" {
		response["dispute"].(fiber.Map)["evidence"] = fiber.Map{
			"url":       evidenceURL,
			"filename":  evidenceFileName,
			"public_id": evidencePublicID,
		}
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

func UploadDisputeEvidence(c *fiber.Ctx) error {
	// Get file from form
	file, err := c.FormFile("evidence")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Validate file size (10MB max)
	maxSize := int64(10 * 1024 * 1024)
	if file.Size > maxSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File too large. Maximum size is 10MB",
		})
	}

	// Upload to Cloudinary
	result, err := cloudinaryService.UploadFile(file, "safeqly/dispute-evidence")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to upload file: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message":   "File uploaded successfully",
		"file_url":  result.SecureURL,
		"public_id": result.PublicID,
		"filename":  file.Filename,
	})
}

// GetMyDisputes retrieves all disputes for the authenticated user
func GetMyDisputes(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var disputes []models.Dispute
	if err := database.DB.
		Preload("Escrow.Buyer").
		Preload("Escrow.Seller").
		Preload("User").
		Joins("JOIN escrows ON disputes.escrow_id = escrows.id").
		Where("escrows.buyer_id = ? OR escrows.seller_id = ?", userID, userID).
		Order("disputes.created_at DESC").
		Find(&disputes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve disputes",
		})
	}

	return c.JSON(fiber.Map{
		"disputes": disputes,
		"count":    len(disputes),
	})
}

// GetDisputeByID retrieves a specific dispute
func GetDisputeByID(c *fiber.Ctx) error {
	disputeID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var dispute models.Dispute
	if err := database.DB.
		Preload("Escrow.Buyer").
		Preload("Escrow.Seller").
		Preload("User").
		First(&dispute, disputeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Dispute not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Check if user is part of this dispute
	if dispute.Escrow.BuyerID != userID && dispute.Escrow.SellerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have access to this dispute",
		})
	}

	return c.JSON(fiber.Map{
		"dispute": dispute,
	})
}

// ResolveDispute - Admin resolves the dispute
func ResolveDispute(c *fiber.Ctx) error {
	disputeID := c.Params("id")
	
	req := new(ResolveDisputeRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var dispute models.Dispute
	if err := database.DB.Preload("Escrow").First(&dispute, disputeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Dispute not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Check if dispute is open
	if dispute.Status != models.DisputeOpen && dispute.Status != models.DisputeInProgress {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Cannot resolve dispute with status: %s", dispute.Status),
		})
	}

	// Resolve dispute based on winner
	var winner models.User
	var winnerID uint

	if req.Winner == "buyer" {
		winnerID = dispute.Escrow.BuyerID
	} else {
		winnerID = dispute.Escrow.SellerID
	}

	if err := database.DB.First(&winner, winnerID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve winner information",
		})
	}

	// Use database transaction
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Get buyer to update escrow balance
		var buyer models.User
		if err := tx.First(&buyer, dispute.Escrow.BuyerID).Error; err != nil {
			return err
		}

		// Remove from buyer's escrow balance
		buyer.EscrowBalance -= dispute.Escrow.Amount
		if err := tx.Save(&buyer).Error; err != nil {
			return err
		}

		// Add amount to winner's balance
		winner.Balance += dispute.Escrow.Amount
		if err := tx.Save(&winner).Error; err != nil {
			return err
		}

		// Update dispute
		now := time.Now()
		dispute.Status = models.DisputeResolved
		dispute.Resolution = req.Resolution
		dispute.ResolvedAt = &now
		
		if err := tx.Save(&dispute).Error; err != nil {
			return err
		}

		// Update escrow status
		dispute.Escrow.Status = models.EscrowCancelled
		if err := tx.Save(&dispute.Escrow).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to resolve dispute",
		})
	}

	// 🔔 SEND NOTIFICATIONS TO BOTH PARTIES
	if err := notificationService.NotifyDisputeResolved(dispute.Escrow.BuyerID, req.Winner, req.Resolution, dispute.ID); err != nil {
		fmt.Printf("Failed to send notification to buyer: %v\n", err)
	}
	
	if err := notificationService.NotifyDisputeResolved(dispute.Escrow.SellerID, req.Winner, req.Resolution, dispute.ID); err != nil {
		fmt.Printf("Failed to send notification to seller: %v\n", err)
	}

	return c.JSON(fiber.Map{
		"message": "Dispute resolved successfully",
		"dispute": fiber.Map{
			"id":          dispute.ID,
			"status":      dispute.Status,
			"resolution":  dispute.Resolution,
			"winner":      req.Winner,
			"resolved_at": dispute.ResolvedAt,
		},
	})
}