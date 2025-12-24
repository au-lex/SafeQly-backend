package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"SafeQly/internal/database"
	"SafeQly/internal/models"
	"SafeQly/internal/services"
)

var notificationService *services.NotificationService


func InitNotificationService() {
	notificationService = services.NewNotificationService()
}

// GetNotifications retrieves all notifications for the authenticated user
func GetNotifications(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	
	// Get query parameters
	limitStr := c.Query("limit", "50")
	offsetStr := c.Query("offset", "0")
	unreadOnly := c.Query("unread_only", "false")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	query := database.DB.Where("user_id = ?", userID)

	// Filter by unread if requested
	if unreadOnly == "true" {
		query = query.Where("is_read = ?", false)
	}

	var notifications []models.Notification
	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve notifications",
		})
	}

	// Get unread count
	var unreadCount int64
	database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&unreadCount)

	return c.JSON(fiber.Map{
		"notifications": notifications,
		"count":         len(notifications),
		"unread_count":  unreadCount,
	})
}

// GetUnreadCount returns the count of unread notifications
func GetUnreadCount(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var unreadCount int64
	if err := database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&unreadCount).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get unread count",
		})
	}

	return c.JSON(fiber.Map{
		"unread_count": unreadCount,
	})
}

// MarkAsRead marks a specific notification as read
func MarkAsRead(c *fiber.Ctx) error {
	notificationID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var notification models.Notification
	if err := database.DB.Where("id = ? AND user_id = ?", notificationID, userID).
		First(&notification).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Notification not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	if !notification.IsRead {
		now := time.Now()
		notification.IsRead = true
		notification.ReadAt = &now

		if err := database.DB.Save(&notification).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to mark notification as read",
			})
		}
	}

	return c.JSON(fiber.Map{
		"message":      "Notification marked as read",
		"notification": notification,
	})
}

// MarkAllAsRead marks all notifications as read for the user
func MarkAllAsRead(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	now := time.Now()
	if err := database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to mark all notifications as read",
		})
	}

	return c.JSON(fiber.Map{
		"message": "All notifications marked as read",
	})
}

// DeleteNotification deletes a specific notification
func DeleteNotification(c *fiber.Ctx) error {
	notificationID := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var notification models.Notification
	if err := database.DB.Where("id = ? AND user_id = ?", notificationID, userID).
		First(&notification).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Notification not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	if err := database.DB.Delete(&notification).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete notification",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Notification deleted successfully",
	})
}

// DeleteAllRead deletes all read notifications for the user
func DeleteAllRead(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	if err := database.DB.Where("user_id = ? AND is_read = ?", userID, true).
		Delete(&models.Notification{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete notifications",
		})
	}

	return c.JSON(fiber.Map{
		"message": "All read notifications deleted successfully",
	})
}