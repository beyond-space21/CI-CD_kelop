package storage

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var S3Client *s3.Client
var BucketName string
var Region string
var Endpoint string

func InitStorage() {
	accessKey := os.Getenv("R2_SPACES_ACCESS_KEY")
	secretKey := os.Getenv("R2_SPACES_SECRET_KEY")
	BucketName = os.Getenv("R2_SPACES_BUCKET")
	Region = os.Getenv("R2_SPACES_REGION")
	Endpoint = os.Getenv("R2_SPACES_ENDPOINT")

	if accessKey == "" || secretKey == "" || BucketName == "" || Region == "" || Endpoint == "" {
		panic("Missing required Cloudflare R2 environment variables")
	}

	// Normalize endpoint - remove trailing slash and ensure proper format
	endpoint := Endpoint
	if len(endpoint) > 0 && endpoint[len(endpoint)-1] == '/' {
		endpoint = endpoint[:len(endpoint)-1]
	}

	// Create AWS config with custom endpoint for Cloudflare R2
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to load AWS config: %v", err))
	}

	// Create S3 client with custom endpoint for Cloudflare R2
	S3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	// Update global Endpoint variable with normalized value
	Endpoint = endpoint

	fmt.Printf("Cloudflare R2 initialized! Endpoint: %s, Region: %s, Bucket: %s\n", Endpoint, Region, BucketName)
}

// GeneratePresignedUploadURL generates a presigned URL for uploading a file to Cloudflare R2
// Returns the presigned URL and any error that occurred
func GeneratePresignedUploadURL(objectKey string, expiration time.Duration) (string, error) {
	if S3Client == nil {
		return "", fmt.Errorf("storage client not initialized. Call InitStorage() first")
	}

	// Create presign client - it automatically inherits configuration from S3Client
	presignClient := s3.NewPresignClient(S3Client)

	request, err := presignClient.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})

	if err != nil {
		fmt.Printf("Presign error: %v\n", err)
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}


// GeneratePresignedGetURL generates a presigned URL for downloading a file from Cloudflare R2
// Returns the presigned URL and any error that occurred
func GeneratePresignedGetURL(objectKey string, expiration time.Duration) (string, error) {
	if S3Client == nil {
		return "", fmt.Errorf("storage client not initialized. Call InitStorage() first")
	}

	// Create presign client - it automatically inherits configuration from S3Client
	presignClient := s3.NewPresignClient(S3Client)

	request, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

func IsFileExists(objectKey string) (bool, error) {
	if S3Client == nil {
		return false, fmt.Errorf("storage client not initialized. Call InitStorage() first")
	}

	_, err := S3Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return false, fmt.Errorf("failed to check if file exists: %w", err)
	}

	return true, nil
}

// DeleteFile deletes a file from Cloudflare R2 storage
// Returns an error if the deletion fails
func DeleteFile(ctx context.Context, objectKey string) error {
	if S3Client == nil {
		return fmt.Errorf("storage client not initialized. Call InitStorage() first")
	}

	if objectKey == "" {
		return fmt.Errorf("object key cannot be empty")
	}

	fmt.Printf("DeleteFile: attempting to delete object from bucket %s: %s\n", BucketName, objectKey)

	result, err := S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		fmt.Printf("DeleteFile ERROR for %s in bucket %s: %v\n", objectKey, BucketName, err)
		return fmt.Errorf("failed to delete file %s from bucket %s: %w", objectKey, BucketName, err)
	}

	// Log deletion result (result.DeleteMarker indicates if a delete marker was created)
	if result.DeleteMarker != nil {
		fmt.Printf("DeleteFile: delete marker created for %s (versioned bucket)\n", objectKey)
	}

	fmt.Printf("DeleteFile: successfully deleted file from R2 bucket %s: %s\n", BucketName, objectKey)
	return nil
}
