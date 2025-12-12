package handlers

import (
	"encoding/base64"
	"fmt"

	"os"
	"path/filepath"
	"strings"
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
	Evidence    string `json:"evidence"` // Base64 encoded image or file URL
}

type ResolveDisputeRequest struct {
	Resolution string `json:"resolution" validate:"required"`
	Winner     string `json:"winner" validate:"required,oneof=buyer seller"`
}

// RaiseDispute allows buyer or seller to raise a dispute
func RaiseDispute(c *fiber.Ctx) error {
	req := new(RaiseDisputeRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userID := c.Locals("user_id").(uint)

	// Find escrow
	var escrow models.Escrow
	if err := database.DB.First(&escrow, req.EscrowID).Error; err != nil {
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
	if err := database.DB.Where("escrow_id = ? AND status IN ?", req.EscrowID, []string{"open", "in_progress"}).First(&existingDispute).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "A dispute already exists for this escrow",
		})
	}

	// Handle evidence file upload (if provided)
	var evidenceURL string
	if req.Evidence != "" {
		// Save the evidence file
		savedPath, err := saveEvidenceFile(req.Evidence, userID, req.EscrowID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to save evidence: %v", err),
			})
		}
		evidenceURL = savedPath
	}

	// Create dispute
	dispute := models.Dispute{
		EscrowID:    req.EscrowID,
		RaisedBy:    userID,
		Reason:      models.DisputeReason(req.Reason),
		Description: req.Description,
		Evidence:    evidenceURL,
		Status:      models.DisputeOpen,
	}

	if err := database.DB.Create(&dispute).Error; err != nil {
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

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Dispute raised successfully. Our team will review it shortly.",
		"dispute": fiber.Map{
			"id":          dispute.ID,
			"escrow_id":   dispute.EscrowID,
			"reason":      dispute.Reason,
			"description": dispute.Description,
			"evidence":    dispute.Evidence,
			"status":      dispute.Status,
			"created_at":  dispute.CreatedAt,
		},
	})
}

// UploadDisputeEvidence handles file upload for dispute evidence
func UploadDisputeEvidence(c *fiber.Ctx) error {
	// Get file from form
	file, err := c.FormFile("evidence")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Validate file type
	allowedTypes := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".pdf":  true,
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedTypes[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file type. Only JPEG, PNG, and PDF are allowed",
		})
	}

	// Validate file size (5MB max)
	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File size exceeds 5MB limit",
		})
	}

	userID := c.Locals("user_id").(uint)
	
	// Create upload directory if it doesn't exist
	uploadDir := "./uploads/disputes"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d_%d%s", userID, time.Now().Unix(), ext)
	filepath := filepath.Join(uploadDir, filename)

	// Save file
	if err := c.SaveFile(file, filepath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	// Return file URL
	fileURL := fmt.Sprintf("/uploads/disputes/%s", filename)

	return c.JSON(fiber.Map{
		"message":  "File uploaded successfully",
		"file_url": fileURL,
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

// ResolveDispute - Admin resolves the dispute (you can implement admin check later)
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

	// Add amount to winner's balance
	winner.Balance += dispute.Escrow.Amount
	if err := database.DB.Save(&winner).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to transfer funds",
		})
	}

	// Update dispute
	now := time.Now()
	dispute.Status = models.DisputeResolved
	dispute.Resolution = req.Resolution
	dispute.ResolvedAt = &now
	
	if err := database.DB.Save(&dispute).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to resolve dispute",
		})
	}

	// Update escrow status
	dispute.Escrow.Status = models.EscrowCancelled
	if err := database.DB.Save(&dispute.Escrow).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update escrow status",
		})
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

// Helper function to save evidence file from base64
func saveEvidenceFile(base64Data string, userID, escrowID uint) (string, error) {
	// Check if it's a base64 string
	if !strings.HasPrefix(base64Data, "data:") {
		// If not base64, assume it's already a URL
		return base64Data, nil
	}

	// Parse base64 data
	parts := strings.Split(base64Data, ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid base64 format")
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	// Determine file extension from mime type
	var ext string
	if strings.Contains(parts[0], "image/jpeg") {
		ext = ".jpg"
	} else if strings.Contains(parts[0], "image/png") {
		ext = ".png"
	} else if strings.Contains(parts[0], "application/pdf") {
		ext = ".pdf"
	} else {
		return "", fmt.Errorf("unsupported file type")
	}

	// Create upload directory
	uploadDir := "./uploads/disputes"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", err
	}

	// Generate filename
	filename := fmt.Sprintf("%d_%d_%d%s", userID, escrowID, time.Now().Unix(), ext)
	filepath := filepath.Join(uploadDir, filename)

	// Save file
	if err := os.WriteFile(filepath, decoded, 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("/uploads/disputes/%s", filename), nil
}