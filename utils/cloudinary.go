package utils

import (
	"context"
	"log"
	"os"

	"path/filepath"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/joho/godotenv"
)

// CloudinaryConfig holds Cloudinary configuration
type CloudinaryConfig struct {
	CloudName    string
	APIKey       string
	APISecret    string
	UploadPreset string
}

// InitCloudinary initializes the Cloudinary client
func InitCloudinary() (*cloudinary.Cloudinary, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file. Using environment variables directly.")
	}

	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"))
	if err != nil {
		return nil, err
	}
	return cld, nil
}

// UploadToCloudinary uploads a file to Cloudinary and returns the secure URL
func UploadToCloudinary(file interface{}, publicID string, folder string) (string, error) {
	cld, err := InitCloudinary()
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	uploadParams := uploader.UploadParams{
		PublicID:     publicID,
		Folder:       folder,
		UploadPreset: os.Getenv("CLOUDINARY_UPLOAD_PRESET"),
	}

	// Apply transformation only for images (not for PDFs)
	fileStr, ok := file.(string)
	if ok {
		ext := filepath.Ext(fileStr)
		if ext != ".pdf" && ext != ".PDF" {
			// Add transformation only for images
			uploadParams.Transformation = "c_thumb,w_200,h_200" // Resize profile pictures
		}
	}

	resp, err := cld.Upload.Upload(ctx, file, uploadParams)
	if err != nil {
		return "", err
	}
	return resp.SecureURL, nil
}
