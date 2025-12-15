package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryService struct {
	cld *cloudinary.Cloudinary
	ctx context.Context
}

func NewCloudinaryService() (*CloudinaryService, error) {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("Cloudinary credentials not set in environment variables")
	}

	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Cloudinary: %w", err)
	}

	return &CloudinaryService{
		cld: cld,
		ctx: context.Background(),
	}, nil
}

type UploadResult struct {
	URL         string `json:"url"`
	SecureURL   string `json:"secure_url"`
	PublicID    string `json:"public_id"`
	Format      string `json:"format"`
	ResourceType string `json:"resource_type"`
	Bytes       int    `json:"bytes"`
}

// UploadFile uploads a file to Cloudinary
func (s *CloudinaryService) UploadFile(file *multipart.FileHeader, folder string) (*UploadResult, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Generate unique filename
	timestamp := time.Now().Unix()
	ext := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%d_%s", timestamp, file.Filename)

	// Upload parameters
	uploadParams := uploader.UploadParams{
		Folder:         folder,
		PublicID:       fileName[:len(fileName)-len(ext)], // Remove extension
		ResourceType:   "auto", // Automatically detect file type
		AllowedFormats: []string{"jpg", "jpeg", "png", "pdf", "doc", "docx"},
	}

	result, err := s.cld.Upload.Upload(s.ctx, src, uploadParams)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to Cloudinary: %w", err)
	}

	return &UploadResult{
		URL:          result.URL,
		SecureURL:    result.SecureURL,
		PublicID:     result.PublicID,
		Format:       result.Format,
		ResourceType: result.ResourceType,
		Bytes:        result.Bytes,
	}, nil
}

// DeleteFile deletes a file from Cloudinary
func (s *CloudinaryService) DeleteFile(publicID string) error {
	_, err := s.cld.Upload.Destroy(s.ctx, uploader.DestroyParams{
		PublicID: publicID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from Cloudinary: %w", err)
	}
	return nil
}

// UploadMultipleFiles uploads multiple files to Cloudinary
func (s *CloudinaryService) UploadMultipleFiles(files []*multipart.FileHeader, folder string) ([]*UploadResult, error) {
	var results []*UploadResult

	for _, file := range files {
		result, err := s.UploadFile(file, folder)
		if err != nil {
			// If one fails, continue with others but log the error
			fmt.Printf("Failed to upload %s: %v\n", file.Filename, err)
			continue
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("all file uploads failed")
	}

	return results, nil
}