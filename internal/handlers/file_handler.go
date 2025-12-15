package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"SafeQly/internal/services"
)

var cloudinaryService *services.CloudinaryService


func InitCloudinaryService() error {
	var err error
	cloudinaryService, err = services.NewCloudinaryService()
	if err != nil {
		return fmt.Errorf("failed to initialize Cloudinary service: %w", err)
	}
	return nil
}

// UploadFile handles single file upload
func UploadFile(c *fiber.Ctx) error {
	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file provided",
		})
	}

	// Validate file size (e.g., 10MB max)
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if file.Size > maxSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("File too large. Maximum size is %dMB", maxSize/(1024*1024)),
		})
	}

	// Get folder from query or use default
	folder := c.Query("folder", "safeqly/escrow-files")

	// Upload to Cloudinary
	result, err := cloudinaryService.UploadFile(file, folder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to upload file: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "File uploaded successfully",
		"file": fiber.Map{
			"url":           result.SecureURL,
			"public_id":     result.PublicID,
			"format":        result.Format,
			"resource_type": result.ResourceType,
			"size_bytes":    result.Bytes,
		},
	})
}

// UploadMultipleFiles handles multiple file uploads
func UploadMultipleFiles(c *fiber.Ctx) error {
	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse form",
		})
	}

	// Get files from form
	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No files provided",
		})
	}

	// Validate number of files (e.g., max 5 files)
	maxFiles := 5
	if len(files) > maxFiles {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Too many files. Maximum is %d files", maxFiles),
		})
	}

	// Get folder from query or use default
	folder := c.Query("folder", "safeqly/escrow-files")

	// Upload files
	results, err := cloudinaryService.UploadMultipleFiles(files, folder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to upload files: %v", err),
		})
	}

	// Format response
	uploadedFiles := make([]fiber.Map, 0, len(results))
	for _, result := range results {
		uploadedFiles = append(uploadedFiles, fiber.Map{
			"url":           result.SecureURL,
			"public_id":     result.PublicID,
			"format":        result.Format,
			"resource_type": result.ResourceType,
			"size_bytes":    result.Bytes,
		})
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("%d file(s) uploaded successfully", len(results)),
		"files":   uploadedFiles,
	})
}

// DeleteFile handles file deletion
func DeleteFile(c *fiber.Ctx) error {
	publicID := c.Query("public_id")
	if publicID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "public_id is required",
		})
	}

	err := cloudinaryService.DeleteFile(publicID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete file: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "File deleted successfully",
	})
}