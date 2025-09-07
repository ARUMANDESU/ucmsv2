package s3

import (
	"context"
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"

	"gitlab.com/ucmsv2/ucms-backend/pkg/errorx"
)

type Client struct {
	s3Client *s3.Client
	bucket   string
}

func NewClient(ctx context.Context, endpoint, accessKey, secretKey, bucket, region string) (*Client, error) {
	const op = "s3.NewClient"
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion(region),
		config.WithBaseEndpoint(endpoint),
	)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	return &Client{
		s3Client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true // Required for MinIO
		}),
		bucket: bucket,
	}, nil
}

func (c *Client) UploadFile(ctx context.Context, key string, file io.Reader, contentType string) error {
	const op = "s3.Client.UploadFile"
	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
		// Optional: Set cache headers, metadata, etc.
		CacheControl: aws.String("max-age=604800"), // 1 week
	})
	return errorx.Wrap(err, op)
}

func (c *Client) DeleteFile(ctx context.Context, key string) error {
	const op = "s3.Client.DeleteFile"
	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return errorx.Wrap(err, op)
}

func (c *Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	const op = "s3.Client.GetObject"
	output, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}
	defer func() {
		cerr := output.Body.Close()
		if err != nil {
			slog.Warn("failed to close S3 object body", slog.String("error", cerr.Error()))
		}
	}()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, errorx.Wrap(err, op)
	}

	return data, nil
}

func (c *Client) CreateBucket(ctx context.Context) error {
	const op = "s3.CreateBucket"
	_, err := c.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return errorx.Wrap(err, op)
	}
	return nil
}

func (c *Client) Bucket() string {
	return c.bucket
}
