// Package objectstore provides S3-compatible object storage access.
package objectstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Client struct {
	client         *s3.S3
	bucket         string
	publicEndpoint string
}

type Config struct {
	Endpoint       string
	PublicEndpoint string
	Bucket         string
	Region         string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
}

func NewClient(cfg Config) (*Client, error) {
	awsCfg := &aws.Config{
		Region:           aws.String(cfg.Region),
		Endpoint:         aws.String(cfg.Endpoint),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
		DisableSSL:       aws.Bool(!cfg.UseSSL),
	}
	sess, err := session.NewSession(awsCfg)
	if err != nil {
		return nil, fmt.Errorf("create s3 session: %w", err)
	}
	return &Client{client: s3.New(sess), bucket: cfg.Bucket, publicEndpoint: cfg.PublicEndpoint}, nil
}

func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error {
	ctx, span := otel.Tracer("avatars-s3").Start(ctx, "s3.put_object")
	defer span.End()
	span.SetAttributes(attribute.String("s3.key", key), attribute.String("s3.operation", "put"))
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read upload body: %w", err)
	}
	_, err = c.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("upload object: %w", err)
	}
	return nil
}

func (c *Client) Download(ctx context.Context, key string) ([]byte, error) {
	ctx, span := otel.Tracer("avatars-s3").Start(ctx, "s3.get_object")
	defer span.End()
	span.SetAttributes(attribute.String("s3.key", key), attribute.String("s3.operation", "get"))
	out, err := c.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("download object: %w", err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read object: %w", err)
	}
	return data, nil
}

func (c *Client) Delete(ctx context.Context, keys ...string) error {
	ctx, span := otel.Tracer("avatars-s3").Start(ctx, "s3.delete_object")
	defer span.End()
	span.SetAttributes(attribute.String("s3.operation", "delete"))
	for _, key := range keys {
		if key == "" {
			continue
		}
		_, err := c.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return fmt.Errorf("delete object %s: %w", key, err)
		}
	}
	return nil
}

func (c *Client) PresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if key == "" {
		return "", nil
	}
	req, _ := c.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	url, err := req.Presign(expiry)
	if err != nil {
		return "", fmt.Errorf("presign url: %w", err)
	}
	if c.publicEndpoint != "" {
		url = replaceURLHost(url, c.publicEndpoint)
	}
	_ = ctx
	return url, nil
}

func replaceURLHost(raw, publicEndpoint string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	pub, err := url.Parse(publicEndpoint)
	if err != nil {
		return raw
	}
	parsed.Scheme = pub.Scheme
	parsed.Host = pub.Host
	return parsed.String()
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	_, err := c.client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err == nil {
		return nil
	}
	_, err = c.client.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}
	return nil
}
