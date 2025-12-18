package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"SafeQly/internal/database"
	"SafeQly/internal/models"
)

type UpdateProfileRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// GetUserProfile retrieves the authenticated user's profile
func GetUserProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":                user.ID,
			"full_name":         user.FullName,
			"email":             user.Email,
			"phone":             user.Phone,
			"user_tag":          user.UserTag,
			"balance":           user.Balance,
			"escrow_balance":    user.EscrowBalance,
			"avatar":            user.Avatar,
			"avatar_public_id":  user.AvatarPublicID,
			"is_email_verified": user.IsEmailVerified,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
		},
	})
}

// UpdateUserProfile updates user profile information
func UpdateUserProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	req := new(UpdateProfileRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Update fields if provided
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Profile updated successfully",
		"user": fiber.Map{
			"id":         user.ID,
			"full_name":  user.FullName,
			"email":      user.Email,
			"phone":      user.Phone,
			"user_tag":   user.UserTag,
			"balance":    user.Balance,
			"updated_at": user.UpdatedAt,
		},
	})
}

// ChangePassword allows user to change their password
func ChangePassword(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	req := new(ChangePasswordRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Current password is incorrect",
		})
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	// Update password
	user.Password = string(hashedPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to change password",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// UploadAvatar uploads or updates user avatar
func UploadAvatar(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Get file from form
	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Validate file size (5MB max for avatar)
	maxSize := int64(5 * 1024 * 1024)
	if file.Size > maxSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File too large. Maximum size is 5MB",
		})
	}

	// Validate file type (images only)
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/jpg" && contentType != "image/webp" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file type. Only JPEG, PNG, and WebP images are allowed",
		})
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Delete old avatar if exists
	if user.AvatarPublicID != "" {
		if err := cloudinaryService.DeleteFile(user.AvatarPublicID); err != nil {
			// Log error but continue with upload
			fmt.Printf("Failed to delete old avatar: %v\n", err)
		}
	}

	// Upload to Cloudinary
	result, err := cloudinaryService.UploadFile(file, "safeqly/avatars")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to upload avatar: %v", err),
		})
	}

	// Update user avatar
	user.Avatar = result.SecureURL
	user.AvatarPublicID = result.PublicID

	if err := database.DB.Save(&user).Error; err != nil {
		// If database update fails, delete the uploaded file
		cloudinaryService.DeleteFile(result.PublicID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user avatar",
		})
	}

	return c.JSON(fiber.Map{
		"message":   "Avatar uploaded successfully",
		"avatar":    user.Avatar,
		"public_id": user.AvatarPublicID,
	})
}

// DeleteAvatar removes user avatar
func DeleteAvatar(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Check if user has an avatar
	if user.AvatarPublicID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No avatar to delete",
		})
	}

	// Delete from Cloudinary
	if err := cloudinaryService.DeleteFile(user.AvatarPublicID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete avatar: %v", err),
		})
	}

	// Update user record
	user.Avatar = ""
	user.AvatarPublicID = ""

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user record",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Avatar deleted successfully",
	})
}