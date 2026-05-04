package pkg

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StoreProvider interface {
	EnsureBucket(ctx context.Context) error
	Put(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error
	DownloadToFile(ctx context.Context, objectKey, destPath string) error
	Bucket() string
}

type minioStore struct {
	client *minio.Client
	bucket string
}

func NewStoreProvider(endpoint, accessKey, secretKey string, useSSL bool, bucket string) (StoreProvider, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("minio endpoint is required")
	}
	if accessKey == "" {
		return nil, fmt.Errorf("minio access key is required")
	}
	if secretKey == "" {
		return nil, fmt.Errorf("minio secret key is required")
	}
	if bucket == "" {
		return nil, fmt.Errorf("minio bucket is required")
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &minioStore{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *minioStore) Bucket() string {
	return s.bucket
}
func (s *minioStore) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		exists, err := s.client.BucketExists(ctx, s.bucket)
		if err == nil && exists {
			return nil
		}
		return err
	}
	return nil
}

func (s *minioStore) Put(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	if objectKey == "" {
		return fmt.Errorf("objectKey is required")
	}
	_, err := s.client.PutObject(ctx, s.bucket, objectKey, reader, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return err
	}
	return nil
}

func (s *minioStore) DownloadToFile(ctx context.Context, objectKey, destPath string) error {
	if objectKey == "" {
		return fmt.Errorf("objectKey is required")
	}
	if destPath == "" {
		return fmt.Errorf("destPath is required")
	}
	if err := s.client.FGetObject(ctx, s.bucket, objectKey, destPath, minio.GetObjectOptions{}); err != nil {
		return err
	}
	return nil
}
