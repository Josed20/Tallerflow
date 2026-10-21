package blob

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const photoURLLifetime = 15 * time.Minute

type Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
}

type Store struct {
	bucket  string
	client  *s3.Client
	presign *s3.PresignClient
}

func NewR2(ctx context.Context, cfg Config) (*Store, error) {
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.User != nil || endpoint.Port() != "" ||
		!strings.HasSuffix(endpoint.Hostname(), ".r2.cloudflarestorage.com") || endpoint.Path != "" || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, errors.New("R2 endpoint must be a Cloudflare HTTPS S3 origin")
	}
	if strings.TrimSpace(cfg.Bucket) == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("R2 bucket and credentials are required")
	}
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load R2 configuration: %w", err)
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = true
	})
	return &Store{bucket: cfg.Bucket, client: client, presign: s3.NewPresignClient(client)}, nil
}

func (s *Store) Put(ctx context.Context, key string, photo []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), Body: bytes.NewReader(photo),
		ContentType: aws.String("image/jpeg"), CacheControl: aws.String("private, no-store"),
	})
	return err
}

func (s *Store) URL(ctx context.Context, key string) (string, error) {
	request, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	}, s3.WithPresignExpires(photoURLLifetime))
	if err != nil {
		return "", err
	}
	return request.URL, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return err
}
