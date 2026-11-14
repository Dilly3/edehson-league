package membership

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ReceiptStore keeps uploaded payment receipts. The database stores only the key.
type ReceiptStore interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

var receipts ReceiptStore = localStore{dir: filepath.Join(os.TempDir(), "edehson-receipts")}

// SetReceiptStore installs the store used for receipts (call once at startup).
func SetReceiptStore(s ReceiptStore) { receipts = s }

// NewReceiptStoreFromEnv returns an S3 store when S3_BUCKET is set, otherwise a
// local-disk store for development.
//
//	S3_BUCKET    bucket name (required for S3)
//	S3_REGION    region, e.g. eu-west-2 (falls back to AWS_REGION)
//	S3_ENDPOINT  optional, for S3-compatible storage (Cloudflare R2, MinIO)
//	S3_PREFIX    optional key prefix, default "receipts/"
//
// Credentials come from the standard AWS chain (AWS_ACCESS_KEY_ID /
// AWS_SECRET_ACCESS_KEY, shared config, or an instance/task role).
func NewReceiptStoreFromEnv(ctx context.Context) (ReceiptStore, error) {
	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	if bucket == "" {
		dir := os.Getenv("RECEIPTS_DIR")
		if dir == "" {
			dir = filepath.Join(os.TempDir(), "edehson-receipts")
		}
		log.Printf("S3_BUCKET not set: storing payment receipts on local disk in %s (development only)", dir)
		return localStore{dir: dir}, nil
	}
	opts := []func(*awsconfig.LoadOptions) error{}
	if region := os.Getenv("S3_REGION"); region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	endpoint := strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})
	prefix := os.Getenv("S3_PREFIX")
	if prefix == "" {
		prefix = "receipts/"
	}
	log.Printf("Storing payment receipts in S3 bucket %s", bucket)
	return s3Store{client: client, bucket: bucket, prefix: prefix}, nil
}

type s3Store struct {
	client *s3.Client
	bucket string
	prefix string
}

func (s s3Store) Put(ctx context.Context, key, contentType string, data []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(s.prefix + key),
		Body: bytes.NewReader(data), ContentType: aws.String(contentType), ContentLength: aws.Int64(int64(len(data))),
	})
	return err
}

func (s s3Store) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(s.prefix + key)})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(io.LimitReader(out.Body, maxReceiptBytes+1))
}

func (s s3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(s.prefix + key)})
	return err
}

type localStore struct{ dir string }

func (l localStore) path(key string) (string, error) {
	clean := filepath.Clean("/" + key)
	if strings.Contains(clean, "..") {
		return "", errors.New("invalid receipt key")
	}
	return filepath.Join(l.dir, clean), nil
}

func (l localStore) Put(_ context.Context, key, _ string, data []byte) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o640)
}

func (l localStore) Get(_ context.Context, key string) ([]byte, error) {
	p, err := l.path(key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

func (l localStore) Delete(_ context.Context, key string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	return os.Remove(p)
}

var slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)

// receiptKey files receipts under the player's name, e.g.
// "chukwuemeka-okafor/2026-11/<request-id>.jpg".
func receiptKey(memberName, requestID, month, contentType string) string {
	slug := strings.Trim(slugUnsafe.ReplaceAllString(strings.ToLower(memberName), "-"), "-")
	if slug == "" {
		slug = "player"
	}
	if len(slug) > 60 {
		slug = slug[:60]
	}
	ext := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "application/pdf": ".pdf"}[contentType]
	return slug + "/" + month + "/" + requestID + ext
}
