package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Store handles Cloudflare R2 storage operations for recordings
// R2 is S3-compatible but with global edge caching via Cloudflare CDN
type R2Store struct {
	client    *s3.Client
	bucket    string
	prefix    string
	accountID string
	publicURL string // Custom domain for public access (e.g., "https://recordings.rexec.io")
}

// R2Config holds Cloudflare R2 configuration
type R2Config struct {
	AccountID       string // Cloudflare account ID
	AccessKeyID     string // R2 access key ID
	SecretAccessKey string // R2 secret access key
	Bucket          string // R2 bucket name
	Prefix          string // Optional prefix for all keys (e.g., "recordings/")
	PublicURL       string // Public URL for accessing files (custom domain or R2.dev URL)
}

// NewR2Store creates a new Cloudflare R2 store
func NewR2Store(cfg R2Config) (*R2Store, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("R2 bucket is required")
	}
	if cfg.AccountID == "" {
		return nil, fmt.Errorf("Cloudflare account ID is required")
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("R2 access credentials are required")
	}

	// R2 endpoint format: https://<account_id>.r2.cloudflarestorage.com
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	// Build AWS config with static credentials
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("auto"), // R2 uses "auto" region
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load R2 config: %w", err)
	}

	// Create S3 client with R2 endpoint
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // R2 requires path-style addressing
	})

	store := &R2Store{
		client:    client,
		bucket:    cfg.Bucket,
		prefix:    cfg.Prefix,
		accountID: cfg.AccountID,
		publicURL: strings.TrimSuffix(cfg.PublicURL, "/"),
	}

	// Verify bucket access
	if err := store.verifyBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to verify R2 bucket access: %w", err)
	}

	log.Printf("[R2] Connected to bucket: %s (account: %s, public URL: %s)", cfg.Bucket, cfg.AccountID, cfg.PublicURL)
	return store, nil
}

// NewR2StoreFromEnv creates a new R2 store from environment variables
func NewR2StoreFromEnv() (*R2Store, error) {
	cfg := R2Config{
		AccountID:       os.Getenv("CF_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("CF_R2_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("CF_R2_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("CF_R2_BUCKET"),
		Prefix:          os.Getenv("CF_R2_PREFIX"),
		PublicURL:       os.Getenv("CF_R2_PUBLIC_URL"),
	}

	// Default prefix if not set
	if cfg.Prefix == "" {
		cfg.Prefix = "recordings/"
	}

	return NewR2Store(cfg)
}

// verifyBucket checks if we can access the bucket
func (r *R2Store) verifyBucket(ctx context.Context) error {
	_, err := r.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(r.bucket),
	})
	return err
}

// key generates the R2 key for a recording
func (r *R2Store) key(recordingID string) string {
	return r.prefix + recordingID + ".cast"
}

// GetPublicURL returns the public CDN URL for a recording
func (r *R2Store) GetPublicURL(recordingID string) string {
	if r.publicURL == "" {
		// Fallback to R2.dev public URL if no custom domain
		// Note: R2.dev URLs require public access to be enabled on the bucket
		return fmt.Sprintf("https://%s.r2.dev/%s", r.bucket, r.key(recordingID))
	}
	return fmt.Sprintf("%s/%s", r.publicURL, r.key(recordingID))
}

// PutRecording uploads recording data to R2 and returns the public URL
func (r *R2Store) PutRecording(ctx context.Context, recordingID string, data []byte) (string, error) {
	key := r.key(recordingID)

	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(r.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String("application/x-asciicast"),
		CacheControl: aws.String("public, max-age=31536000, immutable"), // Cache for 1 year (recordings are immutable)
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload recording to R2: %w", err)
	}

	publicURL := r.GetPublicURL(recordingID)
	log.Printf("[R2] Uploaded recording %s (%d bytes) -> %s", recordingID, len(data), publicURL)
	return publicURL, nil
}

// PutRecordingWithMetadata uploads recording data with custom metadata
func (r *R2Store) PutRecordingWithMetadata(ctx context.Context, recordingID string, data []byte, metadata map[string]string) (string, error) {
	key := r.key(recordingID)

	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(r.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String("application/x-asciicast"),
		CacheControl: aws.String("public, max-age=31536000, immutable"),
		Metadata:     metadata,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload recording to R2: %w", err)
	}

	publicURL := r.GetPublicURL(recordingID)
	log.Printf("[R2] Uploaded recording %s with metadata (%d bytes) -> %s", recordingID, len(data), publicURL)
	return publicURL, nil
}

// GetRecording downloads recording data from R2
func (r *R2Store) GetRecording(ctx context.Context, recordingID string) ([]byte, error) {
	key := r.key(recordingID)

	result, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get recording from R2: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read recording data: %w", err)
	}

	return data, nil
}

// DeleteRecording deletes recording data from R2
func (r *R2Store) DeleteRecording(ctx context.Context, recordingID string) error {
	key := r.key(recordingID)

	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete recording from R2: %w", err)
	}

	log.Printf("[R2] Deleted recording %s", recordingID)
	return nil
}

// RecordingExists checks if a recording exists in R2
func (r *R2Store) RecordingExists(ctx context.Context, recordingID string) (bool, error) {
	key := r.key(recordingID)

	_, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		if httpErr, ok := err.(interface{ HTTPStatusCode() int }); ok && httpErr.HTTPStatusCode() == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetPresignedURL generates a presigned URL for private access to a recording
// This is useful if you want to keep the bucket private but still allow temporary access
func (r *R2Store) GetPresignedURL(ctx context.Context, recordingID string, expiry time.Duration) (string, error) {
	key := r.key(recordingID)

	presignClient := s3.NewPresignClient(r.client)

	result, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:              aws.String(r.bucket),
		Key:                 aws.String(key),
		ResponseContentType: aws.String("application/x-asciicast"),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return result.URL, nil
}

// ListRecordings lists all recordings with the configured prefix
func (r *R2Store) ListRecordings(ctx context.Context, maxKeys int32) ([]string, error) {
	if maxKeys <= 0 {
		maxKeys = 100
	}

	result, err := r.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(r.bucket),
		Prefix:  aws.String(r.prefix),
		MaxKeys: aws.Int32(maxKeys),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list recordings: %w", err)
	}

	var recordings []string
	for _, obj := range result.Contents {
		key := aws.ToString(obj.Key)
		// Extract recording ID from key (remove prefix and .cast extension)
		recordingID := strings.TrimPrefix(key, r.prefix)
		recordingID = strings.TrimSuffix(recordingID, ".cast")
		recordings = append(recordings, recordingID)
	}

	return recordings, nil
}

// GetRecordingInfo returns metadata about a recording
func (r *R2Store) GetRecordingInfo(ctx context.Context, recordingID string) (*RecordingInfo, error) {
	key := r.key(recordingID)

	result, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get recording info: %w", err)
	}

	return &RecordingInfo{
		ID:           recordingID,
		Size:         aws.ToInt64(result.ContentLength),
		LastModified: aws.ToTime(result.LastModified),
		ContentType:  aws.ToString(result.ContentType),
		ETag:         aws.ToString(result.ETag),
		Metadata:     result.Metadata,
		PublicURL:    r.GetPublicURL(recordingID),
	}, nil
}

// RecordingInfo contains metadata about a recording stored in R2
type RecordingInfo struct {
	ID           string
	Size         int64
	LastModified time.Time
	ContentType  string
	ETag         string
	Metadata     map[string]string
	PublicURL    string
}

// CopyRecording copies a recording to a new ID (useful for creating backups)
func (r *R2Store) CopyRecording(ctx context.Context, sourceID, destID string) (string, error) {
	sourceKey := r.key(sourceID)
	destKey := r.key(destID)

	_, err := r.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(r.bucket),
		CopySource: aws.String(fmt.Sprintf("%s/%s", r.bucket, sourceKey)),
		Key:        aws.String(destKey),
	})
	if err != nil {
		return "", fmt.Errorf("failed to copy recording: %w", err)
	}

	publicURL := r.GetPublicURL(destID)
	log.Printf("[R2] Copied recording %s -> %s", sourceID, destID)
	return publicURL, nil
}

// Close closes the R2 store (no-op for R2)
func (r *R2Store) Close() error {
	return nil
}

// GetBucket returns the bucket name
func (r *R2Store) GetBucket() string {
	return r.bucket
}

// GetPrefix returns the prefix
func (r *R2Store) GetPrefix() string {
	return r.prefix
}

// IsConfigured returns true if R2 environment variables are set
func IsR2Configured() bool {
	return os.Getenv("CF_ACCOUNT_ID") != "" &&
		os.Getenv("CF_R2_ACCESS_KEY_ID") != "" &&
		os.Getenv("CF_R2_SECRET_ACCESS_KEY") != "" &&
		os.Getenv("CF_R2_BUCKET") != ""
}
