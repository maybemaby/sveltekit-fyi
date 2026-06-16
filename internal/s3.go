package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Config struct {
	Region          string
	Bucket          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

type S3Client struct {
	client *s3.Client
	bucket string
}

func NewS3Client(ctx context.Context, cfg S3Config) (*S3Client, error) {
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("s3 credentials must be provided via S3_KEY and S3_SECRET")
	}

	loadOpts := []func(*config.LoadOptions) error{config.WithRegion(cfg.Region)}

	awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	awsCfg.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""))

	opts := s3.Options{}
	if cfg.Endpoint != "" {
		opts.BaseEndpoint = aws.String(cfg.Endpoint)
	}
	if cfg.UsePathStyle {
		opts.UsePathStyle = true
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = opts.BaseEndpoint
		o.UsePathStyle = opts.UsePathStyle
	})

	return &S3Client{client: client, bucket: cfg.Bucket}, nil
}

func (c *S3Client) UploadImage(ctx context.Context, key string, body []byte) error {
	contentType := mime.TypeByExtension(path.Ext(key))
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPrivate,
	})
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}

	return nil
}

func (c *S3Client) uploadBackup(ctx context.Context, key string, reader io.Reader) error {
	contentType := "application/octet-stream"

	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPrivate,
	})

	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}

	return nil
}
