package s3

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3Store struct {
	client    *s3.Client
	bucket    string
	region    string
	endpoint  string
	isDefault bool
}

func New(bucket, region, key, secret, endpoint string, isDefault bool) (*S3Store, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(key, secret, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Configure custom endpoint if provided
	var clientOpts []func(*s3.Options)
	if endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(cfg, clientOpts...)

	return &S3Store{
		client:    client,
		bucket:    bucket,
		region:    region,
		endpoint:  endpoint,
		isDefault: isDefault,
	}, nil
}

func (s *S3Store) Save(content io.Reader, filename string) (string, error) {
	ext := filepath.Ext(filename)
	baseFilename := filename[:len(filename)-len(ext)]
	uniqueFilename := fmt.Sprintf("%s-%s%s", baseFilename, uuid.New().String(), ext)
	storagePath := filepath.Join(time.Now().Format("2006/01/02"), uniqueFilename)

	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(storagePath),
		Body:   content,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	return storagePath, nil
}

func (s *S3Store) Get(path string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	return result.Body, nil
}

func (s *S3Store) Delete(path string) error {
	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	return err
}

func (s *S3Store) GetURL(path string) string {
	if s.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, path)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, path)
}

func (s *S3Store) GetSize(path string) (int64, error) {
	result, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get object head from S3: %w", err)
	}
	if result.ContentLength == nil {
		return 0, fmt.Errorf("content length is nil")
	}
	return *result.ContentLength, nil
}

func (s *S3Store) SetExpiry(path string, expiry time.Time) error {
	_, err := s.client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		CopySource: aws.String(fmt.Sprintf("%s/%s", s.bucket, path)),
		Key:        aws.String(path),
		Expires:    aws.Time(expiry),
	})
	return err
}

func (s *S3Store) SetDefault() error {
	s.isDefault = true
	return nil
}

func (s *S3Store) IsDefault() bool {
	return s.isDefault
}

func (s *S3Store) Type() string {
	return "s3"
}
